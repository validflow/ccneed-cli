package update

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// releaseTransport 只替换公开发行请求的网络边界。
type releaseTransport func(*http.Request) (*http.Response, error)

func (f releaseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGitHubSourceIsFixedAndAnonymous(t *testing.T) {
	source := NewGitHubSource(releaseTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://api.github.com/repos/validflow/ccneed-cli/releases/latest" || r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected source request %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"tag_name":"v0.2.0","draft":false,"prerelease":false}`)), Header: make(http.Header)}, nil
	}))
	got, err := source.Latest(context.Background())
	if err != nil || got != "v0.2.0" {
		t.Fatalf("tag=%q error=%v", got, err)
	}
}

func TestGitHubSourceRejectsBadReleases(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
	}{
		{"not public", 404, `{"message":"private"}`},
		{"redirect", 302, `{}`},
		{"draft", 200, `{"tag_name":"v9.0.0","draft":true}`},
		{"prerelease", 200, `{"tag_name":"v9.0.0","prerelease":true}`},
		{"bad tag", 200, `{"tag_name":"garbage"}`},
		{"bad json", 200, `not json`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source := NewGitHubSource(releaseTransport(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tt.status, Body: io.NopCloser(strings.NewReader(tt.body)), Header: make(http.Header)}, nil
			}))
			if tag, err := source.Latest(context.Background()); err == nil || tag != "" {
				t.Fatalf("accepted %s: tag=%s error=%v", tt.name, tag, err)
			}
		})
	}
}

func TestGitHubSourceBoundsResponse(t *testing.T) {
	source := NewGitHubSource(releaseTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"tag_name":"v0.2.0","padding":"%s"}`, strings.Repeat("x", 65536)))), Header: make(http.Header)}, nil
	}))
	if _, err := source.Latest(context.Background()); err == nil {
		t.Fatal("accepted oversized release data")
	}
}
