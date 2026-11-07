package cli

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKBPlanSubmitsWithoutStarting(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "plan.json")
	raw := `{"research_requirement":"品牌反馈","kb_name":"品牌","platforms":[{"platform":"weibo","quota":300,"keywords":[{"keyword":"品牌","tier":1,"keyword_type":"entity"}]}]}`
	if err := os.WriteFile(filename, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/usight-agent/kbs/plan" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["research_requirement"] != "品牌反馈" {
			t.Errorf("lost requirement: %v", body)
		}
		_, _ = fmt.Fprint(w, `{"status":"ready","plan_token":"signed","plan_version":"v1","research_requirement":"品牌反馈","platforms":[{"platform":"weibo","quota":300,"entity_keywords":["品牌"],"other_keywords":[]}],"search_plan":[],"total_quota":300,"estimated_credits":300,"balance":500,"adjustments":[],"expires_at":"2099-11-06T07:30:25Z"}`)
	}, "kb", "plan", "--plan-file", filename)
	if code != 0 || !strings.Contains(output, `"plan_token":"signed"`) {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}

func TestKBStartUsesOnlyPreparedTokenAndRequestID(t *testing.T) {
	filename, token := preparedPlanFile(t)
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/usight-agent/kbs/start" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body) != 2 || body["plan_token"] != token || body["request_id"] != taskTurn {
			t.Errorf("unexpected body %v", body)
		}
		_, _ = fmt.Fprint(w, `{"kb_id":7,"task_run_id":9,"status":"queued"}`)
	}, "kb", "start", "--plan-file", filename, "--request-id", taskTurn)
	if code != 0 || !strings.Contains(output, `"task_run_id":9`) {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}

func preparedPlanFile(t *testing.T) (string, string) {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "prepared.json")
	prepared := map[string]any{"status": "ready", "plan_version": "v1", "expires_at": "2099-11-06T07:30:25Z"}
	payload, err := json.Marshal(map[string]any{"plan": prepared})
	if err != nil {
		t.Fatal(err)
	}
	token := "eyJhbGciOiJIUzI1NiJ9." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
	prepared["plan_token"] = token
	data, err := json.Marshal(prepared)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, data, 0600); err != nil {
		t.Fatal(err)
	}
	return filename, token
}

func TestKBPlanReturnsValidationErrors(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(filename, []byte(`{"research_requirement":"品牌"}`), 0600); err != nil {
		t.Fatal(err)
	}
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"status":"invalid_plan","errors":[{"field":"platforms.0.quota","message":"quota is too small"}]}`)
	}, "kb", "plan", "--plan-file", filename)
	if code != 0 || !strings.Contains(output, `"status":"invalid_plan"`) || !strings.Contains(output, `platforms.0.quota`) {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}

func TestKBExportDoesNotOverwrite(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "data.json")
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/v1/usight-agent/kbs/7/export" || r.URL.Query().Get("format") != "json" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="ccneed-kb-7-brand-20261106T073025Z.json"`)
		w.Header().Set("X-CCNeed-KB-Name", "brand")
		w.Header().Set("X-CCNeed-Exported-At", "2026-11-06T07:30:25Z")
		_, _ = fmt.Fprint(w, `[{"id":1}]`)
	}
	code, output := runTaskCommand(t, handler, "kb", "export", "--kb-id", "7", "--format", "json", "--output", filename)
	if code != 0 {
		t.Fatalf("exit=%d output=%s", code, output)
	}
	data, err := os.ReadFile(filename)
	if err != nil || string(data) != `[{"id":1}]` || !strings.Contains(output, `"kb_id":7`) {
		t.Fatalf("file=%s err=%v output=%s", data, err, output)
	}
	code, output = runTaskCommand(t, handler, "kb", "export", "--kb-id", "7", "--format", "json", "--output", filename)
	if code == 0 {
		t.Fatalf("overwrite accepted: %s", output)
	}
	after, err := os.ReadFile(filename)
	if err != nil || string(after) != string(data) {
		t.Fatal("existing export changed")
	}
}
