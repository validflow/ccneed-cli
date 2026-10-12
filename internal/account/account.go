// Package account 复用平台现有只读积分和订阅接口，不复制账本或发起支付。
package account

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/validflow/ccneed-cli/internal/api"
)

// Balance 是平台返回的当前可用积分，不把 is_pro 误解释成订阅权益。
type Balance struct {
	LeftCredits int64 `json:"left_credits"`
}

// Subscription 只包含账号概览需要的公开计划及到期信息。
type Subscription struct {
	PlanName  *string    `json:"plan_name"`
	PlanLevel *int       `json:"plan_level"`
	ExpiredAt *time.Time `json:"expired_at"`
}

// Overview 汇总两个既有只读接口，不触发初始化或注册追踪。
type Overview struct {
	Credits       Balance        `json:"credits"`
	Subscriptions []Subscription `json:"subscriptions"`
}

// CreditEntry 是消费记录的公开投影，不返回数据库用户字段或内部元数据。
type CreditEntry struct {
	Transaction string       `json:"trans_no"`
	Type        string       `json:"trans_type"`
	Credits     *json.Number `json:"credits,omitempty"`
	CreatedAt   *time.Time   `json:"created_at,omitempty"`
	ExpiresAt   *time.Time   `json:"expired_at,omitempty"`
}

// History 明确是近期窗口，不声称支持完整历史或服务端分页。
type History struct {
	Items       []CreditEntry `json:"items"`
	Scope       string        `json:"scope"`
	SourceLimit int           `json:"source_limit"`
}

// Client 只执行当前登录身份的 HTTP 读取，没有 user_id 或支付参数。
type Client struct{ api *api.Client }

// New 注入已经绑定 origin 和账号的传输。
//
// Args:
//
//	client: 已配置的 API 客户端。
//
// Returns:
//
//	不包含本地账本逻辑的账号读取器。
func New(client *api.Client) *Client { return &Client{api: client} }

// Credits 查询服务端权威余额，不重新计算资产。
//
// Args:
//
//	ctx: 请求上下文。
//
// Returns:
//
//	可用积分，或读取错误。
//
// Errors:
//
//	返回鉴权、传输或余额契约错误。
func (c *Client) Credits(ctx context.Context) (Balance, error) {
	var response struct {
		LeftCredits *int64 `json:"left_credits"`
	}
	if err := c.api.DoJSON(ctx, http.MethodGet, "/api/v1/payment/credits", nil, nil, &response); err != nil {
		return Balance{}, err
	}
	if response.LeftCredits == nil || *response.LeftCredits < 0 {
		return Balance{}, &api.Error{Code: "invalid_response"}
	}
	return Balance{LeftCredits: *response.LeftCredits}, nil
}

// Overview 获取余额和有效订阅，任一读取失败都不返回伪造的完整结果。
//
// Args:
//
//	ctx: 同一调用方上下文。
//
// Returns:
//
//	完整只读概览，或错误。
//
// Errors:
//
//	返回余额或订阅读取错误。
func (c *Client) Overview(ctx context.Context) (Overview, error) {
	credits, err := c.Credits(ctx)
	if err != nil {
		return Overview{}, err
	}
	var response struct {
		Subscriptions *[]Subscription `json:"subscriptions"`
	}
	if err := c.api.DoJSON(ctx, http.MethodGet, "/api/v1/redemption/subscriptions", nil, nil, &response); err != nil {
		return Overview{}, err
	}
	if response.Subscriptions == nil {
		return Overview{}, &api.Error{Code: "invalid_response"}
	}
	return Overview{Credits: credits, Subscriptions: *response.Subscriptions}, nil
}

// History 读取现有接口最近 200 条，再按本地 limit 截取。
//
// Args:
//
//	ctx: 请求上下文。
//	limit: 输出数量，范围为 1–200；不是服务端分页参数。
//
// Returns:
//
//	近期记录和窗口说明，或错误。
//
// Errors:
//
//	返回非法 limit、鉴权或响应错误。
func (c *Client) History(ctx context.Context, limit int) (History, error) {
	if limit < 1 || limit > 200 {
		return History{}, &api.Error{Code: "invalid_input"}
	}
	var items []CreditEntry
	if err := c.api.DoJSON(ctx, http.MethodGet, "/api/v1/payment/credits/list", nil, nil, &items); err != nil {
		return History{}, err
	}
	if items == nil {
		items = []CreditEntry{}
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return History{Items: items, Scope: "recent", SourceLimit: 200}, nil
}
