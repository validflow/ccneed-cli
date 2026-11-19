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

	"github.com/validflow/ccneed-cli/internal/credential"
	"github.com/zalando/go-keyring"
)

func TestLoginCarriesCanonicalAgentToDeviceRequest(t *testing.T) {
	for _, row := range []struct{ input, want string }{
		{"codex", "codex"}, {"claude", "claude"}, {"workbuddy", "workbuddy"},
		{"cx", "codex"}, {"wb", "workbuddy"}, {"cc", "claude"},
		{"kimi", "kimi"}, {"openclaw", "openclaw"}, {"pi", "pi"},
		{"zcode", "zcode"}, {"qoder", "qoder"}, {"trae", "trae"}, {"dsh", "dsh"},
	} {
		t.Run(row.input, func(t *testing.T) {
			keyring.MockInit()
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if r.URL.Path != "/api/v1/client/auth/device" || body["agent"] != row.want || len(body["challenge"]) != 64 {
					t.Errorf("wrong authorization identity: %v", body)
				}
				fixture := loginFixture{t: t}
				fixture.reply(w, map[string]any{"device_code": loginDeviceSecret, "user_code": "ABCDEFGH23", "verification_path": "/auth/cli", "expires_in": 600, "interval": 5})
			}))
			defer server.Close()
			invokeLogin(t, server.URL, credential.New(credential.NewSystem(), nil), "auth", "login", "--agent", row.input)
			if calls != 1 {
				t.Fatalf("device start calls = %d", calls)
			}
		})
	}
}

func TestLoginRejectsInvalidAgentBeforeNetwork(t *testing.T) {
	for _, args := range [][]string{{"--agent", "untrusted-display-name"}, {"--complete", "--agent", "codex"}} {
		keyring.MockInit()
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusBadRequest)
		}))
		var output bytes.Buffer
		command := append([]string{"--no-update-check", "--api-url", server.URL, "auth", "login"}, args...)
		code := Run(context.Background(), command, Options{Stdout: &output, Stderr: &output, Credentials: credential.New(credential.NewSystem(), nil)})
		server.Close()
		if code == 0 || !strings.Contains(output.String(), "invalid_input") || calls.Load() != 0 {
			t.Fatalf("invalid login reached network: exit=%d calls=%d", code, calls.Load())
		}
	}
}
