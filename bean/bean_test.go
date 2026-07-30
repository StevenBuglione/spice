package bean

import (
	"context"
	"errors"
	"strings"
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

func TestOverrideValueAndFactory(t *testing.T) {
	t.Parallel()
	value := Replace("test")
	if !value.Enabled() {
		t.Fatal("Replace() returned a disabled override")
	}
	got, cleanup, err := value.Acquire(t.Context())
	if err != nil || got != "test" || cleanup != nil {
		t.Fatalf("value Acquire() = %q, %v, %v", got, cleanup, err)
	}

	var cleaned atomic.Bool
	factory := ReplaceFactory(func(
		context.Context,
	) (string, lifecycle.Cleanup, error) {
		return "factory", func(context.Context) error {
			cleaned.Store(true)
			return nil
		}, nil
	})
	got, cleanup, err = factory.Acquire(t.Context())
	if err != nil || got != "factory" || cleanup == nil {
		t.Fatalf("factory Acquire() = %q, %v, %v", got, cleanup, err)
	}
	if err := cleanup(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !cleaned.Load() {
		t.Fatal("override cleanup did not run")
	}
}

func TestOverrideRejectsInvalidState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		override Override[string]
		nilCtx   bool
		want     string
	}{
		{
			name:     "nil context",
			override: Replace("value"),
			nilCtx:   true,
			want:     "context is nil",
		},
		{
			name:     "disabled",
			override: Override[string]{},
			want:     "disabled",
		},
		{
			name:     "nil factory",
			override: ReplaceFactory[string](nil),
			want:     "factory is nil",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if test.override.Enabled() != (test.name != "disabled") {
				t.Fatalf("Enabled() = %t", test.override.Enabled())
			}
			ctx := t.Context()
			if test.nilCtx {
				ctx = nil
			}
			_, _, err := test.override.Acquire(ctx)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Acquire() error = %v, want %q", err, test.want)
			}
		})
	}
}
