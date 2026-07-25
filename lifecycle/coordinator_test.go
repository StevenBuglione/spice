package lifecycle

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestCoordinatorStartsAndStopsInDeterministicOrder(t *testing.T) {
	t.Parallel()
	coordinator := NewCoordinator()
	var trace []string
	appendTrace := func(value string) Cleanup {
		return func(context.Context) error {
			trace = append(trace, value)
			return nil
		}
	}
	if err := coordinator.RegisterCleanup("dependency", appendTrace("cleanup dependency")); err != nil {
		t.Fatalf("RegisterCleanup(dependency) error = %v", err)
	}
	if err := coordinator.RegisterCleanup("consumer", appendTrace("cleanup consumer")); err != nil {
		t.Fatalf("RegisterCleanup(consumer) error = %v", err)
	}
	if got := coordinator.State(); got != StateConstructed {
		t.Fatalf("State() = %q, want %q", got, StateConstructed)
	}

	hooks := []Hook{
		{
			ID:    "dependency",
			Start: appendTrace("start dependency"),
			Stop:  appendTrace("stop dependency"),
		},
		{
			ID:    "consumer",
			Start: appendTrace("start consumer"),
			Stop:  appendTrace("stop consumer"),
		},
	}
	if err := coordinator.Start(context.Background(), hooks); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if got := coordinator.State(); got != StateReady {
		t.Fatalf("State() = %q, want %q", got, StateReady)
	}
	if err := coordinator.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if err := coordinator.Stop(context.Background()); err != nil {
		t.Fatalf("second Stop() error = %v", err)
	}
	if got := coordinator.State(); got != StateStopped {
		t.Fatalf("State() = %q, want %q", got, StateStopped)
	}
	want := []string{
		"start dependency",
		"start consumer",
		"stop consumer",
		"stop dependency",
		"cleanup consumer",
		"cleanup dependency",
	}
	if !slices.Equal(trace, want) {
		t.Fatalf("trace = %v, want %v", trace, want)
	}
}

func TestCoordinatorStartupFailureRollsBackAndJoinsErrors(t *testing.T) {
	t.Parallel()
	startFailure := errors.New("start failed")
	stopFailure := errors.New("stop failed")
	cleanupFailure := errors.New("cleanup failed")
	coordinator := NewCoordinator()
	var trace []string
	register := func(id string, failure error) {
		t.Helper()
		err := coordinator.RegisterCleanup(id, func(context.Context) error {
			trace = append(trace, "cleanup "+id)
			return failure
		})
		if err != nil {
			t.Fatalf("RegisterCleanup(%q) error = %v", id, err)
		}
	}
	register("dependency", nil)
	register("failing", nil)
	register("later", cleanupFailure)

	err := coordinator.Start(context.Background(), []Hook{
		{
			ID: "dependency",
			Start: func(context.Context) error {
				trace = append(trace, "start dependency")
				return nil
			},
			Stop: func(context.Context) error {
				trace = append(trace, "stop dependency")
				return stopFailure
			},
		},
		{
			ID: "failing",
			Start: func(context.Context) error {
				trace = append(trace, "start failing")
				return startFailure
			},
			Stop: func(context.Context) error {
				trace = append(trace, "stop failing")
				return nil
			},
		},
		{
			ID: "later",
			Start: func(context.Context) error {
				trace = append(trace, "start later")
				return nil
			},
		},
	})
	for _, expected := range []error{startFailure, stopFailure, cleanupFailure} {
		if !errors.Is(err, expected) {
			t.Fatalf("Start() error %v does not contain %v", err, expected)
		}
	}
	if got := coordinator.State(); got != StateFailed {
		t.Fatalf("State() = %q, want %q", got, StateFailed)
	}
	want := []string{
		"start dependency",
		"start failing",
		"stop dependency",
		"cleanup later",
		"cleanup failing",
		"cleanup dependency",
	}
	if !slices.Equal(trace, want) {
		t.Fatalf("trace = %v, want %v", trace, want)
	}
	before := append([]string(nil), trace...)
	stopErr := coordinator.Stop(context.Background())
	for _, expected := range []error{startFailure, stopFailure, cleanupFailure} {
		if !errors.Is(stopErr, expected) {
			t.Fatalf("Stop() error %v does not contain %v", stopErr, expected)
		}
	}
	if !slices.Equal(trace, before) {
		t.Fatalf("idempotent Stop changed trace from %v to %v", before, trace)
	}
}

func TestCoordinatorAbortRollsBackConstruction(t *testing.T) {
	t.Parallel()
	constructionFailure := errors.New("construction failed")
	cleanupFailure := errors.New("cleanup failed")
	coordinator := NewCoordinator()
	var trace []string
	for _, item := range []struct {
		id      string
		failure error
	}{
		{id: "first"},
		{id: "second", failure: cleanupFailure},
	} {
		if err := coordinator.RegisterCleanup(item.id, func(context.Context) error {
			trace = append(trace, item.id)
			return item.failure
		}); err != nil {
			t.Fatalf("RegisterCleanup(%q) error = %v", item.id, err)
		}
	}

	err := coordinator.Abort(context.Background(), constructionFailure)
	if !errors.Is(err, constructionFailure) || !errors.Is(err, cleanupFailure) {
		t.Fatalf("Abort() error = %v", err)
	}
	if !slices.Equal(trace, []string{"second", "first"}) {
		t.Fatalf("trace = %v", trace)
	}
	if got := coordinator.State(); got != StateFailed {
		t.Fatalf("State() = %q, want %q", got, StateFailed)
	}
	again := coordinator.Abort(context.Background(), errors.New("again"))
	if !errors.Is(again, ErrInvalidTransition) {
		t.Fatalf("second Abort() error = %v", again)
	}
}

func TestCoordinatorRejectsInvalidHooksWithoutChangingState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		hooks []Hook
		want  string
	}{
		{name: "missing ID", hooks: []Hook{{Start: func(context.Context) error { return nil }}}, want: "has no ID"},
		{name: "missing start", hooks: []Hook{{ID: "missing"}}, want: "has no start callback"},
		{
			name: "duplicate ID",
			hooks: []Hook{
				{ID: "duplicate", Start: func(context.Context) error { return nil }},
				{ID: "duplicate", Start: func(context.Context) error { return nil }},
			},
			want: "duplicate hook ID",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			coordinator := NewCoordinator()
			err := coordinator.Start(context.Background(), test.hooks)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Start() error = %v, want %q", err, test.want)
			}
			if got := coordinator.State(); got != StateConstructed {
				t.Fatalf("State() = %q, want %q", got, StateConstructed)
			}
		})
	}
}

func TestCoordinatorHonorsCanceledAndNilContexts(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	coordinator := NewCoordinator()
	cleanupCalled := false
	if err := coordinator.RegisterCleanup("provider", func(received context.Context) error {
		cleanupCalled = true
		if !errors.Is(received.Err(), context.Canceled) {
			t.Fatalf("cleanup context error = %v", received.Err())
		}
		return nil
	}); err != nil {
		t.Fatalf("RegisterCleanup() error = %v", err)
	}
	if err := coordinator.Start(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("Start(canceled) error = %v", err)
	}
	if !cleanupCalled {
		t.Fatal("canceled Start did not run construction cleanup")
	}

	if err := NewCoordinator().Start(nil, nil); err == nil { //nolint:staticcheck // Explicitly verifies the defensive nil-context contract.
		t.Fatal("Start(nil) error = nil")
	}
	if err := NewCoordinator().Stop(nil); err == nil { //nolint:staticcheck // Explicitly verifies the defensive nil-context contract.
		t.Fatal("Stop(nil) error = nil")
	}
	if err := NewCoordinator().Abort(nil, errors.New("failure")); err == nil { //nolint:staticcheck // Explicitly verifies the defensive nil-context contract.
		t.Fatal("Abort(nil) error = nil")
	}
	if err := NewCoordinator().Abort(context.Background(), nil); err == nil {
		t.Fatal("Abort(nil cause) error = nil")
	}
}

func TestCoordinatorConcurrentTransitionsAreRaceSafe(t *testing.T) {
	t.Parallel()
	coordinator := NewCoordinator()
	startEntered := make(chan struct{})
	releaseStart := make(chan struct{})
	startResult := make(chan error, 1)
	go func() {
		startResult <- coordinator.Start(context.Background(), []Hook{{
			ID: "blocking",
			Start: func(context.Context) error {
				close(startEntered)
				<-releaseStart
				return nil
			},
		}})
	}()
	<-startEntered
	if err := coordinator.Stop(context.Background()); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Stop(during start) error = %v", err)
	}
	if err := coordinator.RegisterCleanup("late", func(context.Context) error { return nil }); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("RegisterCleanup(during start) error = %v", err)
	}
	close(releaseStart)
	if err := <-startResult; err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	stopEntered := make(chan struct{})
	releaseStop := make(chan struct{})
	coordinator.started[0].Stop = func(context.Context) error {
		close(stopEntered)
		<-releaseStop
		return nil
	}
	stopResult := make(chan error, 1)
	go func() {
		stopResult <- coordinator.Stop(context.Background())
	}()
	<-stopEntered

	waitCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := coordinator.Stop(waitCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("concurrent Stop() error = %v", err)
	}
	close(releaseStop)
	if err := <-stopResult; err != nil {
		t.Fatalf("first Stop() error = %v", err)
	}
	if err := coordinator.Stop(context.Background()); err != nil {
		t.Fatalf("idempotent Stop() error = %v", err)
	}
}

func TestCoordinatorUninitializedAndTransitionErrors(t *testing.T) {
	t.Parallel()
	var coordinator *Coordinator
	if got := coordinator.State(); got != StateInvalid {
		t.Fatalf("nil State() = %q", got)
	}
	if err := coordinator.RegisterCleanup("provider", func(context.Context) error { return nil }); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("nil RegisterCleanup() error = %v", err)
	}
	cause := errors.New("construction")
	if err := coordinator.Abort(context.Background(), cause); !errors.Is(err, cause) || !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("nil Abort() error = %v", err)
	}
	if err := coordinator.Start(context.Background(), nil); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("nil Start() error = %v", err)
	}
	if err := coordinator.Stop(context.Background()); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("nil Stop() error = %v", err)
	}

	transition := (*TransitionError)(nil)
	if transition.Error() != ErrInvalidTransition.Error() {
		t.Fatalf("nil TransitionError.Error() = %q", transition.Error())
	}
	if !errors.Is(&TransitionError{Operation: "start", State: StateReady}, ErrInvalidTransition) {
		t.Fatal("errors.Is did not recognize TransitionError")
	}
}

func TestCoordinatorStopBeforeStartRunsCleanup(t *testing.T) {
	t.Parallel()
	coordinator := NewCoordinator()
	called := false
	if err := coordinator.RegisterCleanup("provider", func(context.Context) error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("RegisterCleanup() error = %v", err)
	}
	if err := coordinator.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if !called {
		t.Fatal("Stop before Start did not run cleanup")
	}
	if got := coordinator.State(); got != StateStopped {
		t.Fatalf("State() = %q, want %q", got, StateStopped)
	}
}
