package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/validflow/ccneed-cli/internal/api"
)

func TestKBPlatformsReadsCurrentServerListWithoutLocalPlatformEnum(t *testing.T) {
	var calls atomic.Int32
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/insightflow/available-platforms" || r.URL.RawQuery != "" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer test-only-value" || r.ContentLength > 0 {
			t.Error("platform lookup must use the selected account and have no request body")
		}
		_, _ = fmt.Fprint(w, `{"platforms":["twitter","wechat_channels","fixture_new_source"],"internal":"private-value"}`)
	}, "kb", "platforms")
	var result struct {
		Platforms []string `json:"platforms"`
	}
	if code != 0 || calls.Load() != 1 || json.Unmarshal([]byte(output), &result) != nil {
		t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
	}
	if !reflect.DeepEqual(result.Platforms, []string{"twitter", "wechat_channels", "fixture_new_source"}) || strings.Contains(output, "private-") {
		t.Fatalf("server platforms were filtered or private fields leaked: %s", output)
	}
}

func TestKBPlatformsValidatesResponseAndKeepsAnExplicitEmptyList(t *testing.T) {
	for _, scenario := range []struct {
		name string
		body string
		want string
		exit int
	}{
		{"empty", `{"platforms":[]}`, `"platforms":[]`, 0},
		{"missing", `{}`, `"code":"invalid_response"`, 1},
		{"null", `{"platforms":null}`, `"code":"invalid_response"`, 1},
		{"wrong type", `{"platforms":[1]}`, `"code":"invalid_response"`, 1},
		{"blank id", `{"platforms":[""]}`, `"code":"invalid_response"`, 1},
		{"invalid id", `{"platforms":["../private-value"]}`, `"code":"invalid_response"`, 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			code, output := runTaskCommand(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = fmt.Fprint(w, scenario.body)
			}, "kb", "platforms")
			if code != scenario.exit || !strings.Contains(output, scenario.want) || strings.Contains(output, "private-") {
				t.Fatalf("exit=%d output=%s", code, output)
			}
		})
	}
}

func TestKBPlatformsReportsServerFailureWithoutRetryOrFallback(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var calls atomic.Int32
			code, output := runTaskCommand(t, func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(status)
				_, _ = fmt.Fprint(w, `{"detail":"private-server-message"}`)
			}, "kb", "platforms")
			if code == 0 || calls.Load() != 1 || !json.Valid([]byte(output)) || strings.Contains(output, "platforms") || strings.Contains(output, "private-") {
				t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
			}
		})
	}
}

func TestKBPlatformsHelpAndExtraArgumentsNeverConnect(t *testing.T) {
	for _, scenario := range []struct {
		args []string
		exit int
	}{
		{[]string{"kb", "platforms", "--help"}, 0},
		{[]string{"kb", "platforms", "unexpected"}, 1},
	} {
		var calls atomic.Int32
		var output bytes.Buffer
		code := Run(context.Background(), scenario.args, Options{Stdout: &output, Connect: func(context.Context, string) (*api.Client, error) {
			calls.Add(1)
			return nil, &api.Error{Code: "unexpected_connection"}
		}})
		if code != scenario.exit || calls.Load() != 0 {
			t.Fatalf("args=%v exit=%d calls=%d output=%s", scenario.args, code, calls.Load(), output.String())
		}
	}
}
