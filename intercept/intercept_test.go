package intercept

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestChainExecutesInDeclaredOrder(t *testing.T) {
	t.Parallel()
	var order []string
	terminal := func(_ context.Context, request string) (string, error) {
		order = append(order, "terminal:"+request)
		return request + ":response", nil
	}
	first := func(
		ctx context.Context,
		request string,
		next Invocation[string, string],
	) (string, error) {
		order = append(order, "first:before")
		response, err := next(ctx, request+":first")
		order = append(order, "first:after")
		return response, err
	}
	second := func(
		ctx context.Context,
		request string,
		next Invocation[string, string],
	) (string, error) {
		order = append(order, "second:before")
		response, err := next(ctx, request+":second")
		order = append(order, "second:after")
		return response, err
	}
	invocation, err := Chain(terminal, first, second)
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
	response, err := invocation(context.Background(), "request")
	if err != nil || response != "request:first:second:response" {
		t.Fatalf("invocation() = %q, %v", response, err)
	}
	want := "first:before,second:before,terminal:request:first:second,second:after,first:after"
	if got := strings.Join(order, ","); got != want {
		t.Fatalf("order = %q, want %q", got, want)
	}
}

func TestChainSupportsShortCircuitAndError(t *testing.T) {
	t.Parallel()
	terminalCalled := false
	invocation, err := Chain(
		func(context.Context, int) (int, error) {
			terminalCalled = true
			return 0, errors.New("terminal")
		},
		func(
			context.Context,
			int,
			Invocation[int, int],
		) (int, error) {
			return 42, nil
		},
	)
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
	value, err := invocation(context.Background(), 1)
	if err != nil || value != 42 || terminalCalled {
		t.Fatalf("invocation() = %d, %v, terminalCalled=%t", value, err, terminalCalled)
	}
}

func TestChainRejectsInvalidConstructionAndContext(t *testing.T) {
	t.Parallel()
	if _, err := Chain[int, int](nil); err == nil {
		t.Fatal("Chain(nil) error = nil")
	}
	terminal := func(_ context.Context, value int) (int, error) { return value, nil }
	if _, err := Chain(terminal, Interceptor[int, int](nil)); err == nil {
		t.Fatal("Chain(nil interceptor) error = nil")
	}
	invocation, err := Chain(terminal)
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
	if _, err := invocation(nil, 1); err == nil || !strings.Contains(err.Error(), "context is nil") {
		t.Fatalf("invocation(nil) error = %v", err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := invocation(canceled, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("invocation(canceled) error = %v", err)
	}
}

func ExampleChain() {
	invocation, err := Chain(
		func(_ context.Context, request string) (string, error) {
			return "hello " + request, nil
		},
		func(
			ctx context.Context,
			request string,
			next Invocation[string, string],
		) (string, error) {
			return next(ctx, strings.ToUpper(request))
		},
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	response, err := invocation(context.Background(), "spice")
	fmt.Println(response, err)
	// Output: hello SPICE <nil>
}

func BenchmarkChainInvocation(b *testing.B) {
	terminal := func(_ context.Context, request int) (int, error) {
		return request + 1, nil
	}
	decorator := func(
		ctx context.Context,
		request int,
		next Invocation[int, int],
	) (int, error) {
		return next(ctx, request)
	}
	invocation, err := Chain(terminal, decorator, decorator, decorator)
	if err != nil {
		b.Fatalf("Chain() error = %v", err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := invocation(context.Background(), 41); err != nil {
			b.Fatalf("invocation error = %v", err)
		}
	}
}
