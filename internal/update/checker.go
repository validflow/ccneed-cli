// Package update 检查公开稳定发行，离线或检查失败不影响产品命令。
package update

import (
	"context"
	"time"

	"github.com/validflow/ccneed-cli/internal/version"
)

// Record 是不含凭证的最新检查结果；空版本也用于缓存检查失败。
type Record struct {
	CheckedAt time.Time `json:"checked_at"`
	Version   string    `json:"version"`
}

// Cache 只保存公开版本信息，不得保存账号或 API 凭证。
type Cache interface {
	Load() (Record, error)
	Save(Record) error
}

// Source 提供当前公开稳定版本，不接受产品 API 的认证信息。
type Source interface {
	Latest(context.Context) (string, error)
}

// Checker 组合版本来源、缓存与时钟；依赖在构造时确定。
type Checker struct {
	source Source
	cache  Cache
	now    func() time.Time
}

// New 创建每日最多检查一次的更新检测器。
//
// Args:
//
//	source: 公开发行来源。
//	cache: 更新缓存；读取或写入失败不影响命令。
//	now: 可注入当前 UTC 时刻的函数。
//
// Returns:
//
//	有限期检查器。
func New(source Source, cache Cache, now func() time.Time) *Checker {
	return &Checker{source: source, cache: cache, now: now}
}

// Check 返回新稳定版本；所有检查错误都降级为空提醒。
//
// Args:
//
//	ctx: 检查上下文，可由调用方提前取消。
//	current: 当前 CLI 版本，开发版本不查询网络。
//
// Returns:
//
//	应提醒的稳定版本，或空字符串。
func (c *Checker) Check(ctx context.Context, current string) string {
	if !version.IsStable(current) {
		return ""
	}
	now := c.now()
	record, err := c.cache.Load()
	age := now.Sub(record.CheckedAt)
	if err != nil || record.CheckedAt.IsZero() || age < 0 || age >= 24*time.Hour {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		latest, fetchErr := c.source.Latest(ctx)
		if fetchErr != nil {
			latest = ""
		}
		record = Record{CheckedAt: now, Version: latest}
		_ = c.cache.Save(record)
	}
	if version.NewerStable(record.Version, current) {
		return record.Version
	}
	return ""
}
