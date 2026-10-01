package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func TestClientJSONRequest(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/usight-agent/threads" || r.URL.Query().Get("view") != "active" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer test-only-value" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing authentication or JSON content type")
		}
		assertJSONRequestBody(t, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"thread_id":"thread-one"}`))
	}))
	defer server.Close()
	client, err := New(server.URL, "test-only-value", nil)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		ThreadID string `json:"thread_id"`
	}
	err = client.DoJSON(context.Background(), http.MethodPost, "/api/v1/usight-agent/threads", url.Values{"view": {"active"}}, map[string]string{"title": "Example"}, &result)
	if err != nil || result.ThreadID != "thread-one" || calls.Load() != 1 {
		t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
	}
}

// assertJSONRequestBody 检查实际传输的字段和单个 JSON 值，防止只验证响应掩盖请求序列化回归。
func assertJSONRequestBody(t *testing.T, reader io.Reader) {
	t.Helper()
	decoder := json.NewDecoder(reader)
	var body map[string]string
	if err := decoder.Decode(&body); err != nil || len(body) != 1 || body["title"] != "Example" {
		t.Errorf("unexpected JSON request body: %+v, error=%v", body, err)
	}
	if err := decoder.Decode(&body); err != io.EOF {
		t.Errorf("request body must contain exactly one JSON value: %v", err)
	}
}

func TestClientDoesNotFollowRedirect(t *testing.T) {
	var redirected atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected.Add(1) }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	client, err := New(server.URL, "test-only-value", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = client.DoJSON(context.Background(), http.MethodPost, "/api/v1/test", nil, map[string]string{"message": "private"}, nil)
	if err == nil || redirected.Load() != 0 {
		t.Fatalf("redirect error=%v target requests=%d", err, redirected.Load())
	}
}

func TestClientRejectsUnsafePath(t *testing.T) {
	for _, path := range []string{"https://other.example/api/v1/test", "//other.example/api/v1/test", "/api/v1/../admin", "/api/v1/test?token=private", "/api/v1/test#fragment"} {
		client, err := New("https://ccneed.ai", "", nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := client.DoJSON(context.Background(), http.MethodGet, path, nil, nil, nil); err == nil {
			t.Errorf("accepted unsafe path %q", path)
		}
	}
}

func TestClientFailureDoesNotLeakOrRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"detail":"private-server-data"}`)
	}))
	defer server.Close()
	client, err := New(server.URL, "test-only-value", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = client.DoJSON(context.Background(), http.MethodPost, "/api/v1/test", nil, nil, nil)
	if err == nil || strings.Contains(err.Error(), "private-server-data") || strings.Contains(err.Error(), "test-only-value") || calls.Load() != 1 {
		t.Fatalf("error=%v calls=%d", err, calls.Load())
	}
}

func TestClientRejectsInvalidResponse(t *testing.T) {
	for _, response := range []string{`{"ok":true} {"another":true}`, "not json", strings.Repeat("x", 4*1024*1024+1)} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = fmt.Fprint(w, response) }))
		client, err := New(server.URL, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]bool
		err = client.DoJSON(context.Background(), http.MethodGet, "/api/v1/test", nil, nil, &result)
		server.Close()
		if err == nil {
			t.Error("accepted invalid or oversized JSON")
		}
	}
}

func TestClientCancellationPreservesCause(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client, err := New("https://ccneed.ai", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = client.DoJSON(ctx, http.MethodGet, "/api/v1/test", nil, nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation cause lost: %v", err)
	}
}
