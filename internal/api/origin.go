// Package api 提供 origin 绑定、有限期且不重试写请求的 HTTP 传输。
package api

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
)

var errOrigin = errors.New("invalid service origin: use an HTTPS origin or explicit HTTP loopback")

// ParseOrigin 校验服务地址，开发 loopback 可以显式使用 HTTP。
//
// Args:
//
//	raw: 仅包含 scheme、host 和可选端口的地址；不包含 API 路径。
//
// Returns:
//
//	规范化 origin，或地址错误。
//
// Errors:
//
//	拒绝无法绑定凭证的地址。
func ParseOrigin(raw string) (*url.URL, error) {
	return parseOrigin(raw, false)
}

// ParseOriginWithInsecureHTTP allows an explicitly selected HTTP staging IP.
// It retains all origin-only, port, userinfo, and scheme validation.
//
// Args:
//
//	raw: Explicit service origin, without paths, queries, or credentials. Remote HTTP requires a literal IP.
//
// Returns:
//
//	Canonical origin, or an address error.
//
// Errors:
//
//	Rejects malformed or non-HTTP(S) addresses.
func ParseOriginWithInsecureHTTP(raw string) (*url.URL, error) {
	return parseOrigin(raw, true)
}

// parseOrigin centralizes canonicalization for secure and explicit HTTP modes.
//
// Args:
//
//	raw: Service origin to validate.
//	allowInsecureHTTP: Whether non-loopback HTTP was explicitly selected.
//
// Returns:
//
//	Canonical origin, or an address error.
//
// Errors:
//
//	Rejects malformed origins and disallowed schemes.
func parseOrigin(raw string, allowInsecureHTTP bool) (*url.URL, error) {
	origin, err := url.Parse(raw)
	if err != nil || !isOriginOnly(origin) || strings.Contains(raw, "#") {
		return nil, errOrigin
	}
	host := strings.ToLower(origin.Hostname())
	allowed := secureScheme(origin.Scheme, host)
	if allowInsecureHTTP && origin.Scheme == "http" && net.ParseIP(host) != nil {
		allowed = true
	}
	if !allowed {
		return nil, errOrigin
	}
	port, err := canonicalPort(origin)
	if err != nil {
		return nil, err
	}
	if port != "" {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return &url.URL{Scheme: origin.Scheme, Host: host}, nil
}

// isOriginOnly 拒绝会改变凭证归属或请求路径的 URL 部分。
//
// Args:
//
//	origin: 已解析的 URL。
//
// Returns:
//
//	仅包含合法 origin 时返回 true。
func isOriginOnly(origin *url.URL) bool {
	return origin.Hostname() != "" && origin.User == nil && origin.Opaque == "" &&
		(origin.Path == "" || origin.Path == "/") && origin.RawPath == "" &&
		origin.RawQuery == "" && !origin.ForceQuery && origin.Fragment == ""
}

// secureScheme 将明文传输例外限制在 loopback，不接受伪装 localhost 的域名。
//
// Args:
//
//	scheme: URL 协议。
//	host: 小写主机名或 IP，不包含端口。
//
// Returns:
//
//	HTTPS 或明确的 HTTP loopback 返回 true。
func secureScheme(scheme, host string) bool {
	if scheme == "https" {
		return true
	}
	if scheme != "http" {
		return false
	}
	ip := net.ParseIP(host)
	return host == "localhost" || (ip != nil && ip.IsLoopback())
}

// canonicalPort 校验端口并移除协议的默认端口。
//
// Args:
//
//	origin: 已解析的地址。
//
// Returns:
//
//	非默认有效端口，或空字符串和错误。
//
// Errors:
//
//	返回端口超出 1–65535 或端口为空时的安全地址错误。
func canonicalPort(origin *url.URL) (string, error) {
	if strings.HasSuffix(origin.Host, ":") {
		return "", errOrigin
	}
	port := origin.Port()
	if port == "" {
		return "", nil
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return "", errOrigin
	}
	if (origin.Scheme == "https" && number == 443) || (origin.Scheme == "http" && number == 80) {
		return "", nil
	}
	return strconv.Itoa(number), nil
}
