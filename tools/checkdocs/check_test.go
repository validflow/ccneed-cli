package main

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestCheckFile(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		code string
		want string
	}{
		{"missing comment", "package p\nfunc Work() {}", "Work: missing doc comment"},
		{"wrong prefix", "package p\n// Does work.\nfunc Work() {}", "Work: comment must start with symbol name"},
		{"missing args", "package p\n// Work consumes a value.\nfunc Work(value string) {}", "Work: missing Args:"},
		{"missing returns", "package p\n// Work returns a value.\nfunc Work() string { return \"\" }", "Work: missing Returns:"},
		{"missing errors", "package p\n// Work returns an error.\n// Returns:\n//   Result.\nfunc Work() error { return nil }", "Work: missing Errors:"},
		{"missing type comment", "package p\ntype Client struct{}", "Client: missing doc comment"},
		{"missing method comment", "package p\n// Client is a client.\ntype Client struct{}\nfunc (c *Client) Work() {}", "Work: missing doc comment"},
		{"history comment", "package p\n// CCN-123 fixes behavior.\nvar value = 1", "issue identifier in comment"},
		{"valid", "package p\n// Work validates input.\n// Args:\n//   value: Input.\n// Returns:\n//   Result.\n// Errors:\n//   Validation failure.\nfunc Work(value string) error { return nil }", ""},
		{"grouped type docs", "package p\ntype (\n// Client holds state.\nClient struct{}\n)", ""},
		{"issue in string is data", "package p\nvar value = \"CCN-123\"", ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			positions := token.NewFileSet()
			file, err := parser.ParseFile(positions, "example.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			got := strings.Join(checkFile(positions, file), "\n")
			if tt.want == "" && got != "" {
				t.Fatalf("unexpected violations: %s", got)
			}
			if tt.want != "" && !strings.Contains(got, tt.want) {
				t.Fatalf("want violation %q, got %q", tt.want, got)
			}
		})
	}
}
