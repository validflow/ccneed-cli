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

func TestAuthStatusVerifyChecksServerInsteadOfClaimingLocalPresenceIsAuthentication(t *testing.T) {
	keyring.MockInit()
	var revoked atomic.Bool
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		if request.URL.Path != "/api/v1/payment/credits" || request.Header.Get("Authorization") != "Bearer "+loginAccessToken {
			t.Error("verification escaped account-read boundary")
		}
		if revoked.Load() {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err := json.NewEncoder(w).Encode(map[string]int{"left_credits": 123}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	vault := credential.New(credential.NewSystem(), nil)
	err := vault.Save(context.Background(), server.URL, credential.Record{Origin: server.URL, Token: loginAccessToken, CredentialID: "device-one", UserID: "user-one", ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	args := []string{"--no-update-check", "--api-url", server.URL, "auth", "status", "--verify"}
	options := Options{Stdout: &stdout, Stderr: &stderr, Credentials: vault}
	if code := Run(context.Background(), args, options); code != 0 || !strings.Contains(stdout.String(), `"authenticated":true`) || calls.Load() != 1 {
		t.Fatalf("verification did not check real server: exit=%d calls=%d output=%s", code, calls.Load(), stdout.String())
	}
	revoked.Store(true)
	stdout.Reset()
	if code := Run(context.Background(), args, options); code == 0 || strings.Contains(stdout.String(), `"authenticated":true`) {
		t.Fatalf("revoked credential reported authenticated: %s", stdout.String())
	}
}
