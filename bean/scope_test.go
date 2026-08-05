package bean

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/spice-framework/spice/lifecycle"
)

func TestScopeClosesInReverseOrderAndIsIdempotent(t *testing.T) {
	t.Parallel()
	scope, err := NewScope(ScopeRequest)
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, name := range []string{"first", "second"} {
		value := name
		if err := scope.Register(func(context.Context) error {
			order = append(order, value)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := scope.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := scope.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(order, []string{"second", "first"}) {
		t.Fatalf("cleanup order = %v", order)
	}
	if err := scope.Register(func(context.Context) error {
		return nil
	}); err == nil {
		t.Fatal("Register() after Close error = nil")
	}
}

func TestScopedProviderConstructsOncePerScopeConcurrently(t *testing.T) {
	t.Parallel()
	var constructs atomic.Int32
	var cleanups atomic.Int32
	scoped := NewScoped(
		ScopeSession,
		func(context.Context) (*int, lifecycle.Cleanup, error) {
			value := int(constructs.Add(1))
			return &value, func(context.Context) error {
				cleanups.Add(1)
				return nil
			}, nil
		},
	)
	scope, err := NewScope(ScopeSession)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := WithScope(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	provider := scoped.Provider()
	var wait sync.WaitGroup
	values := make(chan *int, 16)
	for range 16 {
		wait.Go(func() {
			value, cleanup, acquireErr := provider.Acquire(ctx)
			if acquireErr != nil {
				t.Error(acquireErr)
				return
			}
			if cleanupErr := cleanup(ctx); cleanupErr != nil {
				t.Error(cleanupErr)
			}
			values <- value
		})
	}
	wait.Wait()
	close(values)
	var first *int
	for value := range values {
		if first == nil {
			first = value
		}
		if value != first {
			t.Fatal("scoped provider returned distinct values")
		}
	}
	if constructs.Load() != 1 || cleanups.Load() != 0 {
		t.Fatalf(
			"before close constructs=%d cleanups=%d",
			constructs.Load(),
			cleanups.Load(),
		)
	}
	if err := scope.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cleanups.Load() != 1 {
		t.Fatalf("cleanups = %d", cleanups.Load())
	}
	if _, _, err := provider.Acquire(ctx); err == nil ||
		!strings.Contains(err.Error(), "scope is closed") {
		t.Fatalf("Acquire() after close error = %v", err)
	}
}

func TestScopedProviderFailsWithoutMatchingScopeAndOnCancellation(t *testing.T) {
	t.Parallel()
	scoped := NewScoped(
		ScopeRequest,
		func(context.Context) (string, lifecycle.Cleanup, error) {
			return "value", nil, nil
		},
	)
	if _, _, err := scoped.Provider().Acquire(
		context.Background(),
	); err == nil {
		t.Fatal("Acquire() without scope error = nil")
	}
	session, err := NewScope(ScopeSession)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := WithScope(context.Background(), session)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := scoped.Provider().Acquire(ctx); err == nil {
		t.Fatal("Acquire() with wrong scope error = nil")
	}
}

func TestRequestScopeMiddlewareOwnsCleanup(t *testing.T) {
	t.Parallel()
	scoped := NewScoped(
		ScopeRequest,
		func(context.Context) (string, lifecycle.Cleanup, error) {
			return "request", func(context.Context) error {
				return errors.New("cleanup failed")
			}, nil
		},
	)
	var observed error
	handler, err := RequestScopeMiddleware(
		http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			value, cleanup, acquireErr := scoped.Provider().Acquire(
				request.Context(),
			)
			if acquireErr != nil {
				t.Error(acquireErr)
				return
			}
			if value != "request" {
				t.Errorf("value = %q", value)
			}
			if cleanupErr := cleanup(request.Context()); cleanupErr != nil {
				t.Error(cleanupErr)
			}
			writer.WriteHeader(http.StatusNoContent)
		}),
		func(_ *http.Request, err error) {
			observed = err
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/", nil),
	)
	if response.Code != http.StatusNoContent ||
		observed == nil ||
		!strings.Contains(observed.Error(), "cleanup failed") {
		t.Fatalf("response=%d cleanup=%v", response.Code, observed)
	}
}

func TestScopeBoundaryValidation(t *testing.T) {
	t.Parallel()
	if _, err := NewScope(ScopeKind("thread")); err == nil {
		t.Fatal("NewScope(thread) error = nil")
	}
	var missing *Scope
	if missing.Kind() != "" {
		t.Fatalf("nil Kind() = %q", missing.Kind())
	}
	if err := missing.Register(noopCleanup()); err == nil {
		t.Fatal("nil Register() error = nil")
	}
	if err := missing.Close(context.Background()); err == nil {
		t.Fatal("nil Close() error = nil")
	}
	scope, err := NewScope(ScopeRequest)
	if err != nil {
		t.Fatal(err)
	}
	if scope.Kind() != ScopeRequest {
		t.Fatalf("Kind() = %q", scope.Kind())
	}
	if err := scope.Register(nil); err == nil {
		t.Fatal("Register(nil) error = nil")
	}
	//nolint:staticcheck // The public contract deliberately rejects a nil context.
	if _, err := WithScope(nil, scope); err == nil {
		t.Fatal("WithScope(nil, scope) error = nil")
	}
	if _, err := WithScope(context.Background(), nil); err == nil {
		t.Fatal("WithScope(ctx, nil) error = nil")
	}
	if err := scope.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := WithScope(
		context.Background(),
		scope,
	); err == nil {
		t.Fatal("WithScope(ctx, closed) error = nil")
	}
	if err := (&Scope{}).Close(context.Background()); err == nil {
		t.Fatal("zero Scope.Close() error = nil")
	}
	//nolint:staticcheck // The public contract deliberately rejects a nil context.
	if err := scope.Close(nil); err == nil {
		t.Fatal("Close(nil) error = nil")
	}
	if _, err := RequestScopeMiddleware(nil, nil); err == nil {
		t.Fatal("RequestScopeMiddleware(nil) error = nil")
	}
}

func TestScopedProviderRejectsInvalidFactoriesAndCleansFailedValues(
	t *testing.T,
) {
	t.Parallel()
	request, err := NewScope(ScopeRequest)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := WithScope(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	var missing *Scoped[string]
	if _, _, err := missing.Provider().Acquire(ctx); err == nil {
		t.Fatal("nil scoped provider error = nil")
	}
	invalid := NewScoped[string](ScopeKind("thread"), nil)
	if _, _, err := invalid.Provider().Acquire(ctx); err == nil {
		t.Fatal("invalid scoped kind error = nil")
	}
	nilFactory := NewScoped[string](ScopeRequest, nil)
	if _, _, err := nilFactory.Provider().Acquire(ctx); err == nil {
		t.Fatal("nil scoped factory error = nil")
	}

	var cleaned atomic.Bool
	want := errors.New("construct failed")
	failing := NewScoped(
		ScopeRequest,
		func(context.Context) (string, lifecycle.Cleanup, error) {
			return "partial", func(context.Context) error {
				cleaned.Store(true)
				return errors.New("cleanup failed")
			}, want
		},
	)
	if _, _, err := failing.Provider().Acquire(ctx); err == nil ||
		!errors.Is(err, want) ||
		!strings.Contains(err.Error(), "cleanup failed") {
		t.Fatalf("failed Acquire() error = %v", err)
	}
	if !cleaned.Load() {
		t.Fatal("failed scoped value was not cleaned")
	}
}
