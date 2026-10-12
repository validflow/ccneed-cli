package cli

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/validflow/ccneed-cli/internal/requests"
)

func TestTasksListRecoversOriginalTurnWithStableCursor(t *testing.T) {
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/v1/usight-agent/threads/"+taskThread+"/turns" || r.URL.Query().Get("client_request_id") != submitRequest || r.URL.Query().Get("before_turn_index") != "4" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		_, _ = fmt.Fprintf(w, `{"thread_id":%q,"turns":[{"turn_id":%q,"conversation_turn_idx":3,"client_request_id":%q,"cloud_status":"completed","execution_id":"private-loop"}],"next_before_turn_index":3}`, taskThread, taskTurn, submitRequest)
	}, "tasks", "list", "--thread-id", taskThread, "--request-id", submitRequest, "--before-turn-index", "4", "--limit", "1")
	if code != 0 || !strings.Contains(output, taskTurn) || !strings.Contains(output, `"next_before_turn_index":3`) || strings.Contains(output, "private-") {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}

func TestTasksListRejectsCrossThreadResponse(t *testing.T) {
	code, output := runTaskCommand(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{"thread_id":%q,"turns":[],"next_before_turn_index":null}`, taskTurn)
	}, "tasks", "list", "--thread-id", taskThread)
	if code == 0 || !strings.Contains(output, "invalid_response") {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}

func TestRequestsListFindsOriginalIDsButNeverOtherOrigin(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("AppData", directory)
	t.Setenv("XDG_CONFIG_HOME", directory)
	// macOS 不读取 XDG_CONFIG_HOME，测试必须同样隔离其用户目录。
	t.Setenv("HOME", directory)
	root, err := requests.DefaultDirectory()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := requests.Prepare(root, "https://ccneed.ai", taskThread, submitRequest, "private-message"); err != nil {
		t.Fatal(err)
	}
	if _, err := requests.Prepare(root, "https://staging.example.test", taskThread, taskTurn, "private-message"); err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	code := Run(context.Background(), []string{"requests", "list", "--thread-id", taskThread}, Options{Stdout: &output, Stderr: &strings.Builder{}})
	if code != 0 || !strings.Contains(output.String(), submitRequest) || strings.Contains(output.String(), taskTurn) || strings.Contains(output.String(), "private-message") || strings.Contains(output.String(), "input_digest") {
		t.Fatalf("exit=%d output=%s", code, output.String())
	}
}
