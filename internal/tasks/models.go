package tasks

import (
	"encoding/json"
	"time"
)

// Action 是完成一次用户确认所需的公开参数，不含原始 checkpoint。
type Action struct {
	ID        string          `json:"action_id"`
	Type      string          `json:"action_type"`
	Reason    *string         `json:"reason,omitempty"`
	Schema    json.RawMessage `json:"resume_schema"`
	Suggested json.RawMessage `json:"suggested,omitempty"`
}

// Artifact 是当前 manifest 中关联本轮的 finding/report，不承诺历史字节不变。
type Artifact struct {
	ID          string  `json:"artifact_id"`
	Name        string  `json:"name"`
	Type        string  `json:"artifact_type"`
	Size        *int64  `json:"size_bytes,omitempty"`
	ContentType *string `json:"content_type,omitempty"`
}

// ProgressEvent is one server-projected, non-sensitive milestone for this turn.
type ProgressEvent struct {
	Seq     int64      `json:"seq"`
	At      *time.Time `json:"at,omitempty"`
	Stage   string     `json:"stage"`
	State   string     `json:"state"`
	Summary string     `json:"summary"`
}

// ProgressPage holds chronological milestones and directional keyset cursors.
type ProgressPage struct {
	Events        []ProgressEvent `json:"events"`
	HasMore       bool            `json:"has_more"`
	NextBeforeSeq *int64          `json:"next_before_seq"`
	NextAfterSeq  *int64          `json:"next_after_seq"`
}

// Detail 只包含服务器承诺的公开回合字段，忽略未知内部字段。
type Detail struct {
	ThreadID              string       `json:"thread_id"`
	TurnID                string       `json:"turn_id"`
	ConversationTurnIndex int          `json:"conversation_turn_idx"`
	Status                string       `json:"status"`
	CloudStatus           string       `json:"cloud_status"`
	StartedAt             *time.Time   `json:"started_at,omitempty"`
	EndedAt               *time.Time   `json:"ended_at,omitempty"`
	Answer                *string      `json:"answer"`
	AnswerTruncated       bool         `json:"answer_truncated"`
	PendingAction         *Action      `json:"pending_action,omitempty"`
	Artifacts             []Artifact   `json:"artifacts"`
	ArtifactsHasMore      bool         `json:"artifacts_has_more"`
	Progress              ProgressPage `json:"progress"`
	WebURL                string       `json:"web_url,omitempty"`
}

// WatchResult 区分观察停止原因与仍在运行的云端状态。
type WatchResult struct {
	Detail
	StopReason string `json:"stop_reason"`
}
