package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/validflow/ccneed-cli/internal/credential"
	"github.com/zalando/go-keyring"
)

type waitingCredentialBackend struct {
	value   string
	started chan struct{}
	release chan struct{}
}

func (b *waitingCredentialBackend) Get(_, _ string) (string, error) {
	close(b.started)
	<-b.release
	return b.value, nil
}
func (b *waitingCredentialBackend) Set(_, _, value string) error { b.value = value; return nil }
func (b *waitingCredentialBackend) Delete(_, _ string) error     { b.value = ""; return nil }

func TestAccountCancellationDuringCredentialReadSendsNoHTTP(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { calls.Add(1) }))
	defer server.Close()
	backend := &waitingCredentialBackend{started: make(chan struct{}), release: make(chan struct{})}
	defer close(backend.release)
	now := time.Now()
	vault := credential.New(backend, func() time.Time { return now })
	if err := vault.Save(context.Background(), server.URL, credential.Record{Origin: server.URL, CredentialID: "device-one", UserID: "user-one", Token: "test-only-secret", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stdout, stderr bytes.Buffer
	result := make(chan int, 1)
	go func() {
		result <- Run(ctx, []string{"--no-update-check", "--api-url", server.URL, "account", "credits"}, Options{Stdout: &stdout, Stderr: &stderr, Credentials: vault})
	}()
	select {
	case <-backend.started:
	case <-time.After(time.Second):
		t.Fatal("command did not start credential read")
	}
	cancel()
	select {
	case code := <-result:
		if code == 0 || calls.Load() != 0 || !json.Valid(stdout.Bytes()) || strings.Contains(stdout.String()+stderr.String(), "test-only-secret") {
			t.Fatalf("exit=%d requests=%d output=%s", code, calls.Load(), stdout.String())
		}
	case <-time.After(time.Second):
		t.Fatal("command remained blocked after cancellation")
	}
}

func TestAuthStatusReportsLocalStateWithoutTokenOrAuthenticationClaim(t *testing.T) {
	keyring.MockInit()
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	vault := credential.New(credential.NewSystem(), func() time.Time { return now })
	origin := "https://preview.example"
	for _, state := range []string{"missing", "present", "expired"} {
		if state == "present" {
			err := vault.Save(context.Background(), origin, credential.Record{Origin: origin, CredentialID: "device-one", UserID: "user-one", Token: "test-only-secret", ExpiresAt: now.Add(time.Hour)})
			if err != nil {
				t.Fatal(err)
			}
		}
		if state == "expired" {
			now = now.Add(time.Hour)
		}
		var stdout, stderr bytes.Buffer
		code := Run(context.Background(), []string{"--no-update-check", "--api-url", origin, "auth", "status"}, Options{Stdout: &stdout, Stderr: &stderr, Credentials: vault})
		var result credential.Status
		if code != 0 || json.Unmarshal(stdout.Bytes(), &result) != nil || result.State != state || result.Origin != origin {
			t.Fatalf("expected=%s exit=%d output=%s", state, code, stdout.String())
		}
		if strings.Contains(stdout.String()+stderr.String(), "test-only-secret") || strings.Contains(stdout.String(), `"authenticated"`) {
			t.Fatal("local status leaked token or claimed server authentication")
		}
	}
}

func TestAuthStatusUnavailableIsNotReportedAsMissing(t *testing.T) {
	keyring.MockInitWithError(errors.New("private system credential information"))
	vault := credential.New(credential.NewSystem(), nil)
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--no-update-check", "auth", "status"}, Options{Stdout: &stdout, Stderr: &stderr, Credentials: vault})
	if code == 0 || !strings.Contains(stdout.String(), `"code":"credential_store_unavailable"`) {
		t.Fatalf("exit=%d output=%s", code, stdout.String())
	}
	if strings.Contains(stdout.String()+stderr.String(), "private system") {
		t.Fatal("system diagnostics leaked")
	}
}

func TestAccountConnectionUsesOnlySelectedOriginStoredCredential(t *testing.T) {
	keyring.MockInit()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer test-only-secret" || r.URL.Path != "/api/v1/payment/credits" {
			t.Error("incorrect authenticated request")
		}
		_, _ = fmt.Fprint(w, `{"left_credits":42}`)
	}))
	defer server.Close()
	other := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("missing other-origin credential sent a request")
	}))
	defer other.Close()
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	vault := credential.New(credential.NewSystem(), func() time.Time { return now })
	if err := vault.Save(context.Background(), server.URL, credential.Record{Origin: server.URL, CredentialID: "device-one", UserID: "user-one", Token: "test-only-secret", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct {
		origin, code string
		exit         int
	}{
		{server.URL, `"left_credits":42`, 0},
		{other.URL, `"code":"not_authenticated"`, 1},
	} {
		var stdout, stderr bytes.Buffer
		exit := Run(context.Background(), []string{"--no-update-check", "--api-url", scenario.origin, "account", "credits"}, Options{Stdout: &stdout, Stderr: &stderr, Credentials: vault})
		if exit != scenario.exit || !strings.Contains(stdout.String(), scenario.code) {
			t.Fatalf("exit=%d output=%s", exit, stdout.String())
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("requests=%d", calls.Load())
	}
}
