package credential

import (
	"context"
	"time"

	"github.com/validflow/ccneed-cli/internal/api"
)

const lookupTimeout = 30 * time.Second

// lookupResult 是只读系统调用的内部返回，不用于命令输出。
type lookupResult struct {
	value string
	err   error
}

// lookup 在系统读取期间监听取消与期限，每个 Vault 最多启动一个未完成读取。
// 取消只结束本地等待，不能保证终止原生解锁提示；不将此机制用于 Set/Delete。
//
// Args:
//
//	ctx: 调用方取消上下文。
//	account: 规范化 origin 的摘要键。
//
// Returns:
//
//	内部凭证载荷，或安全错误。
//
// Errors:
//
//	取消返回原上下文错误；超过 30 秒返回 credential_store_timeout；系统错误由 load 脱敏。
func (v *Vault) lookup(ctx context.Context, account string) (string, error) {
	lookupContext, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()
	select {
	case v.readSlot <- struct{}{}:
	case <-lookupContext.Done():
		return "", lookupCancellation(ctx)
	}
	result := make(chan lookupResult, 1)
	go func() {
		defer func() { <-v.readSlot }()
		value, err := v.backend.Get(keyringService, account)
		result <- lookupResult{value: value, err: err}
	}()
	select {
	case reply := <-result:
		if err := ctx.Err(); err != nil {
			return "", err
		}
		return reply.value, reply.err
	case <-lookupContext.Done():
		return "", lookupCancellation(ctx)
	}
}

// lookupCancellation 区分调用方取消与凭证库读取自身的等待期限。
//
// Args:
//
//	ctx: 原始调用方上下文。
//
// Returns:
//
//	不含系统秘密的取消或超时错误。
//
// Errors:
//
//	保留原始 context 错误；否则返回 credential_store_timeout。
func lookupCancellation(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return &api.Error{Code: "credential_store_timeout"}
}
