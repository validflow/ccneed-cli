package credential

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

// 系统适配器测试只替换 keyring 的 OS 边界，不读写开发者真实凭证库。
func TestSystemBackendRoundTripAndScopedRemoval(t *testing.T) {
	keyring.MockInit()
	backend := NewSystem()
	for _, account := range []string{"one", "two"} {
		if err := backend.Set("test-only-service", account, "test-only-secret"); err != nil {
			t.Fatal(err)
		}
		value, err := backend.Get("test-only-service", account)
		if err != nil || value != "test-only-secret" {
			t.Fatalf("value or read failed: %v", err)
		}
	}
	if err := backend.Delete("test-only-service", "one"); err != nil {
		t.Fatal(err)
	}
	if _, err := backend.Get("test-only-service", "one"); !errors.Is(err, ErrMissing) {
		t.Fatalf("expected missing: %v", err)
	}
	if err := backend.Delete("test-only-service", "one"); !errors.Is(err, ErrMissing) {
		t.Fatalf("expected missing: %v", err)
	}
	if _, err := backend.Get("test-only-service", "two"); err != nil {
		t.Fatal("deleted unrelated keyring account")
	}
}

func TestSystemBackendErrorsDoNotExposeOperatingSystemOutput(t *testing.T) {
	keyring.MockInitWithError(errors.New("private-system-output test-only-secret"))
	backend := NewSystem()
	_, readErr := backend.Get("test-only-service", "one")
	for _, err := range []error{readErr, backend.Set("test-only-service", "one", "test-only-secret"), backend.Delete("test-only-service", "one")} {
		if err == nil || errors.Is(err, ErrMissing) || strings.Contains(err.Error(), "test-only-secret") || strings.Contains(err.Error(), "private-system") {
			t.Fatalf("unsafe error mapping: %v", err)
		}
	}
}

func TestVaultStoreWriteAndDeleteFailuresStayVisible(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	origin := "https://service.example"
	backend := &memoryBackend{values: map[string]string{}, failure: errors.New("private-system-output")}
	vault := New(backend, func() time.Time { return now })
	requireCode(t, vault.Save(context.Background(), origin, testRecord(origin, now)), "credential_store_unavailable")
	requireCode(t, vault.Forget(context.Background(), origin), "credential_store_unavailable")
	if len(backend.values) != 0 {
		t.Fatal("failed save left a partial credential")
	}
}

func TestVaultRejectsUnencodableExpiryAndControlIdentity(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	origin := "https://service.example"
	for _, change := range []func(*Record){
		func(r *Record) { r.ExpiresAt = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC) },
		func(r *Record) { r.UserID = "user\x1b[31m" },
		func(r *Record) { r.CredentialID = strings.Repeat("a", 129) },
		func(r *Record) { r.Origin = "https://name:password@service.example" },
		func(r *Record) { r.ExpiresAt = time.Time{} },
	} {
		backend := &memoryBackend{values: map[string]string{}}
		vault := New(backend, func() time.Time { return now })
		record := testRecord(origin, now)
		change(&record)
		requireCode(t, vault.Save(context.Background(), origin, record), "invalid_credential")
		if backend.writes != 0 {
			t.Fatal("invalid record reached OS credential store")
		}
	}
}
