package account

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/validflow/ccneed-cli/internal/api"
)

// accountServer 使用真实 HTTP client，只替换云端服务器。
func accountServer(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	transport, err := api.New(server.URL, "test-only-value", nil)
	if err != nil {
		t.Fatal(err)
	}
	return New(transport)
}

// overviewHandler 提供两个独立的既有读取接口，并验证认证请求。
func overviewHandler(t *testing.T, calls *atomic.Int32) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer test-only-value" {
			t.Error("not an authenticated read")
		}
		switch r.URL.Path {
		case "/api/v1/payment/credits":
			_, _ = fmt.Fprint(w, `{"left_credits":123,"is_pro":true,"details":{"private_metadata":"do-not-expose"}}`)
		case "/api/v1/redemption/subscriptions":
			_, _ = fmt.Fprint(w, `{"subscriptions":[{"id":123,"plan_name":"Personal","plan_level":1,"expired_at":"2027-01-01T00:00:00Z","private_metadata":"do-not-expose"}]}`)
		default:
			t.Errorf("unexpected API %s", r.URL)
			w.WriteHeader(404)
		}
	}
}

func TestAccountOverviewUsesExistingReadEndpoints(t *testing.T) {
	var calls atomic.Int32
	client := accountServer(t, overviewHandler(t, &calls))
	result, err := client.Overview(context.Background())
	if err != nil || result.Credits.LeftCredits != 123 || len(result.Subscriptions) != 1 || calls.Load() != 2 {
		t.Fatalf("overview=%+v error=%v calls=%d", result, err, calls.Load())
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "private_metadata") || strings.Contains(string(data), "is_pro") {
		t.Fatalf("unsafe account projection: %s", data)
	}
}

func TestBalanceRejectsMissingAndNegativeValues(t *testing.T) {
	for _, response := range []string{`{}`, `{"left_credits":null}`, `{"left_credits":-1}`} {
		client := accountServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = fmt.Fprint(w, response) })
		if _, err := client.Credits(context.Background()); err == nil {
			t.Fatalf("accepted invalid balance %s", response)
		}
	}
}

func TestBalanceZeroIsValid(t *testing.T) {
	client := accountServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = fmt.Fprint(w, `{"left_credits":0}`) })
	if got, err := client.Credits(context.Background()); err != nil || got.LeftCredits != 0 {
		t.Fatalf("balance=%+v error=%v", got, err)
	}
}

func TestHistoryIsRecentWindowAndDoesNotDumpRawRows(t *testing.T) {
	client := accountServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/payment/credits/list" || r.URL.RawQuery != "" {
			t.Errorf("invented pagination or endpoint %s", r.URL)
		}
		_, _ = fmt.Fprint(w, `[{"trans_no":"one","trans_type":"analysis","credits":-2,"user_id":999,"private_metadata":"do-not-expose"},{"trans_no":"two","trans_type":"asset","credits":100}]`)
	})
	got, err := client.History(context.Background(), 1)
	if err != nil || len(got.Items) != 1 || got.Items[0].Transaction != "one" || got.Scope != "recent" || got.SourceLimit != 200 {
		t.Fatalf("history=%+v error=%v", got, err)
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "user_id") || strings.Contains(string(data), "private_metadata") {
		t.Fatalf("raw row exposed: %s", data)
	}
}

func TestHistoryInvalidLimitDoesNotRequest(t *testing.T) {
	var calls atomic.Int32
	client := accountServer(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1) })
	for _, limit := range []int{-1, 0, 201} {
		if _, err := client.History(context.Background(), limit); err == nil {
			t.Errorf("accepted limit %d", limit)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid arguments sent %d requests", calls.Load())
	}
}

func TestOverviewAuthFailureDoesNotFabricateSubscription(t *testing.T) {
	var calls atomic.Int32
	client := accountServer(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(http.StatusUnauthorized) })
	if _, err := client.Overview(context.Background()); err == nil || calls.Load() != 1 {
		t.Fatalf("error=%v requests=%d", err, calls.Load())
	}
}
