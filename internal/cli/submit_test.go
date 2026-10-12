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

const submitRequest = "00000000-0000-0000-0000-000000000333"

// submitRunner 每次重建命令树，共用临时非秘密恢复目录，模拟独立 CLI 调用。
func submitRunner(t *testing.T, handler http.HandlerFunc) func(...string) (int, string) {
	t.Helper()
	directory := t.TempDir()
	t.Setenv("AppData", directory)
	t.Setenv("XDG_CONFIG_HOME", directory)
	// macOS 的 UserConfigDir 读取 HOME；仅在本测试作用域内隔离并自动恢复。
	t.Setenv("HOME", directory)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return func(args ...string) (int, string) {
		var stdout, stderr bytes.Buffer
		options := Options{Stdout: &stdout, Stderr: &stderr, Connect: func(_ context.Context, origin string) (*api.Client, error) {
			return api.New(origin, "test-only-value", nil)
		}}
		code := Run(context.Background(), append([]string{"--api-url", server.URL}, args...), options)
		return code, stdout.String()
	}
}

func submitArgs(message string) []string {
	return []string{"run", message, "--thread-id", taskThread, "--request-id", submitRequest}
}

func TestRunSubmitsOneTurnAndReturnsPublicAcknowledgement(t *testing.T) {
	var calls atomic.Int32
	run := submitRunner(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		checkSubmitBody(t, r, submitRequest)
		_, _ = fmt.Fprintf(w, `{"thread_id":%q,"turn_id":%q,"conversation_turn_idx":1,"status":"pending","existing":false,"execution_id":"private-execution"}`, taskThread, taskTurn)
	})
	code, output := run(submitArgs("分析数据")...)
	if code != 0 || calls.Load() != 1 || !strings.Contains(output, taskTurn) || !strings.Contains(output, submitRequest) || strings.Contains(output, "private-") {
		t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
	}
}

// checkSubmitBody 校验真实 POST 的原始会话、幂等标识和输入，不模拟客户端。
func checkSubmitBody(t *testing.T, r *http.Request, requestID string) {
	t.Helper()
	if r.Method != "POST" || r.URL.Path != "/api/v1/usight-agent/threads/"+taskThread+"/turns" {
		t.Errorf("unexpected request %s %s", r.Method, r.URL)
	}
	var body struct {
		Message   string `json:"message"`
		RequestID string `json:"client_request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("invalid body: %v", err)
		return
	}
	if body.Message != "分析数据" || body.RequestID != requestID {
		t.Errorf("wrong submission %+v", body)
	}
}

func TestRunResponseLossReusesSameRequestAcrossInvocations(t *testing.T) {
	var calls atomic.Int32
	run := submitRunner(t, func(w http.ResponseWriter, r *http.Request) {
		checkSubmitBody(t, r, submitRequest)
		if calls.Add(1) == 1 {
			_, _ = fmt.Fprint(w, "lost acknowledgement")
			return
		}
		_, _ = fmt.Fprintf(w, `{"thread_id":%q,"turn_id":%q,"conversation_turn_idx":1,"status":"running","existing":true}`, taskThread, taskTurn)
	})
	code, output := run(submitArgs("分析数据")...)
	if code == 0 || calls.Load() != 1 || !strings.Contains(output, "acceptance_unknown") || !strings.Contains(output, submitRequest) || !strings.Contains(output, taskThread) {
		t.Fatalf("first exit=%d calls=%d output=%s", code, calls.Load(), output)
	}
	code, output = run(submitArgs("分析数据")...)
	if code != 0 || calls.Load() != 2 || !strings.Contains(output, `"existing":true`) || !strings.Contains(output, taskTurn) {
		t.Fatalf("retry exit=%d calls=%d output=%s", code, calls.Load(), output)
	}
}

func TestRunRejectsChangedPromptForSameRequestBeforeSecondWrite(t *testing.T) {
	var calls atomic.Int32
	run := submitRunner(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = fmt.Fprintf(w, `{"thread_id":%q,"turn_id":%q,"conversation_turn_idx":1,"status":"pending","existing":false}`, taskThread, taskTurn)
	})
	if code, output := run(submitArgs("分析数据")...); code != 0 {
		t.Fatalf("initial exit=%d output=%s", code, output)
	}
	code, output := run(submitArgs("另一个需求")...)
	if code == 0 || calls.Load() != 1 || !strings.Contains(output, "request_id_conflict") {
		t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
	}
}

func TestRunInvalidArgumentsNeverConnect(t *testing.T) {
	for _, args := range [][]string{
		submitArgs("  "),
		submitArgs(strings.Repeat("字", 20001)),
		{"run", "分析数据", "--thread-id", "../bad"},
		{"run", "分析数据", "--thread-id", taskThread, "--request-id", "../bad"},
	} {
		var calls atomic.Int32
		var stdout, stderr bytes.Buffer
		code := Run(context.Background(), args, Options{Stdout: &stdout, Stderr: &stderr, Connect: func(context.Context, string) (*api.Client, error) {
			calls.Add(1)
			return nil, &api.Error{Code: "unexpected"}
		}})
		if code == 0 || calls.Load() != 0 || !strings.Contains(stdout.String(), "invalid_input") {
			t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), stdout.String())
		}
	}
}
