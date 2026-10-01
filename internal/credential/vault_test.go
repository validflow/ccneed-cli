package credential

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/validflow/ccneed-cli/internal/api"
)

type memoryBackend struct {
	values                 map[string]string
	failure                error
	reads, writes, deletes int
}

func (b *memoryBackend) Get(service, account string) (string, error) {
	b.reads++
	if b.failure != nil {
		return "", b.failure
	}
	value, ok := b.values[service+"|"+account]
	if !ok {
		return "", ErrMissing
	}
	return value, nil
}
func (b *memoryBackend) Set(service, account, secret string) error {
	b.writes++
	if b.failure != nil {
		return b.failure
	}
	b.values[service+"|"+account] = secret
	return nil
}
func (b *memoryBackend) Delete(service, account string) error {
	b.deletes++
	if b.failure != nil {
		return b.failure
	}
	key := service + "|" + account
	if _, ok := b.values[key]; !ok {
		return ErrMissing
	}
	delete(b.values, key)
	return nil
}

func testRecord(origin string, now time.Time) Record {
	return Record{Origin: origin, CredentialID: "device-one", UserID: "user-one", Token: "test-only-secret", ExpiresAt: now.Add(time.Hour)}
}

func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	var result *api.Error
	if !errors.As(err, &result) || result.Code != code {
		t.Fatalf("expected code %s, got %v", code, err)
	}
}

func TestVaultOriginIsolationAndSafeOutput(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	backend := &memoryBackend{values: map[string]string{}}
	vault := New(backend, func() time.Time { return now })
	production := "https://service.example"
	staging := "https://preview.example"
	if err := vault.Save(context.Background(), production, testRecord(production, now)); err != nil {
		t.Fatal(err)
	}
	if _, err := vault.Connect(context.Background(), staging); err == nil {
		t.Fatal("other origin reused production credential")
	}
	status, err := vault.Inspect(context.Background(), production)
	if err != nil || status.State != "present" || status.UserID != "user-one" {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	client, err := vault.Connect(context.Background(), "https://SERVICE.example:443/")
	if err != nil || client.Origin() != production {
		t.Fatalf("client=%v err=%v", client, err)
	}
}

func TestVaultSerializationHasNoToken(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	origin := "https://service.example"
	vault := New(&memoryBackend{values: map[string]string{}}, func() time.Time { return now })
	record := testRecord(origin, now)
	if err := vault.Save(context.Background(), origin, record); err != nil {
		t.Fatal(err)
	}
	status, err := vault.Inspect(context.Background(), origin)
	if err != nil {
		t.Fatal(err)
	}
	for _, output := range []any{status, record} {
		encoded, err := json.Marshal(output)
		if err != nil || strings.Contains(string(encoded), "test-only-secret") {
			t.Fatal("public JSON exposed token")
		}
	}
}

func TestVaultSaveValidatesBeforeWriting(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	origin := "https://service.example"
	for _, scenario := range []struct {
		name   string
		change func(*Record)
		code   string
	}{
		{"wrong origin", func(r *Record) { r.Origin = "https://other.example" }, "origin_mismatch"},
		{"expired", func(r *Record) { r.ExpiresAt = now }, "credential_expired"},
		{"empty token", func(r *Record) { r.Token = "" }, "invalid_credential"},
		{"header injection", func(r *Record) { r.Token = "secret\r\nother" }, "invalid_credential"},
		{"oversized", func(r *Record) { r.Token = strings.Repeat("a", 2000) }, "invalid_credential"},
		{"missing identity", func(r *Record) { r.UserID = "" }, "invalid_credential"},
		{"missing device", func(r *Record) { r.CredentialID = "" }, "invalid_credential"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			backend := &memoryBackend{values: map[string]string{}}
			vault := New(backend, func() time.Time { return now })
			record := testRecord(origin, now)
			scenario.change(&record)
			requireCode(t, vault.Save(context.Background(), origin, record), scenario.code)
			if backend.writes != 0 {
				t.Fatal("invalid record touched credential store")
			}
		})
	}
}

func TestVaultExpirationDoesNotSendRequestOrDeleteCredential(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	backend := &memoryBackend{values: map[string]string{}}
	vault := New(backend, func() time.Time { return now })
	origin := "https://service.example"
	if err := vault.Save(context.Background(), origin, testRecord(origin, now)); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Hour)
	_, err := vault.Connect(context.Background(), origin)
	requireCode(t, err, "credential_expired")
	status, err := vault.Inspect(context.Background(), origin)
	if err != nil || status.State != "expired" || backend.deletes != 0 {
		t.Fatalf("status=%+v deletes=%d err=%v", status, backend.deletes, err)
	}
}

func TestVaultRejectsCorruptedOrMisboundStoredRecords(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	for _, payload := range []string{
		"invalid JSON",
		`{"version":9}`,
		`{"version":1,"origin":"https://other.example","credential_id":"device-one","user_id":"user-one","expires_at":"2026-10-01T01:00:00Z","token":"test-only-secret"}`,
		strings.Repeat("a", 3000),
	} {
		backend := &memoryBackend{values: map[string]string{}}
		vault := New(backend, func() time.Time { return now })
		if err := vault.Save(context.Background(), "https://service.example", testRecord("https://service.example", now)); err != nil {
			t.Fatal(err)
		}
		for key := range backend.values {
			backend.values[key] = payload
		}
		_, err := vault.Connect(context.Background(), "https://service.example")
		requireCode(t, err, "credential_corrupt")
		if strings.Contains(err.Error(), payload) || strings.Contains(err.Error(), "test-only-secret") {
			t.Fatal("corrupt secret leaked")
		}
	}
}

func TestVaultMissingUnavailableCancelledAndInvalidOrigin(t *testing.T) {
	backend := &memoryBackend{values: map[string]string{}}
	vault := New(backend, time.Now)
	status, err := vault.Inspect(context.Background(), "https://service.example")
	if err != nil || status.State != "missing" {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	_, err = vault.Connect(context.Background(), "https://service.example")
	requireCode(t, err, "not_authenticated")
	backend.failure = errors.New("sensitive backend details test-only-secret")
	_, err = vault.Inspect(context.Background(), "https://service.example")
	requireCode(t, err, "credential_store_unavailable")
	if strings.Contains(err.Error(), "test-only-secret") {
		t.Fatal("backend error leaked")
	}
	reads := backend.reads
	_, err = vault.Connect(context.Background(), "https://secret@service.example")
	requireCode(t, err, "invalid_origin")
	if reads != backend.reads {
		t.Fatal("invalid origin accessed keyring")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = vault.Connect(ctx, "https://service.example")
	if !errors.Is(err, context.Canceled) || reads != backend.reads {
		t.Fatalf("cancelled operation reached store: %v", err)
	}
	_, err = New(nil, nil).Connect(context.Background(), "https://service.example")
	requireCode(t, err, "credential_store_unavailable")
}

func TestVaultForgetIsIdempotentAndScoped(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	backend := &memoryBackend{values: map[string]string{}}
	vault := New(backend, func() time.Time { return now })
	for _, origin := range []string{"https://service.example", "https://preview.example"} {
		if err := vault.Save(context.Background(), origin, testRecord(origin, now)); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if err := vault.Forget(context.Background(), "https://preview.example"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := vault.Connect(context.Background(), "https://service.example"); err != nil {
		t.Fatal("forget removed other origin")
	}
	status, err := vault.Inspect(context.Background(), "https://preview.example")
	if err != nil || status.State != "missing" {
		t.Fatalf("status=%+v err=%v", status, err)
	}
}

func TestVaultConnectionUsesActualBoundHTTPTransport(t *testing.T) {
	now := time.Now()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer test-only-secret" {
			t.Error("credential not attached")
		}
		_, _ = fmt.Fprint(w, `{"left_credits":42}`)
	}))
	defer server.Close()
	vault := New(&memoryBackend{values: map[string]string{}}, func() time.Time { return now })
	if err := vault.Save(context.Background(), server.URL, testRecord(server.URL, now)); err != nil {
		t.Fatal(err)
	}
	client, err := vault.Connect(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	var output struct {
		Credits int `json:"left_credits"`
	}
	if err := client.DoJSON(context.Background(), http.MethodGet, "/api/v1/payment/credits", nil, nil, &output); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || output.Credits != 42 {
		t.Fatalf("calls=%d credits=%d", calls, output.Credits)
	}
}
