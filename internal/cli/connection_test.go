package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/validflow/ccneed-cli/internal/api"
)

func TestSelectedOriginRejectsDifferentAuthenticatedConnection(t *testing.T) {
	var calls atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"left_credits":42}`))
	})
	selected := httptest.NewServer(handler)
	defer selected.Close()
	other := httptest.NewServer(handler)
	defer other.Close()
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--api-url", selected.URL, "account", "credits"}, Options{
		Stdout: &stdout, Stderr: &stderr,
		Connect: func(_ context.Context, origin string) (*api.Client, error) {
			if origin != selected.URL {
				t.Fatalf("resolver received unexpected origin %q", origin)
			}
			return api.New(other.URL, "test-only-secret", nil)
		},
	})
	if code == 0 || calls.Load() != 0 || !json.Valid(stdout.Bytes()) || !strings.Contains(stdout.String(), `"code":"origin_mismatch"`) {
		t.Fatalf("exit=%d requests=%d output=%s", code, calls.Load(), stdout.String())
	}
	if strings.Contains(stdout.String()+stderr.String(), "test-only-secret") {
		t.Fatal("credential leaked")
	}
}

func TestEmptyAuthenticatedConnectionFailsWithoutPanic(t *testing.T) {
	for _, client := range []*api.Client{nil, {}} {
		t.Run("unavailable", func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("invalid authentication result panicked: %v", recovered)
				}
			}()
			var stdout, stderr bytes.Buffer
			code := Run(context.Background(), []string{"account", "credits"}, Options{
				Stdout: &stdout, Stderr: &stderr,
				Connect: func(context.Context, string) (*api.Client, error) { return client, nil },
			})
			if code == 0 || !json.Valid(stdout.Bytes()) || !strings.Contains(stdout.String(), `"code":"not_authenticated"`) {
				t.Fatalf("exit=%d output=%s", code, stdout.String())
			}
		})
	}
}
