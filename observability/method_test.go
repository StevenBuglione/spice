package observability

import (
	"context"
	"errors"
	"slices"
	"testing"
)

type methodObserver struct {
	name   string
	events *[]string
	result *MethodResult
}

func (observer methodObserver) BeginMethod(
	ctx context.Context,
	_ MethodDefinition,
) (context.Context, func(MethodResult)) {
	*observer.events = append(*observer.events, "begin "+observer.name)
	return ctx, func(result MethodResult) {
		*observer.events = append(*observer.events, "finish "+observer.name)
		*observer.result = result
	}
}

func TestObserveComposesObserversAndReturnsWorkError(t *testing.T) {
	t.Parallel()
	definition := validMethodDefinition()
	workErr := errors.New("work failed")
	var events []string
	var first, second MethodResult
	err := Observe(context.Background(), definition, []MethodObserver{
		methodObserver{name: "first", events: &events, result: &first},
		methodObserver{name: "second", events: &events, result: &second},
	}, func(context.Context) error {
		events = append(events, "work")
		return workErr
	})
	if !errors.Is(err, workErr) {
		t.Fatalf("Observe() error = %v", err)
	}
	if want := []string{
		"begin first", "begin second", "work", "finish second", "finish first",
	}; !slices.Equal(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
	for _, result := range []MethodResult{first, second} {
		if result.Definition != definition || !errors.Is(result.Err, workErr) ||
			result.Duration < 0 || result.Panicked {
			t.Fatalf("method result = %#v", result)
		}
	}
}

func TestObserveReportsAndReraisesPanic(t *testing.T) {
	t.Parallel()
	panicValue := &struct{ message string }{"boom"}
	var events []string
	var result MethodResult
	recovered := recoverObserved(func() {
		if err := Observe(context.Background(), validMethodDefinition(), []MethodObserver{
			methodObserver{name: "observer", events: &events, result: &result},
		}, func(context.Context) error {
			panic(panicValue)
		}); err != nil {
			t.Fatalf("Observe() error = %v, want panic", err)
		}
	})
	if recovered != panicValue || !result.Panicked ||
		!errors.Is(result.Err, ErrMethodPanicked) {
		t.Fatalf("recovered = %#v, result = %#v", recovered, result)
	}
}

func TestObserveRejectsInvalidInputs(t *testing.T) {
	t.Parallel()
	definition := validMethodDefinition()
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "nil context", run: func() error {
			return Observe(nil, definition, nil, func(context.Context) error { //nolint:staticcheck // Nil context is an intentional fail-closed boundary case.
				return nil
			})
		}},
		{name: "definition", run: func() error {
			return Observe(context.Background(), MethodDefinition{}, nil, func(context.Context) error { return nil })
		}},
		{name: "nil work", run: func() error {
			return Observe(context.Background(), definition, nil, nil)
		}},
		{name: "nil observer", run: func() error {
			return Observe(context.Background(), definition, []MethodObserver{nil}, func(context.Context) error { return nil })
		}},
		{name: "nil observer context", run: func() error {
			return Observe(context.Background(), definition, []MethodObserver{nilContextMethodObserver{}}, func(context.Context) error { return nil })
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := test.run(); err == nil {
				t.Fatal("Observe() error = nil")
			}
		})
	}
}

type nilContextMethodObserver struct{}

func (nilContextMethodObserver) BeginMethod(
	context.Context,
	MethodDefinition,
) (context.Context, func(MethodResult)) {
	return nil, nil
}

func validMethodDefinition() MethodDefinition {
	return MethodDefinition{
		ID:      "example.com/orders.DefaultOrderService.Create",
		Module:  "example.com/orders",
		Service: "DefaultOrderService",
		Method:  "Create",
	}
}

func recoverObserved(action func()) (recovered any) {
	defer func() {
		recovered = recover()
	}()
	action()
	return nil
}
