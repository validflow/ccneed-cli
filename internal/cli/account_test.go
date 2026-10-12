package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/validflow/ccneed-cli/internal/api"
)

func TestAccountCreditsCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/v1/payment/credits" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		_, _ = fmt.Fprint(w, `{"left_credits":42}`)
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	connect := func(_ context.Context, origin string) (*api.Client, error) {
		if origin != server.URL {
			t.Errorf("origin switched to %s", origin)
		}
		return api.New(origin, "test-only-value", nil)
	}
	code := Run(context.Background(), []string{"--api-url", server.URL, "account", "credits"}, Options{Stdout: &stdout, Stderr: &stderr, Connect: connect})
	if code != 0 || !strings.Contains(stdout.String(), `"left_credits":42`) {
		t.Fatalf("exit=%d output=%s", code, stdout.String())
	}
	var result map[string]int64
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
}

func TestAccountCommandsFailClosedWithoutLogin(t *testing.T) {
	for _, args := range [][]string{{"account", "credits"}, {"account", "overview"}, {"account", "history"}} {
		var stdout, stderr bytes.Buffer
		code := Run(context.Background(), args, Options{Stdout: &stdout, Stderr: &stderr})
		if code == 0 || !strings.Contains(stdout.String(), `"code":"not_authenticated"`) {
			t.Fatalf("exit=%d output=%s", code, stdout.String())
		}
	}
}

func TestAccountInvalidOriginAndLimitDoNotConnect(t *testing.T) {
	for _, args := range [][]string{{"--api-url", "http://remote.example", "account", "credits"}, {"account", "history", "--limit", "201"}} {
		var calls atomic.Int32
		var stdout, stderr bytes.Buffer
		code := Run(context.Background(), args, Options{Stdout: &stdout, Stderr: &stderr, Connect: func(context.Context, string) (*api.Client, error) {
			calls.Add(1)
			return nil, &api.Error{Code: "unexpected"}
		}})
		if code == 0 || calls.Load() != 0 {
			t.Fatalf("exit=%d connections=%d output=%s", code, calls.Load(), stdout.String())
		}
	}
}

// accountReadHandler 替换云端边界，命令、账号服务和 HTTP 传输仍使用真实对象。
func accountReadHandler(t *testing.T, calls *atomic.Int32) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer test-only-value" {
			t.Error("request is not an authenticated read")
		}
		switch r.URL.Path {
		case "/api/v1/payment/credits":
			_, _ = fmt.Fprint(w, `{"left_credits":42}`)
		case "/api/v1/redemption/subscriptions":
			_, _ = fmt.Fprint(w, `{"subscriptions":[]}`)
		case "/api/v1/payment/credits/list":
			_, _ = fmt.Fprint(w, `[{"trans_no":"first","credits":-2},{"trans_no":"second","credits":-3}]`)
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func TestAccountOverviewAndHistoryCommands(t *testing.T) {
	for _, test := range []struct {
		name     string
		args     []string
		want     string
		requests int32
	}{
		{"overview", []string{"account", "overview"}, `"subscriptions":[]`, 2},
		{"history", []string{"account", "history", "--limit", "1"}, `"source_limit":200`, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(accountReadHandler(t, &calls))
			defer server.Close()
			var stdout, stderr bytes.Buffer
			options := Options{Stdout: &stdout, Stderr: &stderr, Connect: func(_ context.Context, origin string) (*api.Client, error) {
				return api.New(origin, "test-only-value", nil)
			}}
			args := append([]string{"--api-url", server.URL}, test.args...)
			code := Run(context.Background(), args, options)
			if code != 0 || calls.Load() != test.requests || !strings.Contains(stdout.String(), test.want) || strings.Contains(stdout.String(), "second") {
				t.Fatalf("exit=%d requests=%d output=%s", code, calls.Load(), stdout.String())
			}
			if !json.Valid(stdout.Bytes()) || stderr.Len() != 0 {
				t.Fatalf("invalid output streams: stdout=%s stderr=%s", stdout.String(), stderr.String())
			}
		})
	}
}

func TestAccountServiceErrorKeepsCodeWithoutLeakingResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprint(w, `{"detail":"private-server-value"}`)
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--api-url", server.URL, "account", "credits"}, Options{
		Stdout: &stdout, Stderr: &stderr,
		Connect: func(_ context.Context, origin string) (*api.Client, error) {
			return api.New(origin, "test-only-value", nil)
		},
	})
	if code == 0 || !strings.Contains(stdout.String(), `"code":"forbidden"`) || !strings.Contains(stdout.String(), `"http_status":403`) {
		t.Fatalf("exit=%d output=%s", code, stdout.String())
	}
	if strings.Contains(stdout.String()+stderr.String(), "private-server-value") {
		t.Fatal("service response leaked")
	}
}
