package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/credential"
)

// pollResponse 仅存在于兑换边界，不能作为公开命令输出。
type pollResponse struct {
	Status       string    `json:"status"`
	Interval     int       `json:"interval"`
	Token        string    `json:"access_token"`
	CredentialID string    `json:"credential_id"`
	UserID       string    `json:"user_id"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Complete 有界轮询先前的当前 origin 请求，用户确认不等于绕过官网授权。
//
// Args:
//
//	ctx: 调用方上下文；取消只结束本地等待。
//	timeout: 本地等待期限，范围一秒到十分钟。
//
// Returns:
//
//	登录结果、待授权状态或安全错误。
//
// Errors:
//
//	证明缺失、坏响应、网络或存储失败时停止，不自动重新创建设备请求。
func (s *Service) Complete(ctx context.Context, timeout time.Duration) (Outcome, error) {
	if timeout < time.Second || timeout > 10*time.Minute {
		return Outcome{}, &api.Error{Code: "invalid_timeout"}
	}
	pending, err := s.vault.LoadPending(ctx, s.client.Origin())
	if err != nil {
		return Outcome{}, err
	}
	bounded, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		outcome, stopped, err := s.awaitPoll(bounded, ctx, pending)
		if stopped {
			return outcome, err
		}
		var response pollResponse
		err = s.client.DoJSON(bounded, http.MethodPost, "/api/v1/client/auth/token", nil, map[string]string{"device_code": pending.DeviceCode, "verifier": pending.Verifier}, &response)
		if err != nil {
			return pollFailure(bounded, ctx, pending, err)
		}
		outcome, done, err := s.handlePoll(ctx, &pending, response)
		if done || err != nil {
			return outcome, err
		}
	}
}

// pollFailure 区分用户取消、有限期等待和真正传输失败。
//
// Args:
//
//	bounded: 本地轮询期限。
//	parent: 调用方取消上下文。
//	pending: 尚未确认成功的设备请求。
//	err: 脱敏传输错误。
//
// Returns:
//
//	公开等待结果，或取消/网络错误。
//
// Errors:
//
//	除本地轮询期限到达外，保留取消或安全传输错误；不猜测服务器是否已兑换。
func pollFailure(bounded, parent context.Context, pending credential.Pending, err error) (Outcome, error) {
	if parent.Err() != nil {
		return Outcome{}, parent.Err()
	}
	if bounded.Err() != nil {
		return pendingOutcome(pending, "pending", "timeout"), nil
	}
	return Outcome{}, err
}

// awaitPoll 等待服务器允许的窗口，设备到期和本地等待超时分别处理。
//
// Args:
//
//	bounded: 本地有限期等待。
//	parent: 调用方取消上下文。
//	pending: 当前设备请求。
//
// Returns:
//
//	停止时的公开结果、是否停止、或调用方取消错误。
//
// Errors:
//
//	调用方取消或到期清理失败时返回错误；本地超时不取消云端授权。
func (s *Service) awaitPoll(bounded, parent context.Context, pending credential.Pending) (Outcome, bool, error) {
	if !pending.ExpiresAt.After(s.now()) {
		outcome, err := s.terminal(parent, pending, "expired")
		return outcome, true, err
	}
	next := pending.NextPollAt
	if pending.ExpiresAt.Before(next) {
		next = pending.ExpiresAt
	}
	if err := waitUntil(bounded, next); err != nil {
		if parent.Err() != nil {
			return Outcome{}, true, parent.Err()
		}
		return pendingOutcome(pending, "pending", "timeout"), true, nil
	}
	if !pending.ExpiresAt.After(s.now()) {
		outcome, err := s.terminal(parent, pending, "expired")
		return outcome, true, err
	}
	return Outcome{}, false, nil
}

// handlePoll 按状态保存等待窗口或结束兑换，不允许等待响应夹带秘密。
//
// Args:
//
//	ctx: 本地存储上下文。
//	pending: 当前设备请求；等待时更新其下次轮询时间。
//	response: 内部兑换响应。
//
// Returns:
//
//	公开结果、是否停止轮询、或安全错误。
//
// Errors:
//
//	未知状态、非法间隔或存储失败时停止。
func (s *Service) handlePoll(ctx context.Context, pending *credential.Pending, response pollResponse) (Outcome, bool, error) {
	switch response.Status {
	case "authorized":
		outcome, err := s.accept(ctx, *pending, response)
		return outcome, true, err
	case "denied", "expired":
		outcome, err := s.terminal(ctx, *pending, response.Status)
		return outcome, true, err
	case "pending", "slow_down":
		if response.Interval < 5 || response.Interval > 60 || response.Token != "" {
			return Outcome{}, true, &api.Error{Code: "invalid_response"}
		}
		pending.Interval = response.Interval
		pending.NextPollAt = s.now().Add(time.Duration(response.Interval) * time.Second)
		return Outcome{}, false, s.vault.SavePending(ctx, pending.Origin, *pending)
	default:
		return Outcome{}, true, &api.Error{Code: "invalid_response"}
	}
}

// accept 校验并保存凭证；保存失败时尽力撤销刚签发的当前凭证。
//
// Args:
//
//	ctx: 本地存储上下文。
//	pending: 当前 origin 的设备请求。
//	response: 一次性授权响应，不输出原对象。
//
// Returns:
//
//	无秘密的登录结果，或安全错误。
//
// Errors:
//
//	凭证不合法、保存或请求清理失败时不伪装成功。
func (s *Service) accept(ctx context.Context, pending credential.Pending, response pollResponse) (Outcome, error) {
	if !strings.HasPrefix(response.Token, "ccn_cli_") || len(response.Token) != 51 || response.ExpiresAt.After(s.now().Add(31*24*time.Hour)) {
		return Outcome{}, &api.Error{Code: "invalid_response"}
	}
	record := credential.Record{Origin: pending.Origin, Token: response.Token, CredentialID: response.CredentialID, UserID: response.UserID, ExpiresAt: response.ExpiresAt}
	if err := s.vault.Save(ctx, pending.Origin, record); err != nil {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		client, createErr := api.NewWithInsecureHTTP(pending.Origin, response.Token, nil)
		if createErr == nil {
			_ = client.DoJSON(cleanup, http.MethodPost, "/api/v1/client/auth/revoke", nil, nil, nil)
		}
		return Outcome{}, err
	}
	if err := s.vault.ForgetPending(ctx, pending.Origin); err != nil {
		return Outcome{}, err
	}
	return Outcome{Status: "authenticated", Origin: record.Origin, CredentialID: record.CredentialID, UserID: record.UserID, ExpiresAt: &record.ExpiresAt}, nil
}

// terminal 清理不可恢复的短期请求，不删除独立的网站或其他环境凭证。
//
// Args:
//
//	ctx: 本地取消上下文。
//	pending: 短期请求。
//	status: 拒绝或过期状态。
//
// Returns:
//
//	公开终止状态，或清理错误。
//
// Errors:
//
//	凭证库不能删除时返回安全错误。
func (s *Service) terminal(ctx context.Context, pending credential.Pending, status string) (Outcome, error) {
	if err := s.vault.ForgetPending(ctx, pending.Origin); err != nil {
		return Outcome{}, err
	}
	return pendingOutcome(pending, status, ""), nil
}

// waitUntil 用可取消计时器等待限频窗口，不在后台创建无限轮询。
//
// Args:
//
//	ctx: 有期限的调用上下文。
//	next: 下次允许轮询时间。
//
// Returns:
//
//	等待完成时 nil，或上下文错误。
//
// Errors:
//
//	取消或期限到达时返回 context 错误。
func waitUntil(ctx context.Context, next time.Time) error {
	timer := time.NewTimer(time.Until(next))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}
