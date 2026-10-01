package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/validflow/ccneed-cli/internal/version"
)

func TestRunVersionAndErrorExitCodes(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		code int
	}{
		{"version", []string{"version"}, 0},
		{"unknown command", []string{"private-unknown-input"}, 1},
		{"invalid flag", []string{"version", "--private-value"}, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := Run(context.Background(), tt.args, Options{Version: version.Info{Version: "dev", Commit: "unknown"}, Stdout: &stdout, Stderr: &stderr})
			if got != tt.code {
				t.Fatalf("exit=%d want=%d", got, tt.code)
			}
			var result map[string]json.RawMessage
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatalf("machine result: %s", stdout.String())
			}
			if strings.Contains(stdout.String(), "private-") || strings.Contains(stderr.String(), "private-") {
				t.Fatal("raw argument leaked in diagnostic")
			}
		})
	}
}
