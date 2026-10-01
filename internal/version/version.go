// Package version 维护 CLI 构建信息及稳定发行版本比较，不跟随 Club 服务版本。
package version

import (
	"regexp"
	"strings"
)

var stablePattern = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

// IsStable 校验稳定发行身份，排除开发版本和预发行版本。
//
// Args:
//
//	value: 构建或发行版本字符串。
//
// Returns:
//
//	合法稳定 SemVer 时返回 true。
func IsStable(value string) bool { return stablePattern.MatchString(value) }

// Info 是编译时注入的非敏感发行身份。
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// NewerStable 判断两个稳定 SemVer 中候选是否更新。
//
// Args:
//
//	candidate: 发布源给出的版本，允许 v 前缀和 build metadata。
//	current: 本地版本；dev、预发行和非法值不参加提醒。
//
// Returns:
//
//	两者有效且 candidate 严格更新时返回 true。
func NewerStable(candidate, current string) bool {
	next := stablePattern.FindStringSubmatch(candidate)
	installed := stablePattern.FindStringSubmatch(current)
	if next == nil || installed == nil {
		return false
	}
	for index := 1; index <= 3; index++ {
		if difference := compareNumber(next[index], installed[index]); difference != 0 {
			return difference > 0
		}
	}
	return false
}

// compareNumber 比较无前导零的十进制文本，避免机器整数溢出。
//
// Args:
//
//	left: 正则已校验的十进制数。
//	right: 正则已校验的十进制数。
//
// Returns:
//
//	left 较小时为负，相等为零，较大为正。
func compareNumber(left, right string) int {
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	return strings.Compare(left, right)
}
