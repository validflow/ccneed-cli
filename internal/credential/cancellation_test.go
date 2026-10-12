package credential

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// blockedBackend 让只读 OS 边界等待明确释放，避免靠固定 sleep 制造竞态。
type blockedBackend struct {
	*memoryBackend
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (b *blockedBackend) Get(service, account string) (string, error) {
	b.calls.Add(1)
	select {
	case b.started <- struct{}{}:
	default:
	}
	<-b.release
	return b.memoryBackend.Get(service, account)
}

func TestVaultRepeatedCancellationDoesNotStartAdditionalSystemReads(t *testing.T) {
	backend := &blockedBackend{memoryBackend: &memoryBackend{values: map[string]string{}}, started: make(chan struct{}, 2), release: make(chan struct{})}
	defer close(backend.release)
	vault := New(backend, time.Now)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := make(chan error, 1)
	go func() { _, err := vault.Connect(ctx, "https://service.example"); first <- err }()
	select {
	case <-backend.started:
	case <-time.After(time.Second):
		t.Fatal("system read did not start")
	}
	cancel()
	select {
	case err := <-first:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancellation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("first lookup stayed blocked")
	}
	secondContext, stop := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer stop()
	_, err := vault.Connect(secondContext, "https://other.example")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected bounded second lookup: %v", err)
	}
	if backend.calls.Load() != 1 {
		t.Fatalf("cancelled read spawned another OS prompt: %d", backend.calls.Load())
	}
}

func TestVaultCancellationAfterSystemReadStarted(t *testing.T) {
	backend := &blockedBackend{memoryBackend: &memoryBackend{values: map[string]string{}}, started: make(chan struct{}, 2), release: make(chan struct{})}
	defer close(backend.release)
	vault := New(backend, time.Now)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { _, err := vault.Inspect(ctx, "https://service.example"); result <- err }()
	select {
	case <-backend.started:
	case <-time.After(time.Second):
		t.Fatal("read never entered OS boundary")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancellation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled credential lookup remained blocked")
	}
	if backend.writes != 0 || backend.deletes != 0 {
		t.Fatal("cancelling a read modified credentials")
	}
}

func TestVaultDeadlineAfterSystemReadStarted(t *testing.T) {
	backend := &blockedBackend{memoryBackend: &memoryBackend{values: map[string]string{}}, started: make(chan struct{}, 2), release: make(chan struct{})}
	defer close(backend.release)
	vault := New(backend, time.Now)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := vault.Connect(ctx, "https://service.example")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline: %v", err)
	}
	if backend.calls.Load() != 1 {
		t.Fatal("lookup did not enter OS boundary")
	}
}
