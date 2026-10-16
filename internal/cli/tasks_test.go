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

const taskThread = "00000000-0000-0000-0000-000000000111"
const taskTurn = "00000000-0000-0000-0000-000000000222"

func taskFixture(status string) string {
	return fmt.Sprintf(`{"thread_id":%q,"turn_id":%q,"conversation_turn_idx":1,"status":%q,"cloud_status":%q,"answer":"公开结论","artifacts":[],"execution_id":"private-execution","reasoning":"private-reasoning"}`, taskThread, taskTurn, status, status)
}

func runTaskCommand(t *testing.T, handler http.HandlerFunc, args ...string) (int, string) {
	t.Helper()
	server := httptest.NewServer(handler)
	defer server.Close()
	var stdout, stderr bytes.Buffer
	options := Options{Stdout: &stdout, Stderr: &stderr, Connect: func(_ context.Context, origin string) (*api.Client, error) {
		return api.New(origin, "test-only-value", nil)
	}}
	code := Run(context.Background(), append([]string{"--api-url", server.URL}, args...), options)
	return code, stdout.String()
}

func taskArgs(command string) []string {
	return []string{"tasks", command, "--thread-id", taskThread, "--turn-id", taskTurn}
}

func TestTasksGetUsesOwnedPublicSnapshot(t *testing.T) {
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		want := "/api/v1/usight-agent/threads/" + taskThread + "/turns/" + taskTurn
		if r.Method != "GET" || r.URL.Path != want || r.Header.Get("Authorization") != "Bearer test-only-value" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		_, _ = fmt.Fprint(w, taskFixture("completed"))
	}, taskArgs("get")...)
	if code != 0 || !strings.Contains(output, `"answer":"公开结论"`) || !json.Valid([]byte(output)) || strings.Contains(output, "private-") {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}

func TestTasksGetPaginatesOnlyRequestedTurn(t *testing.T) {
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("after_seq") != "7" {
			t.Errorf("unexpected progress query: %s", r.URL.RawQuery)
		}
		_, _ = fmt.Fprintf(w, `{"thread_id":%q,"turn_id":%q,"conversation_turn_idx":1,"status":"running","cloud_status":"running","artifacts":[],"progress":{"events":[{"seq":8,"at":"2026-10-16T00:00:00Z","stage":"analysis","state":"started","summary":"Analyzing data"}],"has_more":false,"next_after_seq":8}}`, taskThread, taskTurn)
	}, append(taskArgs("get"), "--limit", "2", "--after-seq", "7")...)
	if code != 0 || !strings.Contains(output, `"next_after_seq":8`) || !strings.Contains(output, `"web_url":`) || !strings.Contains(output, taskThread) {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}

func TestTasksWatchReturnsWaitingUserWithoutPollingAgain(t *testing.T) {
	var calls atomic.Int32
	body := fmt.Sprintf(`{"thread_id":%q,"turn_id":%q,"conversation_turn_idx":1,"status":"waiting_user","cloud_status":"paused","pending_action":{"action_id":"choice-one","action_type":"ask_user","reason":"选择平台","resume_schema":{"type":"object","required":["platform"]},"suggested":{"options":["rednote","douyin"]}},"artifacts":[],"progress":{"events":[{"seq":12,"stage":"work","state":"waiting_user","summary":"Work step waiting user."}],"has_more":false,"next_after_seq":12}}`, taskThread, taskTurn)
	code, output := runTaskCommand(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = fmt.Fprint(w, body)
	}, append(taskArgs("watch"), "--timeout", "1s")...)
	if code != 0 || calls.Load() != 1 || !strings.Contains(output, `"stop_reason":"waiting_user"`) || !strings.Contains(output, `"action_id":"choice-one"`) || !strings.Contains(output, `"next_after_seq":12`) {
		t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
	}
}

func TestTasksWatchTimeoutPreservesLastCloudStateWithoutWrites(t *testing.T) {
	var calls atomic.Int32
	body := fmt.Sprintf(`{"thread_id":%q,"turn_id":%q,"conversation_turn_idx":1,"status":"running","cloud_status":"running","artifacts":[]}`, taskThread, taskTurn)
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "GET" {
			t.Error("watch wrote to cloud")
		}
		_, _ = fmt.Fprint(w, body)
	}, append(taskArgs("watch"), "--timeout", "1s")...)
	if code != 0 || calls.Load() != 1 || !strings.Contains(output, `"stop_reason":"timeout"`) || !strings.Contains(output, `"status":"running"`) || !strings.Contains(output, taskTurn) {
		t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
	}
}

func TestTasksRejectWrongIdentityAndMalformedState(t *testing.T) {
	for _, body := range []string{
		strings.ReplaceAll(taskFixture("completed"), taskTurn, taskThread),
		taskFixture("not-a-status"),
		strings.ReplaceAll(taskFixture("completed"), `"completed"`, `"waiting_user"`),
	} {
		code, output := runTaskCommand(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, body) }, taskArgs("get")...)
		if code == 0 || !strings.Contains(output, `"code":"invalid_response"`) {
			t.Fatalf("exit=%d output=%s", code, output)
		}
	}
}

func TestTasksGetDoesNotExposeDeniedTurnResponse(t *testing.T) {
	code, output := runTaskCommand(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"detail":"private-turn-secret"}`)
	}, taskArgs("get")...)
	if code == 0 || !strings.Contains(output, `"code":"not_found"`) || strings.Contains(output, "private-turn-secret") {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}

func TestTasksValidateArgumentsBeforeConnection(t *testing.T) {
	for _, args := range [][]string{
		{"tasks", "get", "--thread-id", "../bad", "--turn-id", taskTurn},
		append(taskArgs("watch"), "--timeout", "0s"),
		append(taskArgs("watch"), "--timeout", "11m"),
		append(taskArgs("get"), "--limit", "0"),
		append(taskArgs("get"), "--before-seq", "7", "--after-seq", "8"),
		append(taskArgs("get"), "--after-seq", "-1"),
		append(taskArgs("get"), "--after-seq", "0"),
	} {
		var calls atomic.Int32
		var stdout, stderr bytes.Buffer
		code := Run(context.Background(), args, Options{Stdout: &stdout, Stderr: &stderr, Connect: func(context.Context, string) (*api.Client, error) {
			calls.Add(1)
			return nil, &api.Error{Code: "unexpected"}
		}})
		if code == 0 || calls.Load() != 0 || !strings.Contains(stdout.String(), `"code":"invalid_input"`) {
			t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), stdout.String())
		}
	}
}
