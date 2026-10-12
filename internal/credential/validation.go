package credential

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/validflow/ccneed-cli/internal/api"
)

const keyringService = "ccneed-cli.credentials.v1"
const maxRecordBytes = 2048

// location 校验地址并生成不含用户输入的固定长度凭证库键。
//
// Args:
//
//	ctx: 本地取消上下文；取消后不访问凭证库。
//	origin: 用户显式选择的服务 origin。
//
// Returns:
//
//	规范化 origin、origin 摘要键，或安全错误。
//
// Errors:
//
//	地址无效、上下文取消或凭证库未配置时失败关闭。
func (v *Vault) location(ctx context.Context, origin string) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	parse := api.ParseOrigin
	if v != nil && v.allowInsecureHTTP {
		parse = api.ParseOriginWithInsecureHTTP
	}
	parsed, err := parse(origin)
	if err != nil {
		return "", "", &api.Error{Code: "invalid_origin"}
	}
	if v == nil || v.backend == nil {
		return "", "", &api.Error{Code: "credential_store_unavailable"}
	}
	canonical := parsed.String()
	return canonical, fmt.Sprintf("%x", sha256.Sum256([]byte(canonical))), nil
}

// validateRecord 校验凭证身份、地址绑定与安全序列化字段。
//
// Args:
//
//	record: 凭证记录，尚未证明来源可信。
//	origin: 当前选择的规范化 origin。
//
// Returns:
//
//	校验错误，或 nil。
//
// Errors:
//
//	绑定不符返回 origin_mismatch，其余字段错误返回 invalid_credential。
func validateRecord(record Record, origin string, allowInsecureHTTP bool) error {
	parse := api.ParseOrigin
	if allowInsecureHTTP {
		parse = api.ParseOriginWithInsecureHTTP
	}
	bound, err := parse(record.Origin)
	if err != nil {
		return &api.Error{Code: "invalid_credential"}
	}
	if bound.String() != origin {
		return &api.Error{Code: "origin_mismatch"}
	}
	if !safeIdentifier(record.CredentialID) || !safeIdentifier(record.UserID) || record.ExpiresAt.IsZero() {
		return &api.Error{Code: "invalid_credential"}
	}
	if record.Token == "" || len(record.Token) > 1024 ||
		strings.IndexFunc(record.Token, func(char rune) bool { return char < 33 || char > 126 }) != -1 {
		return &api.Error{Code: "invalid_credential"}
	}
	return nil
}

// safeIdentifier 拒绝控制字符和过长身份，避免状态输出包含终端控制序列。
//
// Args:
//
//	value: 服务返回的身份标识。
//
// Returns:
//
//	有界、可打印 ASCII 身份时为 true。
func safeIdentifier(value string) bool {
	return len(value) > 0 && len(value) <= 128 &&
		strings.IndexFunc(value, func(char rune) bool { return char < 33 || char > 126 }) == -1
}
