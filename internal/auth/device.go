// Package auth 负责官网设备授权，不复制网站 Cookie，不在终端输出秘密。
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/credential"
)

// Service 将登录 HTTP、短期证明与系统存储组合，不管理云端任务。
type Service struct {
	client *api.Client
	vault  *credential.Vault
	now    func() time.Time
}

// Outcome 是公开授权状态，不包含设备秘密或客户端 token。
type Outcome struct {
	Status          string     `json:"status"`
	Origin          string     `json:"origin"`
	VerificationURL string     `json:"verification_url,omitempty"`
	UserCode        string     `json:"user_code,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	CredentialID    string     `json:"credential_id,omitempty"`
	UserID          string     `json:"user_id,omitempty"`
	Reason          string     `json:"reason,omitempty"`
}

// New 构建 origin 已绑定的设备授权业务。
//
// Args:
//
//	client: 已选服务的无凭证 HTTP client。
//	vault: 系统凭证存储。
//	now: 时钟；nil 使用 time.Now。
//
// Returns:
//
//	独立授权业务。
func New(client *api.Client, vault *credential.Vault, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{client: client, vault: vault, now: now}
}

// Begin 立即返回官网链接，把设备证明留在系统凭证库；不会阻塞等用户。
//
// Args:
//
//	ctx: 本地请求上下文。
//
// Returns:
//
//	可展示给用户的等待状态，或安全错误。
//
// Errors:
//
//	已有有效凭证、配置错误、网络失败或系统存储失败时不静默覆盖。
func (s *Service) Begin(ctx context.Context) (Outcome, error) {
	origin := s.client.Origin()
	local, err := s.vault.Inspect(ctx, origin)
	if err != nil {
		return Outcome{}, err
	}
	if local.State == "present" {
		return Outcome{}, &api.Error{Code: "credential_already_present"}
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return Outcome{}, &api.Error{Code: "randomness_unavailable"}
	}
	verifier := base64.RawURLEncoding.EncodeToString(random)
	challenge := fmt.Sprintf("%x", sha256.Sum256([]byte(verifier)))
	started := s.now()
	var response struct {
		DeviceCode       string `json:"device_code"`
		UserCode         string `json:"user_code"`
		VerificationPath string `json:"verification_path"`
		ExpiresIn        int    `json:"expires_in"`
		Interval         int    `json:"interval"`
	}
	err = s.client.DoJSON(ctx, http.MethodPost, "/api/v1/client/auth/device", nil, map[string]string{"challenge": challenge}, &response)
	if err != nil {
		return Outcome{}, err
	}
	if response.VerificationPath != "/auth/cli" || response.ExpiresIn != 600 || response.Interval != 5 {
		return Outcome{}, &api.Error{Code: "invalid_response"}
	}
	pending := credential.Pending{Origin: origin, DeviceCode: response.DeviceCode, Verifier: verifier, UserCode: response.UserCode,
		ExpiresAt: started.Add(10 * time.Minute), NextPollAt: s.now().Add(5 * time.Second), Interval: 5}
	if err := s.vault.SavePending(ctx, origin, pending); err != nil {
		return Outcome{}, err
	}
	return pendingOutcome(pending, "pending", ""), nil
}

// pendingOutcome 只投影用户可以展示的字段；链接跟随请求 origin，不接收任意服务 URL。
//
// Args:
//
//	pending: 内部设备请求。
//	status: 公开状态。
//	reason: 本地等待停止原因。
//
// Returns:
//
//	不含秘密的状态。
func pendingOutcome(pending credential.Pending, status, reason string) Outcome {
	return Outcome{Status: status, Origin: pending.Origin, VerificationURL: pending.Origin + "/auth/cli?" + url.Values{"user_code": {pending.UserCode}}.Encode(),
		UserCode: pending.UserCode, ExpiresAt: &pending.ExpiresAt, Reason: reason}
}
