package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestFilesUploadStreamsMultipartAndReturnsReusableLibraryID(t *testing.T) {
	content := "id,text\n1,x\n"
	filename := filepath.Join(t.TempDir(), "posts.csv")
	if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		checkUploadedFile(t, r, content)
		_, _ = fmt.Fprintf(w, `{"library_file_id":%q,"display_name":"posts.csv","size_bytes":%d,"status":"ready","content_type":"text/csv","object_key":"private-key"}`, taskTurn, len(content))
	}, "files", "upload", filename)
	if code != 0 || calls.Load() != 1 || !strings.Contains(output, taskTurn) || strings.Contains(output, "private-") {
		t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
	}
}

// checkUploadedFile 验证真正的 multipart 上传内容与认证请求，避免以 JSON 假冒文件上传。
func checkUploadedFile(t *testing.T, r *http.Request, content string) {
	t.Helper()
	if r.Method != "POST" || r.URL.Path != "/api/v1/usight-agent/library/files" {
		t.Errorf("unexpected request %s %s", r.Method, r.URL)
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		t.Error(err)
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(file)
	if err != nil || string(data) != content || header.Filename != "posts.csv" {
		t.Errorf("wrong upload name=%s bytes=%q err=%v", header.Filename, data, err)
	}
}

func TestUploadUnknownAcceptanceNeverRetries(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "posts.csv")
	if err := os.WriteFile(filename, []byte("a,b\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	code, output := runTaskCommand(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = fmt.Fprint(w, "lost response")
	}, "files", "upload", filename)
	if code == 0 || calls.Load() != 1 || !strings.Contains(output, "acceptance_unknown") {
		t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
	}
}

func TestUploadRejectsInvalidLocalSourcesWithoutNetwork(t *testing.T) {
	directory := t.TempDir()
	oversized := filepath.Join(directory, "oversized.csv")
	file, err := os.Create(oversized)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(100*1024*1024 + 1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{directory, filepath.Join(directory, "missing.csv"), oversized} {
		var calls atomic.Int32
		code, output := runTaskCommand(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) }, "files", "upload", path)
		if code == 0 || calls.Load() != 0 || !strings.Contains(output, "invalid_file") {
			t.Fatalf("exit=%d calls=%d output=%s", code, calls.Load(), output)
		}
	}
}
