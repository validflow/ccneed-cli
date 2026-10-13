// Package conversations 连接真实会话，不用标题或本地 task_id 替代 thread_id。
package conversations

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"unicode/utf8"

	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/resources"
)

// Client 只调用现有会话 API，不运行云端 Agent。
type Client struct{ api *api.Client }

// Created 是新建会话及初始绑定的公开回执。
type Created struct {
	ThreadID        string   `json:"thread_id"`
	KBIDs           []int64  `json:"kb_ids"`
	FileIDs         []string `json:"library_file_ids"`
	MetadataVersion string   `json:"metadata_version"`
	ResourceWindow  string   `json:"resource_window_state"`
	Title           *string  `json:"title"`
}

// Summary 表示用户可选的真实根会话，不包含原始日志或隐藏执行状态。
type Summary struct {
	ThreadID       string   `json:"thread_id"`
	Title          string   `json:"title"`
	Preview        *string  `json:"preview"`
	Archived       bool     `json:"archived"`
	KBIDs          []int64  `json:"kb_ids"`
	FileIDs        []string `json:"library_file_ids"`
	ResourceWindow string   `json:"resource_window_state"`
}

// Listing 显示现有接口的来源上限，不把截断列表描述成全部历史。
type Listing struct {
	Threads         []Summary `json:"threads"`
	SourceLimit     int       `json:"source_limit"`
	HasMorePossible bool      `json:"has_more_possible"`
}

// New 注入当前 origin 的认证传输。
//
// Args:
//
//	client: 已选择且已认证的 API 连接。
//
// Returns:
//
//	会话能力客户端。
func New(client *api.Client) *Client { return &Client{api: client} }

// ValidateCreate 在连接之前校验初始资源与可选标题。
//
// Args:
//
//	selection: 已有知识库和用户文件标识。
//	title: 可选的初始标题，最多两百 Unicode 字符。
//
// Returns:
//
//	参数检查结果。
//
// Errors:
//
//	无效绑定或标题返回 invalid_input。
func ValidateCreate(selection resources.Selection, title string) error {
	if err := resources.ValidateSelection(selection); err != nil {
		return err
	}
	if !utf8.ValidString(title) || utf8.RuneCountInString(title) > 200 {
		return &api.Error{Code: "invalid_input"}
	}
	return nil
}

// Create 只创建会话和初始绑定，不提交任务，不自动重试非幂等写入。
//
// Args:
//
//	ctx: 本地 HTTP 上下文。
//	selection: 明确选择的已有资源。
//	title: 可选标题，空值由网站现有逻辑处理。
//
// Returns:
//
//	初始绑定回执，或错误。
//
// Errors:
//
//	无法确认接收时返回 acceptance_unknown，应先查看会话列表而非再次创建。
func (c *Client) Create(ctx context.Context, selection resources.Selection, title string) (Created, error) {
	if err := ValidateCreate(selection, title); err != nil {
		return Created{}, err
	}
	if selection.KBIDs == nil {
		selection.KBIDs = []int64{}
	}
	if selection.FileIDs == nil {
		selection.FileIDs = []string{}
	}
	body := struct {
		KBIDs   []int64  `json:"seed_kb_ids"`
		FileIDs []string `json:"library_file_ids"`
		Title   string   `json:"title,omitempty"`
	}{selection.KBIDs, selection.FileIDs, title}
	var result Created
	if err := c.api.DoJSON(ctx, http.MethodPost, "/api/v1/usight-agent/threads", nil, body, &result); err != nil {
		return Created{}, createFailure(err)
	}
	if !resources.ValidUUID(result.ThreadID) || !validWindow(result.ResourceWindow) || !validRevision(result.MetadataVersion) || resources.ValidateSelection(resources.Selection{KBIDs: result.KBIDs, FileIDs: result.FileIDs}) != nil {
		return Created{}, &api.Error{Code: "acceptance_unknown"}
	}
	return result, nil
}

// DeleteEmpty reclaims a newly created shell only if no turn was admitted.
//
// Args:
//
//	ctx: Request context.
//	threadID: Root conversation created for the first message.
//
// Returns:
//
// An error when the guarded cleanup request fails.
func (c *Client) DeleteEmpty(ctx context.Context, threadID string) error {
	if !resources.ValidUUID(threadID) {
		return &api.Error{Code: "invalid_input"}
	}
	query := url.Values{"only_if_empty": {"true"}}
	var result struct {
		Deleted bool `json:"deleted"`
	}
	return c.api.DoJSON(ctx, http.MethodDelete, "/api/v1/usight-agent/threads/"+threadID, query, nil, &result)
}

// ValidateList 校验列表参数，防止拼接未公布查询。
//
// Args:
//
//	view: active 或 archived。
//	limit: 本地展示数量，一到两百。
//
// Returns:
//
//	参数检查结果。
//
// Errors:
//
//	未知视图或数量返回 invalid_input。
func ValidateList(view string, limit int) error {
	if (view != "active" && view != "archived") || limit < 1 || limit > 200 {
		return &api.Error{Code: "invalid_input"}
	}
	return nil
}

// List 读取服务现有最近两百个会话，不把同名会话自动合并。
//
// Args:
//
//	ctx: 本地请求上下文。
//	view: active 或 archived。
//	limit: 本地展示上限。
//
// Returns:
//
//	真实会话列表及截断提示，或错误。
//
// Errors:
//
//	无效参数、鉴权或响应不符合公开契约时失败关闭。
func (c *Client) List(ctx context.Context, view string, limit int) (Listing, error) {
	if err := ValidateList(view, limit); err != nil {
		return Listing{}, err
	}
	rows := []Summary{}
	if err := c.api.DoJSON(ctx, http.MethodGet, "/api/v1/usight-agent/threads", url.Values{"view": {view}}, nil, &rows); err != nil {
		return Listing{}, err
	}
	if len(rows) > 200 {
		return Listing{}, &api.Error{Code: "invalid_response"}
	}
	for _, row := range rows {
		if !resources.ValidUUID(row.ThreadID) || row.Title == "" || !validWindow(row.ResourceWindow) {
			return Listing{}, &api.Error{Code: "invalid_response"}
		}
	}
	if rows == nil {
		rows = []Summary{}
	}
	result := Listing{Threads: rows, SourceLimit: 200, HasMorePossible: len(rows) > limit || len(rows) == 200}
	if len(rows) > limit {
		result.Threads = rows[:limit]
	}
	return result, nil
}

// validWindow 限制服务器所有的资料绑定窗口，不由客户端扩大权限。
//
// Args:
//
//	state: 服务返回的资源窗口。
//
// Returns:
//
//	既有窗口集合包含该值时为真。
func validWindow(state string) bool {
	return state == "initial" || state == "locked" || state == "report_window"
}

// validRevision 检查可用于 CAS 的非负元数据版本。
//
// Args:
//
//	version: 服务器的十进制版本字符串。
//
// Returns:
//
//	可解析非负整数时为真。
func validRevision(version string) bool {
	_, err := strconv.ParseUint(version, 10, 64)
	return err == nil
}

// createFailure 保留明确拒绝，网络或响应不确定时禁止自动再创建。
//
// Args:
//
//	err: 单次 HTTP 或解码错误。
//
// Returns:
//
//	安全公开错误。
//
// Errors:
//
//	非明确 4xx 返回 acceptance_unknown。
func createFailure(err error) error {
	var remote *api.Error
	if errors.As(err, &remote) && remote.Status >= 400 && remote.Status < 500 {
		return err
	}
	return &api.Error{Code: "acceptance_unknown"}
}
