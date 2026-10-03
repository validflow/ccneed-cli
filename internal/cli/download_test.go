package cli

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func downloadArgs(output string) []string {
	return []string{"artifacts", "download", "--thread-id", taskThread, "--artifact-id", taskTurn, "--output", output}
}

func TestArtifactDownloadUsesExplicitPathAndIgnoresRemoteFilename(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "report.pdf")
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		want := "/api/v1/usight-agent/threads/" + taskThread + "/workdir/artifacts/" + taskTurn + "/content"
		if r.Method != "GET" || r.URL.Path != want {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="../../evil.pdf"`)
		_, _ = fmt.Fprint(w, "%PDF-test")
	}, downloadArgs(target)...)
	if code != 0 || !strings.Contains(output, `"size_bytes":9`) {
		t.Fatalf("exit=%d output=%s", code, output)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "%PDF-test" {
		t.Fatalf("saved=%q err=%v", data, err)
	}
	checkDownloadDirectory(t, directory, 1)
}

// checkDownloadDirectory 检查只有预期文件，既没有服务器文件名也没有残留临时文件。
func checkDownloadDirectory(t *testing.T, directory string, count int) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != count {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
}

func TestDownloadRefusesExistingTargetBeforeNetwork(t *testing.T) {
	target := filepath.Join(t.TempDir(), "report.pdf")
	if err := os.WriteFile(target, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	code, output := runTaskCommand(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) }, downloadArgs(target)...)
	if code == 0 || calls.Load() != 0 || !strings.Contains(output, "output_exists") {
		t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "original" {
		t.Fatalf("existing file changed: %q err=%v", data, err)
	}
}

func TestDownloadFailuresLeaveNoTargetOrTemporaryFile(t *testing.T) {
	for _, state := range []string{"not-found", "too-large", "incomplete"} {
		directory := t.TempDir()
		target := filepath.Join(directory, "report.pdf")
		var calls atomic.Int32
		code, output := runTaskCommand(t, func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			switch state {
			case "not-found":
				w.WriteHeader(http.StatusNotFound)
			case "too-large":
				w.Header().Set("Content-Length", "5242881")
			case "incomplete":
				w.Header().Set("Content-Length", "100")
				_, _ = fmt.Fprint(w, "x")
			}
		}, downloadArgs(target)...)
		if code == 0 || calls.Load() != 1 {
			t.Fatalf("state=%s exit=%d calls=%d output=%s", state, code, calls.Load(), output)
		}
		checkDownloadDirectory(t, directory, 0)
	}
}

func TestArtifactReadReturnsPublicTextInsteadOfRawTerminalBytes(t *testing.T) {
	code, output := runTaskCommand(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		_, _ = fmt.Fprint(w, "# 公开结论\n")
	}, "artifacts", "read", "--thread-id", taskThread, "--artifact-id", taskTurn)
	if code != 0 || !strings.Contains(output, "公开结论") || !strings.Contains(output, `"text":`) {
		t.Fatalf("exit=%d output=%s", code, output)
	}
}
