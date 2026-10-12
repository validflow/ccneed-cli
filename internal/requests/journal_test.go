package requests

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/validflow/ccneed-cli/internal/api"
)

const testThread = "00000000-0000-0000-0000-000000000111"
const testRequest = "00000000-0000-0000-0000-000000000222"

func TestNewIDUsesIndependentVersionFourUUIDs(t *testing.T) {
	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	values := map[string]bool{}
	for range 20 {
		id, err := NewID()
		if err != nil || !pattern.MatchString(id) || values[id] {
			t.Fatalf("id=%q err=%v duplicate=%v", id, err, values[id])
		}
		values[id] = true
	}
}

func TestPrepareIsImmutableAndNeverPersistsPrompt(t *testing.T) {
	directory := t.TempDir()
	first, err := Prepare(directory, "https://example.test", testThread, testRequest, "private-input")
	if err != nil {
		t.Fatal(err)
	}
	again, err := Prepare(directory, "https://example.test", testThread, testRequest, "private-input")
	if err != nil || again != first {
		t.Fatalf("record changed: %+v err=%v", again, err)
	}
	_, err = Prepare(directory, "https://example.test", testThread, testRequest, "different-input")
	requireCode(t, err, "request_id_conflict")
	data := readSingleJournal(t, directory)
	if strings.Contains(string(data), "private-input") || strings.Contains(string(data), "different-input") {
		t.Fatal("journal persisted private input")
	}
	references, err := List(directory, "https://example.test", testThread)
	if err != nil || len(references) != 1 || references[0].RequestID != testRequest {
		t.Fatalf("references=%+v err=%v", references, err)
	}
}

// readSingleJournal 校验只有一份真实请求记录，并读取它供隐私断言使用。
func readSingleJournal(t *testing.T, directory string) []byte {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%d err=%v", len(entries), err)
	}
	data, err := os.ReadFile(filepath.Join(directory, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestJournalRejectsDamagedRecordWithoutOverwriting(t *testing.T) {
	directory := t.TempDir()
	if _, err := Prepare(directory, "https://example.test", testThread, testRequest, "input"); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(directory, entries[0].Name())
	if err := os.WriteFile(filename, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = Prepare(directory, "https://example.test", testThread, testRequest, "input")
	requireCode(t, err, "recovery_unavailable")
	_, err = List(directory, "https://example.test", testThread)
	requireCode(t, err, "recovery_unavailable")
	data, err := os.ReadFile(filename)
	if err != nil || string(data) != "{" {
		t.Fatalf("damaged evidence changed err=%v", err)
	}
}

func TestListDoesNotCreateDirectoryOrReadAnotherOrigin(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "not-created")
	references, err := List(directory, "https://example.test", testThread)
	if err != nil || len(references) != 0 {
		t.Fatalf("references=%+v err=%v", references, err)
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatalf("read created directory: %v", err)
	}
	if _, err := Prepare(directory, "https://other.test", testThread, testRequest, "input"); err != nil {
		t.Fatal(err)
	}
	references, err = List(directory, "https://example.test", testThread)
	if err != nil || len(references) != 0 {
		t.Fatalf("cross-origin records=%+v err=%v", references, err)
	}
}

func TestPrepareRejectsFileInsteadOfDirectory(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "ordinary-file")
	if err := os.WriteFile(directory, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := Prepare(directory, "https://example.test", testThread, testRequest, "input")
	requireCode(t, err, "recovery_unavailable")
	data, err := os.ReadFile(directory)
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing file changed err=%v", err)
	}
}

// requireCode 校验真实存储错误的稳定机器码，不依赖错误文案。
func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	var response *api.Error
	if !errors.As(err, &response) || response.Code != code {
		t.Fatalf("error=%v want=%s", err, code)
	}
}
