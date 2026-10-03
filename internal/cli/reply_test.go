package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestTasksReplySendsStructuredAnswerForSameTurn(t *testing.T) {
	answer := filepath.Join(t.TempDir(), "answer.json")
	if err := os.WriteFile(answer, []byte(`{"answers":{"platform":"rednote"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || r.URL.Path != "/api/v1/usight-agent/threads/"+taskThread+"/resume" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		checkReplyBody(t, r)
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprintf(w, `{"thread_id":%q,"status":"active","next_turn_idx":2}`, taskThread)
	}, append(taskArgs("reply"), "--action-id", "choice-one", "--answer-file", answer)...)
	if code != 0 || calls.Load() != 1 || !strings.Contains(output, `"status":"queued"`) || !strings.Contains(output, taskTurn) {
		t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
	}
}

// checkReplyBody 保留 HTTP 边界的身份和结构化回答断言，不终止服务端协程。
func checkReplyBody(t *testing.T, r *http.Request) {
	t.Helper()
	var body struct {
		TurnID   string                     `json:"turn_id"`
		ActionID string                     `json:"action_id"`
		Payload  map[string]json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("invalid body: %v", err)
		return
	}
	if body.TurnID != taskTurn || body.ActionID != "choice-one" || string(body.Payload["answers"]) != `{"platform":"rednote"}` {
		t.Errorf("wrong structured answer %+v", body)
	}
}

func TestTasksReplyDoesNotRetryUnknownAcceptance(t *testing.T) {
	answer := filepath.Join(t.TempDir(), "answer.json")
	if err := os.WriteFile(answer, []byte(`{"decision":"approved"}`), 0600); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = fmt.Fprint(w, "not-valid-json")
	}, append(taskArgs("reply"), "--action-id", "choice-one", "--answer-file", answer)...)
	if code == 0 || calls.Load() != 1 || !strings.Contains(output, `"code":"acceptance_unknown"`) {
		t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
	}
}

func TestTasksReplyRejectsMalformedOrOversizedAnswerLocally(t *testing.T) {
	for _, input := range []string{"plain text", `null`, `[]`, `{"reply":"x"} {}`, `{"reply":"` + strings.Repeat("x", 65536) + `"}`} {
		answer := filepath.Join(t.TempDir(), "answer.json")
		if err := os.WriteFile(answer, []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
		var calls atomic.Int32
		code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }, append(taskArgs("reply"), "--action-id", "choice-one", "--answer-file", answer)...)
		if code == 0 || calls.Load() != 0 || !strings.Contains(output, `"code":"invalid_input"`) {
			t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
		}
	}
}
