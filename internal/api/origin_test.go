package api

import "testing"

func TestParseOrigin(t *testing.T) {
	tests := []struct {
		name, input, want string
		invalid           bool
	}{
		{name: "production", input: "https://ccneed.ai", want: "https://ccneed.ai"},
		{name: "normalize", input: "https://CCNEED.ai:443/", want: "https://ccneed.ai"},
		{name: "localhost", input: "http://localhost:8000/", want: "http://localhost:8000"},
		{name: "ipv4 loopback", input: "http://127.0.0.1:8000", want: "http://127.0.0.1:8000"},
		{name: "ipv6 loopback", input: "http://[::1]:8000", want: "http://[::1]:8000"},
		{name: "insecure", input: "http://ccneed.ai", invalid: true},
		{name: "not loopback", input: "http://127.0.0.1.evil.example", invalid: true},
		{name: "userinfo", input: "https://token@ccneed.ai", invalid: true},
		{name: "path", input: "https://ccneed.ai/api/v1", invalid: true},
		{name: "query", input: "https://ccneed.ai?token=private", invalid: true},
		{name: "empty query", input: "https://ccneed.ai?", invalid: true},
		{name: "fragment", input: "https://ccneed.ai#private", invalid: true},
		{name: "empty", input: "", invalid: true},
		{name: "relative", input: "ccneed.ai", invalid: true},
		{name: "unsupported", input: "ftp://ccneed.ai", invalid: true},
		{name: "bad port", input: "https://ccneed.ai:99999", invalid: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOrigin(tt.input)
			if tt.invalid {
				if err == nil {
					t.Fatalf("ParseOrigin(%q) accepted unsafe origin %v", tt.input, got)
				}
				return
			}
			if err != nil || got.String() != tt.want {
				t.Fatalf("ParseOrigin(%q) = %v, %v; want %q", tt.input, got, err, tt.want)
			}
		})
	}
}
