package update

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileCacheRoundTripAndReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ccneed", "update.json")
	cache := NewFileCache(path)
	now := time.Now().UTC().Truncate(time.Second)
	for _, version := range []string{"v0.2.0", "v0.3.0"} {
		want := Record{CheckedAt: now, Version: version}
		if err := cache.Save(want); err != nil {
			t.Fatal(err)
		}
		got, err := cache.Load()
		if err != nil || got != want {
			t.Fatalf("record=%+v error=%v want=%+v", got, err, want)
		}
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("temporary files left behind: %v", entries)
	}
}

func TestFileCacheMissingIsRecoverable(t *testing.T) {
	if _, err := NewFileCache(filepath.Join(t.TempDir(), "missing.json")).Load(); err == nil {
		t.Fatal("missing cache accepted")
	}
}
