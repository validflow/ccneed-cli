package update

import (
	"context"
	"errors"
	"testing"
	"time"
)

// memoryCache 是真实行为的非敏感缓存替身，不模拟检查器本身。
type memoryCache struct {
	record           Record
	loadErr, saveErr error
}

func (c *memoryCache) Load() (Record, error) { return c.record, c.loadErr }
func (c *memoryCache) Save(r Record) error   { c.record = r; return c.saveErr }

type recordingSource struct {
	version string
	err     error
	calls   int
	bounded bool
}

func (s *recordingSource) Latest(ctx context.Context) (string, error) {
	s.calls++
	_, s.bounded = ctx.Deadline()
	return s.version, s.err
}

func TestCheckerRefreshesAndCaches(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	cache := &memoryCache{}
	source := &recordingSource{version: "v0.2.0"}
	checker := New(source, cache, func() time.Time { return now })
	for range 2 {
		if got := checker.Check(context.Background(), "0.1.0"); got != "v0.2.0" {
			t.Fatalf("reminder=%q", got)
		}
	}
	if source.calls != 1 || !source.bounded || !cache.record.CheckedAt.Equal(now) {
		t.Fatalf("calls=%d bounded=%v cached=%+v", source.calls, source.bounded, cache.record)
	}
}

func TestCheckerFailuresAreNonFatalAndCached(t *testing.T) {
	now := time.Now()
	cache := &memoryCache{}
	source := &recordingSource{err: errors.New("offline")}
	checker := New(source, cache, func() time.Time { return now })
	for range 2 {
		if got := checker.Check(context.Background(), "0.1.0"); got != "" {
			t.Fatalf("failure reminder=%q", got)
		}
	}
	if source.calls != 1 || !cache.record.CheckedAt.Equal(now) {
		t.Fatalf("failure not cached: calls=%d", source.calls)
	}
}

func TestCheckerCacheEdgeCases(t *testing.T) {
	now := time.Now()
	for _, tt := range []struct {
		name    string
		at      time.Time
		refresh bool
	}{
		{"fresh", now.Add(-time.Hour), false},
		{"expired", now.Add(-24 * time.Hour), true},
		{"future", now.Add(time.Hour), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cache := &memoryCache{record: Record{CheckedAt: tt.at, Version: "v0.2.0"}}
			source := &recordingSource{version: "v0.3.0"}
			got := New(source, cache, func() time.Time { return now }).Check(context.Background(), "0.1.0")
			want := "v0.2.0"
			if tt.refresh {
				want = "v0.3.0"
			}
			if got != want || (source.calls > 0) != tt.refresh {
				t.Fatalf("version=%s calls=%d", got, source.calls)
			}
		})
	}
}

func TestCheckerIgnoresDevAndPrerelease(t *testing.T) {
	for _, current := range []string{"dev", "0.1.0-rc.1", ""} {
		source := &recordingSource{version: "v9.0.0"}
		if got := New(source, &memoryCache{}, time.Now).Check(context.Background(), current); got != "" || source.calls != 0 {
			t.Fatalf("dev reminder=%s calls=%d", got, source.calls)
		}
	}
}
