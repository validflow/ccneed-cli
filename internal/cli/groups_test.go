package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

func TestUnknownSubcommandsFailWithoutNetwork(t *testing.T) {
	for _, group := range []string{"auth", "account", "tasks", "threads", "kb", "files", "artifacts"} {
		t.Run(group, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			calls := 0
			code := Run(context.Background(), []string{group, "not-a-command", "unexpected"}, Options{
				Stdout: &stdout, Stderr: &stderr,
				CheckUpdate: func(context.Context, string) string { calls++; return "" },
			})
			if code == 0 || calls != 0 || !json.Valid(stdout.Bytes()) {
				t.Fatalf("exit=%d checks=%d output=%s", code, calls, stdout.String())
			}
		})
	}
}

func TestGroupHelpDoesNotCheckNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	calls := 0
	code := Run(context.Background(), []string{"tasks"}, Options{
		Stdout: &stdout, Stderr: &stderr,
		CheckUpdate: func(context.Context, string) string { calls++; return "" },
	})
	if code != 0 || calls != 0 || stdout.Len() == 0 {
		t.Fatalf("exit=%d checks=%d output=%s", code, calls, stdout.String())
	}
}
