package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestKBListDisplaysSelectionAndCollectionStatus(t *testing.T) {
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/usight-agent/kbs" || r.Method != "GET" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		_, _ = fmt.Fprint(w, `[{"id":7,"name":"社媒","can_select":false,"active_run_status":"running","post_count":10,"disabled_reason":"collecting","internal":"private-value"}]`)
	}, "kb", "list")
	if code != 0 || !strings.Contains(output, `"id":7`) || !strings.Contains(output, `"can_select":false`) || !strings.Contains(output, `"active_run_status":"running"`) || strings.Contains(output, "private-") {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}

func TestKBCreationReturnsWebsiteHandoffWithoutStartingCollection(t *testing.T) {
	var calls atomic.Int32
	code, output := runTaskCommand(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) }, "kb", "create")
	if code != 0 || calls.Load() != 0 || !strings.Contains(output, "website_required") || !strings.Contains(output, "/insightflow/kb") {
		t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
	}
}

func TestFilesListReturnsExistingFileIDsForBinding(t *testing.T) {
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/usight-agent/library/files" || r.Method != "GET" || r.URL.Query().Get("limit") != "20" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		_, _ = fmt.Fprintf(w, `[{"library_file_id":%q,"display_name":"posts.csv","size_bytes":12,"status":"ready","object_key":"private-object"}]`, taskTurn)
	}, "files", "list", "--limit", "20")
	if code != 0 || !strings.Contains(output, taskTurn) || !strings.Contains(output, "posts.csv") || strings.Contains(output, "private-") {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}

func TestThreadsResourcesReadsServerWindowAndRevision(t *testing.T) {
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/v1/usight-agent/threads/"+taskThread+"/resources" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		_, _ = fmt.Fprintf(w, `{"thread_id":%q,"kb_ids":[7],"library_file_ids":[],"metadata_version":"3","resource_window_state":"report_window"}`, taskThread)
	}, "threads", "resources", "--thread-id", taskThread)
	if code != 0 || !strings.Contains(output, `"metadata_version":"3"`) || !strings.Contains(output, "report_window") {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}

func TestThreadsAppendUsesExactRevisionAndOnlyAdds(t *testing.T) {
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/usight-agent/threads/"+taskThread+"/resources/append" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		checkAppendBody(t, r)
		_, _ = fmt.Fprintf(w, `{"kb_ids":[7],"library_file_ids":[%q],"added_kb_ids":[],"added_library_file_ids":[%q],"metadata_version":"4","resource_window_state":"report_window","kb_content_scope":"all"}`, taskTurn, taskTurn)
	}, "threads", "append", "--thread-id", taskThread, "--file-id", taskTurn, "--expected-version", "3")
	if code != 0 || !strings.Contains(output, `"metadata_version":"4"`) {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}

// checkAppendBody 保持资料增加与 CAS 输入的完整 HTTP 契约断言。
func checkAppendBody(t *testing.T, r *http.Request) {
	t.Helper()
	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Error(err)
		return
	}
	if string(body["expected_metadata_version"]) != `"3"` || string(body["add_library_file_ids"]) != fmt.Sprintf("[%q]", taskTurn) {
		t.Errorf("wrong append %+v", body)
	}
	if _, exists := body["remove"]; exists {
		t.Error("append must not remove resources")
	}
}

func TestArtifactsListContainsOnlyPublicFindingsAndReports(t *testing.T) {
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/v1/usight-agent/threads/"+taskThread+"/artifacts" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		_, _ = fmt.Fprintf(w, `{"thread_id":%q,"artifacts":[{"artifact_id":%q,"name":"报告.pdf","artifact_type":"report","size_bytes":12,"content_type":"application/pdf","locator":"private-locator"}],"has_more":false}`, taskThread, taskTurn)
	}, "artifacts", "list", "--thread-id", taskThread)
	if code != 0 || !strings.Contains(output, taskTurn) || !strings.Contains(output, "报告.pdf") || strings.Contains(output, "private-") {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}

func TestResourcesInvalidParametersNeverWrite(t *testing.T) {
	for _, args := range [][]string{
		{"threads", "append", "--thread-id", taskThread, "--file-id", taskTurn},
		{"threads", "append", "--thread-id", taskThread, "--expected-version", "3"},
		{"threads", "resources", "--thread-id", "bad"},
		{"files", "list", "--limit", "501"},
		{"artifacts", "list", "--thread-id", "../bad"},
	} {
		var calls atomic.Int32
		code, output := runTaskCommand(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) }, args...)
		if code == 0 || calls.Load() != 0 || !strings.Contains(output, "invalid_input") {
			t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
		}
	}
}
