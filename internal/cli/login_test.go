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
	"time"

	"github.com/validflow/ccneed-cli/internal/credential"
	"github.com/zalando/go-keyring"
)

const loginDeviceSecret = "ddddddddddddddddddddddddddddddddddddddddddd"
const loginAccessToken = "ccn_cli_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

type loginFixture struct {
	t       *testing.T
	revoked atomic.Bool
}

func (f *loginFixture) reply(w http.ResponseWriter, value any) {
	f.t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		f.t.Error(err)
	}
}

func (f *loginFixture) device(w http.ResponseWriter, request *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		f.t.Error(err)
	}
	if len(body["challenge"]) != 64 || request.Header.Get("Authorization") != "" {
		f.t.Error("invalid device start boundary")
	}
	f.reply(w, map[string]any{"device_code": loginDeviceSecret, "user_code": "ABCDEFGH23", "verification_path": "/auth/cli", "expires_in": 600, "interval": 5})
}

func (f *loginFixture) token(w http.ResponseWriter, request *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		f.t.Error(err)
	}
	if body["device_code"] != loginDeviceSecret || len(body["verifier"]) != 43 {
		f.t.Error("device proof missing")
	}
	f.reply(w, map[string]any{"status": "authorized", "access_token": loginAccessToken, "credential_id": "device-one", "user_id": "website-user", "expires_at": time.Now().Add(time.Hour).UTC().Format(time.RFC3339)})
}

func (f *loginFixture) serve(w http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/api/v1/client/auth/device":
		f.device(w, request)
	case "/api/v1/client/auth/token":
		f.token(w, request)
	case "/api/v1/payment/credits":
		if request.Header.Get("Authorization") != "Bearer "+loginAccessToken || f.revoked.Load() {
			f.t.Error("wrong origin-bound token")
		}
		f.reply(w, map[string]int{"left_credits": 123})
	case "/api/v1/client/auth/revoke":
		if request.Header.Get("Authorization") != "Bearer "+loginAccessToken {
			f.t.Error("revocation missing credential")
		}
		f.revoked.Store(true)
		f.reply(w, map[string]bool{"revoked": true})
	default:
		f.t.Errorf("unexpected endpoint %s", request.URL.Path)
		w.WriteHeader(404)
	}
}

func invokeLogin(t *testing.T, origin string, vault *credential.Vault, args ...string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	command := append([]string{"--no-update-check", "--api-url", origin}, args...)
	code := Run(context.Background(), command, Options{Stdout: &stdout, Stderr: &stderr, Credentials: vault})
	if code != 0 {
		t.Fatalf("args=%v exit=%d output=%s", args, code, stdout.String())
	}
	output := stdout.String() + stderr.String()
	if strings.Contains(output, loginDeviceSecret) || strings.Contains(output, loginAccessToken) {
		t.Fatal("command exposed a device secret or credential")
	}
	return stdout.String()
}

func assertLocalCredentialMissing(t *testing.T, origin string, vault *credential.Vault) {
	t.Helper()
	status, err := vault.Inspect(context.Background(), origin)
	if err != nil || status.State != "missing" {
		t.Fatal("unexpected local credential")
	}
}

func TestWebsiteDeviceLoginAccountAndRevocationThroughRealHTTP(t *testing.T) {
	keyring.MockInit()
	fixture := &loginFixture{t: t}
	server := httptest.NewServer(http.HandlerFunc(fixture.serve))
	defer server.Close()
	vault := credential.New(credential.NewSystem(), nil)
	initial := invokeLogin(t, server.URL, vault, "auth", "login")
	if !strings.Contains(initial, `"status":"pending"`) || !strings.Contains(initial, server.URL+"/auth/cli?user_code=ABCDEFGH23") {
		t.Fatalf("invalid public authorization output %s", initial)
	}
	assertLocalCredentialMissing(t, server.URL, vault)
	completed := invokeLogin(t, server.URL, vault, "auth", "login", "--complete", "--timeout", "10s")
	if !strings.Contains(completed, `"status":"authenticated"`) {
		t.Fatalf("invalid completion %s", completed)
	}
	if output := invokeLogin(t, server.URL, vault, "account", "credits"); !strings.Contains(output, `"left_credits":123`) {
		t.Fatalf("invalid balance %s", output)
	}
	invokeLogin(t, server.URL, vault, "auth", "logout")
	if !fixture.revoked.Load() {
		t.Fatal("logout forgot credential without server revocation")
	}
	assertLocalCredentialMissing(t, server.URL, vault)
}
