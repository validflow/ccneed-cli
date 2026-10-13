package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

// checkInitialBinding verifies that a new conversation binds only the selected resources.
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
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/usight-agent/threads" || r.URL.Query().Get("view") != "archived" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		_, _ = fmt.Fprintf(w, `[{"thread_id":%q,"title":"Existing conversation","archived":true,"kb_ids":[],"library_file_ids":[],"resource_window_state":"locked","internal":"private-value"}]`, taskThread)
	}, "threads", "list", "--view", "archived", "--limit", "1")
	if code != 0 || !strings.Contains(output, taskThread) || !strings.Contains(output, "Existing conversation") || !strings.Contains(output, `"source_limit":200`) || strings.Contains(output, "private-") {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}

func TestThreadsCreateIsNotACommand(t *testing.T) {
	var calls atomic.Int32
	code, output := runTaskCommand(t, func(http.ResponseWriter, *http.Request) {
		calls.Add(1)
	}, "threads", "create", "--kb-id", "7")
	if code == 0 || calls.Load() != 0 {
		t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
	}
}

func TestThreadsInvalidListNeverMakesRequest(t *testing.T) {
	for _, args := range [][]string{
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
