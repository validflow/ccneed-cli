package tasks

import (
	"context"
	"errors"
	"time"

	"github.com/validflow/ccneed-cli/internal/api"
)

// ValidateTimeout 限制终端观察期限，不允许无界阻塞。
//
// Args:
//
//	timeout: 本地观察的最大时长。
//
// Returns:
//
//	参数错误，或 nil。
//
// Errors:
//
//	时长小于一秒或超过十分钟时返回 invalid_input。
func ValidateTimeout(timeout time.Duration) error {
	if timeout < time.Second || timeout > 10*time.Minute {
		return &api.Error{Code: "invalid_input"}
	}
	return nil
}

// Watch 有期限地读取同一回合；等待用户、可恢复中断或终态立即交回宿主。
//
// Args:
//
//	ctx: 宿主取消上下文；取消只停止本地观察。
//	threadID: 根会话 UUID。
//	turnID: 目标回合 UUID。
//	timeout: 一秒到十分钟的本地观察期限。
//
// Returns:
//
//	最近一次状态和停止原因，或查询错误。
//
// Errors:
//
//	参数、权限或坏响应失败时返回错误；超时不表示云端已停止。
func (c *Client) Watch(ctx context.Context, threadID, turnID string, timeout time.Duration) (WatchResult, error) {
	if err := ValidateIDs(threadID, turnID); err != nil {
		return WatchResult{}, err
	}
	if err := ValidateTimeout(timeout); err != nil {
		return WatchResult{}, err
	}
	observation, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.observe(ctx, observation, threadID, turnID)
}

// observe 保留最后可信快照，观察截止不写入云端状态。
//
// Args:
//
//	ctx: 宿主原始上下文。
//	observation: 已施加本地期限的观察上下文。
//	threadID: 根会话 UUID。
//	turnID: 目标回合 UUID。
//
// Returns:
//
//	最后公开快照与停止原因，或查询错误。
//
// Errors:
//
//	网络、权限、协议或宿主取消时返回错误。
func (c *Client) observe(ctx, observation context.Context, threadID, turnID string) (WatchResult, error) {
	last := Detail{
		ThreadID: threadID, TurnID: turnID, Status: "unknown", CloudStatus: "unknown",
		Artifacts: []Artifact{}, Progress: ProgressPage{Events: []ProgressEvent{}},
		WebURL: c.api.Origin() + "/insightflow/" + threadID,
	}
	for {
		result, err := c.Get(observation, threadID, turnID)
		if err != nil {
			if observation.Err() != nil && ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
				return WatchResult{Detail: last, StopReason: "timeout"}, nil
			}
			return WatchResult{}, err
		}
		last = result
		if reason := stopReason(result.Status); reason != "" {
			return WatchResult{Detail: result, StopReason: reason}, nil
		}
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-observation.Done():
			timer.Stop()
			if ctx.Err() != nil {
				return WatchResult{}, ctx.Err()
			}
			return WatchResult{Detail: last, StopReason: "timeout"}, nil
		case <-timer.C:
		}
	}
}

// stopReason 仅在宿主应接手或回合结束时停止本地观察。
//
// Args:
//
//	status: 经过协议校验的公开状态。
//
// Returns:
//
//	停止原因；继续运行或等待外部系统时返回空字符串。
func stopReason(status string) string {
	switch status {
	case "waiting_user", "interrupted", "completed", "failed", "cancelled":
		return status
	default:
		return ""
	}
}
