package version

import "testing"

func TestNewerStable(t *testing.T) {
	tests := []struct {
		candidate, current string
		want               bool
	}{
		{"v0.1.1", "0.1.0", true},
		{"0.10.0", "v0.9.9", true},
		{"1.0.0", "0.99.99", true},
		{"1.0.0+sha.123", "0.9.0", true},
		{"100000000000000000000.0.0", "99999999999999999999.0.0", true},
		{"v1.0.0", "1.0.0", false},
		{"1.0.0+new", "1.0.0+old", false},
		{"0.9.0", "1.0.0", false},
		{"1.0.0-rc.1", "0.9.0", false},
		{"1.0.0", "1.0.0-dev", false},
		{"1.0.0", "dev", false},
		{"01.0.0", "0.9.0", false},
		{"1.0", "0.9.0", false},
		{"1.0.0+", "0.9.0", false},
		{"1.0.0", "garbage", false},
	}
	for _, tt := range tests {
		t.Run(tt.candidate+"/"+tt.current, func(t *testing.T) {
			if got := NewerStable(tt.candidate, tt.current); got != tt.want {
				t.Fatalf("NewerStable(%q, %q) = %v, want %v", tt.candidate, tt.current, got, tt.want)
			}
		})
	}
}
