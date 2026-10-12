package credential

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"github.com/validflow/ccneed-cli/internal/api"
)

var proofPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43,128}$`)
var codePattern = regexp.MustCompile(`^[A-HJ-NP-Z2-9]{10}$`)

// Pending 保存当前 origin 的短期授权请求；秘密不进入公开 JSON。
type Pending struct {
	Origin     string    `json:"origin"`
	UserCode   string    `json:"user_code"`
	ExpiresAt  time.Time `json:"expires_at"`
	NextPollAt time.Time `json:"next_poll_at"`
	Interval   int       `json:"interval"`
	DeviceCode string    `json:"-"`
	Verifier   string    `json:"-"`
}

// pendingWire 仅用于系统凭证库，不用于 stdout 或命令参数。
type pendingWire struct {
	Pending
	Version    int    `json:"version"`
	DeviceCode string `json:"device_code"`
	Verifier   string `json:"verifier"`
}

// SavePending 将请求保存在独立的 origin 键，不替换已经登录的凭证。
//
// Args:
//
//	ctx: 本地取消上下文。
//	origin: 用户明确选择的服务。
//	pending: 最多十分钟有效的授权证明。
//
// Returns:
//
//	保存错误，或 nil。
//
// Errors:
//
//	请求无效、过期、错环境或系统存储不可用时失败关闭。
func (v *Vault) SavePending(ctx context.Context, origin string, pending Pending) error {
	canonical, account, err := v.location(ctx, origin)
	if err != nil {
		return err
	}
	if !validPending(pending, canonical, v.allowInsecureHTTP) || pending.ExpiresAt.After(v.now().Add(10*time.Minute)) {
		return &api.Error{Code: "invalid_auth_request"}
	}
	if !pending.ExpiresAt.After(v.now()) {
		return &api.Error{Code: "auth_request_expired"}
	}
	pending.Origin = canonical
	raw, err := json.Marshal(pendingWire{Pending: pending, Version: 1, DeviceCode: pending.DeviceCode, Verifier: pending.Verifier})
	if err != nil || len(raw) > maxRecordBytes {
		return &api.Error{Code: "invalid_auth_request"}
	}
	if err := v.backend.Set(keyringService, "pending."+account, string(raw)); err != nil {
		return &api.Error{Code: "credential_store_unavailable"}
	}
	return nil
}

// LoadPending 读取当前 origin 的短期请求，不向其他环境回退。
//
// Args:
//
//	ctx: 可取消的读取上下文。
//	origin: 当前服务地址。
//
// Returns:
//
//	内部授权请求，或安全错误。
//
// Errors:
//
//	缺失、损坏、过期或系统读取失败时不发起网络请求。
func (v *Vault) LoadPending(ctx context.Context, origin string) (Pending, error) {
	canonical, account, err := v.location(ctx, origin)
	if err != nil {
		return Pending{}, err
	}
	raw, err := v.lookup(ctx, "pending."+account)
	if errors.Is(err, ErrMissing) {
		return Pending{}, &api.Error{Code: "auth_request_missing"}
	}
	var safe *api.Error
	if errors.As(err, &safe) || ctx.Err() != nil {
		return Pending{}, err
	}
	if err != nil {
		return Pending{}, &api.Error{Code: "credential_store_unavailable"}
	}
	var wire pendingWire
	if len(raw) > maxRecordBytes || json.Unmarshal([]byte(raw), &wire) != nil || wire.Version != 1 {
		return Pending{}, &api.Error{Code: "auth_request_corrupt"}
	}
	pending := wire.Pending
	pending.DeviceCode, pending.Verifier = wire.DeviceCode, wire.Verifier
	if !validPending(pending, canonical, v.allowInsecureHTTP) {
		return Pending{}, &api.Error{Code: "auth_request_corrupt"}
	}
	if !pending.ExpiresAt.After(v.now()) {
		return Pending{}, &api.Error{Code: "auth_request_expired"}
	}
	return pending, nil
}

// ForgetPending 删除当前 origin 的短期请求，不影响其他服务或现有凭证。
//
// Args:
//
//	ctx: 本地取消上下文。
//	origin: 当前服务地址。
//
// Returns:
//
//	删除错误，或 nil。
//
// Errors:
//
//	系统存储失败时返回脱敏错误；缺失幂等成功。
func (v *Vault) ForgetPending(ctx context.Context, origin string) error {
	_, account, err := v.location(ctx, origin)
	if err != nil {
		return err
	}
	if err := v.backend.Delete(keyringService, "pending."+account); err != nil && !errors.Is(err, ErrMissing) {
		return &api.Error{Code: "credential_store_unavailable"}
	}
	return nil
}

// validPending 校验公开确认码与秘密格式，不信任系统库中的序列化内容。
//
// Args:
//
//	pending: 待校验请求。
//	origin: 当前规范化地址。
//
// Returns:
//
//	所有字段符合短期请求契约时为 true。
func validPending(pending Pending, origin string, allowInsecureHTTP bool) bool {
	parse := api.ParseOrigin
	if allowInsecureHTTP {
		parse = api.ParseOriginWithInsecureHTTP
	}
	bound, err := parse(pending.Origin)
	return err == nil && bound.String() == origin && proofPattern.MatchString(pending.DeviceCode) &&
		proofPattern.MatchString(pending.Verifier) && codePattern.MatchString(pending.UserCode) &&
		!pending.ExpiresAt.IsZero() && !pending.NextPollAt.IsZero() && pending.Interval >= 5 && pending.Interval <= 60
}
