package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestThreadsCreateBindsExistingOwnedResourcesWithoutStartingTurn(t *testing.T) {
	var calls atomic.Int32
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || r.URL.Path != "/api/v1/usight-agent/threads" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		checkInitialBinding(t, r)
		_, _ = fmt.Fprintf(w, `{"thread_id":%q,"kb_ids":[7],"library_file_ids":[%q],"metadata_version":"0","resource_window_state":"initial","title":"研究"}`, taskThread, taskTurn)
	}, "threads", "create", "--kb-id", "7", "--file-id", taskTurn)
	if code != 0 || calls.Load() != 1 || !strings.Contains(output, taskThread) || !strings.Contains(output, `"metadata_version":"0"`) {
		t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
	}
}

// checkInitialBinding 验证实际发送的初始资料标识，保持完整断言而非降低门禁。
func checkInitialBinding(t *testing.T, r *http.Request) {
	t.Helper()
	var body struct {
		KBIDs   []int64  `json:"seed_kb_ids"`
		FileIDs []string `json:"library_file_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Error(err)
		return
	}
	if len(body.KBIDs) != 1 || body.KBIDs[0] != 7 || len(body.FileIDs) != 1 || body.FileIDs[0] != taskTurn {
		t.Errorf("wrong binding %+v", body)
	}
}

func TestThreadsListReturnsNamesAndRealIDsWithExplicitSourceLimit(t *testing.T) {
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/v1/usight-agent/threads" || r.URL.Query().Get("view") != "archived" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		_, _ = fmt.Fprintf(w, `[{"thread_id":%q,"title":"同名会话","archived":true,"kb_ids":[],"library_file_ids":[],"resource_window_state":"locked","internal":"private-value"}]`, taskThread)
	}, "threads", "list", "--view", "archived", "--limit", "1")
	if code != 0 || !strings.Contains(output, taskThread) || !strings.Contains(output, "同名会话") || !strings.Contains(output, `"source_limit":200`) || strings.Contains(output, "private-") {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}

func TestThreadsCreateUnknownAcceptanceDoesNotRetry(t *testing.T) {
	var calls atomic.Int32
	code, output := runTaskCommand(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = fmt.Fprint(w, "lost response")
	}, "threads", "create")
	if code == 0 || calls.Load() != 1 || !strings.Contains(output, "acceptance_unknown") {
		t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
	}
}

func TestThreadsInvalidBindingNeverMakesRequest(t *testing.T) {
	for _, args := range [][]string{
		{"threads", "create", "--kb-id", "0"},
		{"threads", "create", "--file-id", "../bad"},
		{"threads", "create", "--title", strings.Repeat("字", 201)},
		{"threads", "list", "--view", "other"},
		{"threads", "list", "--limit", "201"},
	} {
		var calls atomic.Int32
		code, output := runTaskCommand(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) }, args...)
		if code == 0 || calls.Load() != 0 || !strings.Contains(output, "invalid_input") {
			t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
		}
	}
}
