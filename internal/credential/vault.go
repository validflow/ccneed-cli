// Package credential 负责按服务 origin 隔离的系统凭证存储，不读取网站 Cookie。
package credential

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/validflow/ccneed-cli/internal/api"
)

// ErrMissing 标识当前 origin 没有本地凭证，不与凭证库不可用混淆。
var ErrMissing = errors.New("credential missing")

// Backend 是系统凭证库边界；实现不能将秘密写入日志或进程参数。
type Backend interface {
	Get(service, account string) (string, error)
	Set(service, account, secret string) error
	Delete(service, account string) error
}

// Record 是已授权的凭证；Token 禁止通过普通 JSON 输出。
type Record struct {
	Origin       string    `json:"origin"`
	CredentialID string    `json:"credential_id"`
	UserID       string    `json:"user_id"`
	ExpiresAt    time.Time `json:"expires_at"`
	Token        string    `json:"-"`
}

// Status 描述本地凭证状态，不声称服务端鉴权或撤销检查已成功。
type Status struct {
	Origin       string     `json:"origin"`
	State        string     `json:"state"`
	CredentialID string     `json:"credential_id,omitempty"`
	UserID       string     `json:"user_id,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

// Vault 将一个 origin 的凭证交给系统凭证库，不使用明文文件回退。
type Vault struct {
	backend           Backend
	now               func() time.Time
	readSlot          chan struct{}
	allowInsecureHTTP bool
}

// New 创建显式注入系统凭证库与时钟的存储。
//
// Args:
//
//	backend: 系统凭证库，不可为 nil。
//	now: 当前时间；nil 使用 time.Now。
//
// Returns:
//
//	按 origin 隔离的凭证存储。
func New(backend Backend, now func() time.Time) *Vault {
	if now == nil {
		now = time.Now
	}
	return &Vault{backend: backend, now: now, readSlot: make(chan struct{}, 1)}
}

// WithInsecureHTTP returns a per-invocation view for explicitly selected HTTP staging.
// It shares the same origin-keyed backend without changing the default vault policy.
//
// Returns:
//
//	A policy-scoped vault, or nil when the receiver is nil.
func (v *Vault) WithInsecureHTTP() *Vault {
	if v == nil {
		return nil
	}
	copy := *v
	copy.allowInsecureHTTP = true
	return &copy
}

// Save 保存已授权且未过期的凭证，不允许静默改写所属 origin。
//
// Args:
//
//	ctx: 本地取消上下文。
//	origin: 用户显式选择的服务 origin。
//	record: 服务端授权结果。
//
// Returns:
//
//	保存错误，或 nil。
//
// Errors:
//
//	不合法、过期、错环境或系统凭证库不可用时失败关闭。
func (v *Vault) Save(ctx context.Context, origin string, record Record) error {
	canonical, account, err := v.location(ctx, origin)
	if err != nil {
		return err
	}
	if err := validateRecord(record, canonical, v.allowInsecureHTTP); err != nil {
		return err
	}
	if !record.ExpiresAt.After(v.now()) {
		return &api.Error{Code: "credential_expired"}
	}
	record.Origin = canonical
	encoded, err := encodeRecord(record)
	if err != nil {
		return err
	}
	if err := v.backend.Set(keyringService, account, encoded); err != nil {
		return &api.Error{Code: "credential_store_unavailable"}
	}
	return nil
}

// Connect 只使用当前 origin 未过期的凭证构造 API client。
//
// Args:
//
//	ctx: 本地取消上下文。
//	origin: 用户显式选择的服务 origin。
//
// Returns:
//
//	已绑定 API client，或安全错误。
//
// Errors:
//
//	凭证缺失、过期、损坏或系统凭证库不可用时不回退其他环境。
func (v *Vault) Connect(ctx context.Context, origin string) (*api.Client, error) {
	record, _, err := v.load(ctx, origin)
	if err != nil {
		return nil, err
	}
	if !record.ExpiresAt.After(v.now()) {
		return nil, &api.Error{Code: "credential_expired"}
	}
	if v.allowInsecureHTTP {
		return api.NewWithInsecureHTTP(record.Origin, record.Token, nil)
	}
	return api.New(record.Origin, record.Token, nil)
}

// Inspect 查询当前 origin 的本地状态，不发出网络请求。
//
// Args:
//
//	ctx: 本地取消上下文。
//	origin: 用户显式选择的服务 origin。
//
// Returns:
//
//	不包含秘密的本地状态，或安全错误。
//
// Errors:
//
//	地址无效、数据损坏或系统凭证库不可用时返回错误，不谎报未登录。
func (v *Vault) Inspect(ctx context.Context, origin string) (Status, error) {
	record, canonical, err := v.load(ctx, origin)
	status := Status{Origin: canonical, State: "missing"}
	var apiError *api.Error
	if errors.As(err, &apiError) && apiError.Code == "not_authenticated" {
		return status, nil
	}
	if err != nil {
		return Status{}, err
	}
	status.State = "present"
	if !record.ExpiresAt.After(v.now()) {
		status.State = "expired"
	}
	status.CredentialID, status.UserID, status.ExpiresAt = record.CredentialID, record.UserID, &record.ExpiresAt
	return status, nil
}

// Forget 只删除当前 origin 的本地凭证；不代表服务端已经撤销。
//
// Args:
//
//	ctx: 本地取消上下文。
//	origin: 用户显式选择的服务 origin。
//
// Returns:
//
//	删除错误，或 nil。
//
// Errors:
//
//	地址无效或系统凭证库不可用时失败；已无凭证时幂等成功。
func (v *Vault) Forget(ctx context.Context, origin string) error {
	_, account, err := v.location(ctx, origin)
	if err != nil {
		return err
	}
	if err := v.backend.Delete(keyringService, account); err != nil && !errors.Is(err, ErrMissing) {
		return &api.Error{Code: "credential_store_unavailable"}
	}
	return nil
}

// load 读取当前 origin 的版本化记录，不将过期凭证当作其他环境可复用凭证。
//
// Args:
//
//	ctx: 本地取消上下文。
//	origin: 当前选择的服务 origin。
//
// Returns:
//
//	记录、规范化 origin，或安全错误。
//
// Errors:
//
//	凭证不存在、损坏、错环境或系统凭证库不可用时失败关闭。
func (v *Vault) load(ctx context.Context, origin string) (Record, string, error) {
	canonical, account, err := v.location(ctx, origin)
	if err != nil {
		return Record{}, "", err
	}
	raw, err := v.lookup(ctx, account)
	if ctx.Err() != nil {
		return Record{}, canonical, ctx.Err()
	}
	var lookupError *api.Error
	if errors.As(err, &lookupError) {
		return Record{}, canonical, lookupError
	}
	if errors.Is(err, ErrMissing) {
		return Record{}, canonical, &api.Error{Code: "not_authenticated"}
	}
	if err != nil {
		return Record{}, canonical, &api.Error{Code: "credential_store_unavailable"}
	}
	record, err := decodeRecord(raw, canonical, v.allowInsecureHTTP)
	return record, canonical, err
}

// wireRecord 是只用于系统凭证库的内部序列化格式，不用于公开命令输出。
type wireRecord struct {
	Version      int       `json:"version"`
	Origin       string    `json:"origin"`
	CredentialID string    `json:"credential_id"`
	UserID       string    `json:"user_id"`
	ExpiresAt    time.Time `json:"expires_at"`
	Token        string    `json:"token"`
}

// encodeRecord 限制系统凭证库载荷大小，保持六种发行平台的存储契约一致。
//
// Args:
//
//	record: 已验证的凭证。
//
// Returns:
//
//	内部版本化 JSON，或安全错误。
//
// Errors:
//
//	编码或大小校验失败时返回 invalid_credential。
func encodeRecord(record Record) (string, error) {
	payload := wireRecord{Version: 1, Origin: record.Origin, CredentialID: record.CredentialID, UserID: record.UserID, ExpiresAt: record.ExpiresAt, Token: record.Token}
	raw, err := json.Marshal(payload)
	if err != nil || len(raw) > maxRecordBytes {
		return "", &api.Error{Code: "invalid_credential"}
	}
	return string(raw), nil
}

// decodeRecord 对记录版本、大小和 origin 重新校验，不能信任本地保存的数据。
//
// Args:
//
//	raw: 系统凭证库返回的内部 JSON。
//	origin: 当前选择的规范化 origin。
//
// Returns:
//
//	验证后的凭证，或安全错误。
//
// Errors:
//
//	任何载荷或绑定错误都返回 credential_corrupt，不回显原始数据。
func decodeRecord(raw, origin string, allowInsecureHTTP bool) (Record, error) {
	var payload wireRecord
	if len(raw) > maxRecordBytes || json.Unmarshal([]byte(raw), &payload) != nil || payload.Version != 1 {
		return Record{}, &api.Error{Code: "credential_corrupt"}
	}
	record := Record{Origin: payload.Origin, CredentialID: payload.CredentialID, UserID: payload.UserID, ExpiresAt: payload.ExpiresAt, Token: payload.Token}
	if err := validateRecord(record, origin, allowInsecureHTTP); err != nil {
		return Record{}, &api.Error{Code: "credential_corrupt"}
	}
	return record, nil
}
