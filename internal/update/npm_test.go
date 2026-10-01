package update

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNPMSourceUsesOnlyStableRegistryChannel(t *testing.T) {
	source := NewNPMSource("@validflow/ccneed-cli", releaseTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "registry.npmjs.org" || r.URL.EscapedPath() != "/@validflow%2Fccneed-cli/latest" || r.Header.Get("Authorization") != "" {
			t.Fatalf("unexpected registry request: %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"name":"@validflow/ccneed-cli","version":"0.2.0"}`)), Header: make(http.Header)}, nil
	}))
	got, err := source.Latest(context.Background())
	if err != nil || got != "0.2.0" {
		t.Fatalf("version=%q error=%v", got, err)
	}
}

func TestNPMSourceRejectsUnavailableOrMismatchedPackage(t *testing.T) {
	for _, body := range []string{
		`{"name":"@validflow/ccneed-cli","version":"0.2.0-rc.1"}`,
		`{"name":"@other/cli","version":"0.2.0"}`,
		`{"version":"0.2.0"}`, "invalid", strings.Repeat("x", 65537),
	} {
		source := NewNPMSource("@validflow/ccneed-cli", releaseTransport(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}))
		if value, err := source.Latest(context.Background()); err == nil || value != "" {
			t.Fatal("accepted invalid package")
		}
	}
	for _, status := range []int{404, 302, 500} {
		source := NewNPMSource("@validflow/ccneed-cli", releaseTransport(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
		}))
		if _, err := source.Latest(context.Background()); err == nil {
			t.Fatal("accepted unavailable registry")
		}
	}
}

func TestNPMSourceRejectsInvalidIdentityBeforeNetwork(t *testing.T) {
	for _, name := range []string{"", "https://evil.invalid", "@validflow/cli/../../other", "@Validflow/cli"} {
		source := NewNPMSource(name, releaseTransport(func(_ *http.Request) (*http.Response, error) { t.Fatal("unexpected network"); return nil, nil }))
		if _, err := source.Latest(context.Background()); err == nil {
			t.Fatal("accepted invalid package name")
		}
	}
}
