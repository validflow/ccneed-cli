package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestAppendSendsExplicitEmptyListsInsteadOfNull(t *testing.T) {
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			KBIDs   []int64  `json:"add_kb_ids"`
			FileIDs []string `json:"add_library_file_ids"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || body.KBIDs == nil || body.FileIDs == nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		_, _ = fmt.Fprintf(w, `{"kb_ids":[7],"library_file_ids":[%q],"metadata_version":"4","resource_window_state":"report_window","added_kb_ids":[],"added_library_file_ids":[%q]}`, taskTurn, taskTurn)
	}, "threads", "append", "--thread-id", taskThread, "--file-id", taskTurn, "--expected-version", "3")
	if code != 0 || !strings.Contains(output, `"metadata_version":"4"`) {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}

func TestKBGetUsesNumericIdentityWithoutSelectingByName(t *testing.T) {
	code, output := runTaskCommand(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `[{"id":1,"name":"同名","can_select":false,"post_count":2},{"id":7,"name":"同名","can_select":true,"post_count":10}]`)
	}, "kb", "get", "--kb-id", "7")
	if code != 0 || !strings.Contains(output, `"id":7`) || !strings.Contains(output, `"post_count":10`) || strings.Contains(output, `"id":1`) {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}
