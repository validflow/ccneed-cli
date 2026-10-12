package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/credential"
	"github.com/zalando/go-keyring"
)

func pendingService(t *testing.T, handler http.HandlerFunc) (*Service, credential.Pending) {
	t.Helper()
	keyring.MockInit()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := api.New(server.URL, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	vault := credential.New(credential.NewSystem(), nil)
	pending := credential.Pending{Origin: server.URL, UserCode: "ABCDEFGH23", DeviceCode: strings.Repeat("d", 43), Verifier: strings.Repeat("v", 43), ExpiresAt: time.Now().Add(time.Minute), NextPollAt: time.Now().Add(-time.Second), Interval: 5}
	if err := vault.SavePending(context.Background(), server.URL, pending); err != nil {
		t.Fatal(err)
	}
	return New(client, vault, nil), pending
}

func authReply(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Error(err)
	}
}

func TestCompleteHTTPDeadlineIsPendingTimeoutNotCloudAuthorizationFailure(t *testing.T) {
	service, pending := pendingService(t, func(_ http.ResponseWriter, request *http.Request) {
		if _, err := io.Copy(io.Discard, request.Body); err != nil {
			t.Error(err)
			return
		}
		<-request.Context().Done()
	})
	result, err := service.Complete(context.Background(), time.Second)
	if err != nil || result.Status != "pending" || result.Reason != "timeout" {
		t.Fatalf("HTTP wait deadline became authorization failure: result=%+v err=%v", result, err)
	}
	if _, err := service.vault.LoadPending(context.Background(), pending.Origin); err != nil {
		t.Fatal("timeout lost device proof")
	}
}

func TestTerminalDecisionCleansOnlyPendingProof(t *testing.T) {
	for _, status := range []string{"denied", "expired"} {
		t.Run(status, func(t *testing.T) {
			service, pending := pendingService(t, func(w http.ResponseWriter, _ *http.Request) {
				authReply(t, w, map[string]string{"status": status})
			})
			result, err := service.Complete(context.Background(), time.Second)
			if err != nil || result.Status != status {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			_, err = service.vault.LoadPending(context.Background(), pending.Origin)
			var safe *api.Error
			if !errors.As(err, &safe) || safe.Code != "auth_request_missing" {
				t.Fatalf("proof retained: %v", err)
			}
			local, err := service.vault.Inspect(context.Background(), pending.Origin)
			if err != nil || local.State != "missing" {
				t.Fatal("denied or expired authorization signed in")
			}
		})
	}
}

func TestSlowDownPersistsServerWindowAcrossCommands(t *testing.T) {
	service, pending := pendingService(t, func(w http.ResponseWriter, _ *http.Request) {
		authReply(t, w, map[string]any{"status": "slow_down", "interval": 10})
	})
	started := time.Now()
	result, err := service.Complete(context.Background(), time.Second)
	if err != nil || result.Status != "pending" || result.Reason != "timeout" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	saved, err := service.vault.LoadPending(context.Background(), pending.Origin)
	if err != nil || saved.Interval != 10 || saved.NextPollAt.Before(started.Add(10*time.Second)) {
		t.Fatalf("server window lost: %+v err=%v", saved, err)
	}
}

func TestPollUnknownOrSecretBearingWaitingResponseFailsClosed(t *testing.T) {
	for _, response := range []map[string]any{{"status": "success"}, {"status": "pending", "interval": 1}, {"status": "pending", "interval": 5, "access_token": "secret"}} {
		t.Run(response["status"].(string), func(t *testing.T) {
			service, pending := pendingService(t, func(w http.ResponseWriter, _ *http.Request) { authReply(t, w, response) })
			_, err := service.Complete(context.Background(), time.Second)
			var safe *api.Error
			if !errors.As(err, &safe) || safe.Code != "invalid_response" {
				t.Fatalf("unsafe response accepted: %v", err)
			}
			local, err := service.vault.Inspect(context.Background(), pending.Origin)
			if err != nil || local.State != "missing" {
				t.Fatal("invalid response created credential")
			}
		})
	}
}

func TestCompleteMissingProofSendsNoHTTP(t *testing.T) {
	service, pending := pendingService(t, func(_ http.ResponseWriter, _ *http.Request) { t.Error("missing proof sent HTTP") })
	if err := service.vault.ForgetPending(context.Background(), pending.Origin); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Complete(context.Background(), time.Second); err == nil {
		t.Fatal("missing proof accepted")
	}
}
