package bean

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/StevenBuglione/spice/lifecycle"
)

func TestOptionalRepresentsPresenceWithoutAmbiguity(t *testing.T) {
	t.Parallel()
	if value, present := Some("ready").Get(); !present || value != "ready" {
		t.Fatalf("Some().Get() = %q, %v", value, present)
	}
	if value, present := None[string]().Get(); present || value != "" {
		t.Fatalf("None().Get() = %q, %v", value, present)
	}
}

func TestLazyResolvesOnceAndRetainsFailure(t *testing.T) {
	t.Parallel()
	want := errors.New("unavailable")
	var calls atomic.Int32
	lazy := NewLazy(func(context.Context) (string, error) {
		calls.Add(1)
		return "", want
	})
	for range 2 {
		if _, err := lazy.Get(context.Background()); !errors.Is(err, want) {
			t.Fatalf("Get() error = %v", err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("resolver calls = %d", calls.Load())
	}
}

func TestProviderValidatesContextAndReturnsIdempotentCleanup(t *testing.T) {
	t.Parallel()
	var cleanupCalls atomic.Int32
	provider := NewProvider(func(
		context.Context,
	) (string, lifecycle.Cleanup, error) {
		return "value", func(context.Context) error {
			cleanupCalls.Add(1)
			return nil
		}, nil
	})
	//nolint:staticcheck // The public contract deliberately rejects a nil context.
	if _, _, err := provider.Acquire(nil); err == nil {
		t.Fatal("Acquire(nil) error = nil")
	}
	value, cleanup, err := provider.Acquire(context.Background())
	if err != nil || value != "value" {
		t.Fatalf("Acquire() = %q, %v", value, err)
	}
	if err := cleanup(context.Background()); err != nil {
		t.Fatalf("cleanup() error = %v", err)
	}
	if err := cleanup(context.Background()); err != nil {
		t.Fatalf("second cleanup() error = %v", err)
	}
	if cleanupCalls.Load() != 1 {
		t.Fatalf("cleanup calls = %d", cleanupCalls.Load())
	}
}

func TestHandlesFailClosedOnNilState(t *testing.T) {
	t.Parallel()
	var lazy *Lazy[string]
	if _, err := lazy.Get(context.Background()); err == nil {
		t.Fatal("nil Lazy.Get() error = nil")
	}
	empty := NewLazy[string](nil)
	if _, err := empty.Get(context.Background()); err == nil {
		t.Fatal("nil resolver error = nil")
	}
	var provider Provider[string]
	if _, _, err := provider.Acquire(context.Background()); err == nil {
		t.Fatal("nil provider error = nil")
	}
}
