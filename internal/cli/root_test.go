package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/validflow/ccneed-cli/internal/version"
)

func TestVersionCommandKeepsUpdateReminderOffStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := New(Options{Version: version.Info{Version: "0.1.0", Commit: "test-commit"}, Stdout: &stdout, Stderr: &stderr, CheckUpdate: func(context.Context, string) string { return "v0.2.0" }})
	command.SetArgs([]string{"version"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("version command: %v", err)
	}
	var result version.Info
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid machine output: %s", stdout.String())
	}
	if result.Version != "0.1.0" || result.Commit != "test-commit" {
		t.Fatalf("version=%+v", result)
	}
	if !strings.Contains(stderr.String(), "v0.2.0") || strings.Contains(stdout.String(), "update") {
		t.Fatalf("stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestHelpDoesNotCheckNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	calls := 0
	command := New(Options{Stdout: &stdout, Stderr: &stderr, CheckUpdate: func(context.Context, string) string { calls++; return "v0.2.0" }})
	command.SetArgs([]string{"--help"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if calls != 0 || !strings.Contains(stdout.String(), "ccneed") {
		t.Fatalf("help=%s network calls=%d", stdout.String(), calls)
	}
}

func TestUpdateCheckCanBeDisabled(t *testing.T) {
	var stdout, stderr bytes.Buffer
	calls := 0
	code := Run(context.Background(), []string{"--no-update-check", "version"}, Options{
		Version: version.Info{Version: "0.1.0"}, Stdout: &stdout, Stderr: &stderr,
		CheckUpdate: func(context.Context, string) string { calls++; return "v0.2.0" },
	})
	if code != 0 || calls != 0 || stderr.Len() != 0 || !json.Valid(stdout.Bytes()) {
		t.Fatalf("exit=%d checks=%d stdout=%s stderr=%s", code, calls, stdout.String(), stderr.String())
	}
}
