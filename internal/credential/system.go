package credential

import (
	"errors"

	"github.com/zalando/go-keyring"
)

// SystemBackend 使用操作系统凭证库，不退回明文文件。
type SystemBackend struct{}

// NewSystem 创建当前用户的系统凭证库适配器；不主动解锁或写入数据。
//
// Returns:
//
//	Windows Credential Manager、macOS Keychain 或 Linux Secret Service 适配器。
func NewSystem() *SystemBackend { return &SystemBackend{} }

// Get 读取一个指定的系统凭证库项，不输出秘密。
//
// Args:
//
//	service: 固定的产品命名空间。
//	account: 规范化 origin 的摘要键。
//
// Returns:
//
//	内部凭证载荷，或错误。
//
// Errors:
//
//	不存在返回 ErrMissing，系统不可用返回不含秘密的错误。
func (b *SystemBackend) Get(service, account string) (string, error) {
	value, err := keyring.Get(service, account)
	if err != nil {
		return "", systemError(err)
	}
	return value, nil
}

// Set 写入一个指定的系统凭证库项，不在进程参数中传递秘密。
//
// Args:
//
//	service: 固定的产品命名空间。
//	account: 规范化 origin 的摘要键。
//	secret: 只在内存和系统凭证库存在的内部载荷。
//
// Returns:
//
//	系统写入错误，或 nil。
//
// Errors:
//
//	系统不可用、锁定或拒绝写入时失败，不提供不安全回退。
func (b *SystemBackend) Set(service, account, secret string) error {
	return systemError(keyring.Set(service, account, secret))
}

// Delete 只删除指定项，不调用系统凭证库的 DeleteAll。
//
// Args:
//
//	service: 固定的产品命名空间。
//	account: 规范化 origin 的摘要键。
//
// Returns:
//
//	系统删除错误，或 nil。
//
// Errors:
//
//	不存在返回 ErrMissing，系统不可用返回安全错误。
func (b *SystemBackend) Delete(service, account string) error {
	return systemError(keyring.Delete(service, account))
}

// systemError 不向上暴露可能包含秘密的系统原始错误。
//
// Args:
//
//	err: 系统凭证库调用结果。
//
// Returns:
//
//	不含秘密的错误，或 nil。
//
// Errors:
//
//	不存在映射为 ErrMissing，其余错误归为系统不可用。
func systemError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrMissing
	}
	return errors.New("system credential store unavailable")
}
