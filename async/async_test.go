package async

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	errFirst  = errors.New("first failed")
	errSecond = errors.New("second failed")
)

func TestExecutorAppliesBackpressureAndObservesTasks(t *testing.T) {
	t.Parallel()
	var observationsMu sync.Mutex
	var observations []Result
	executor := newTestExecutor(t, context.Background(), 1, func(_ context.Context, result Result) {
		observationsMu.Lock()
		observations = append(observations, result)
		observationsMu.Unlock()
	})
	release := make(chan struct{})
	started := make(chan struct{})
	if err := executor.Submit(context.Background(), taskDefinition("first"), func(context.Context) error {
		close(started)
		<-release
		return nil
	}); err != nil {
		t.Fatalf("Submit(first) error = %v", err)
	}
	<-started
	submitted := make(chan error, 1)
	go func() {
		submitted <- executor.Submit(
			context.Background(),
			taskDefinition("second"),
			func(context.Context) error { return nil },
		)
	}()
	select {
	case err := <-submitted:
		t.Fatalf("second Submit() completed before a slot was free: %v", err)
	case <-time.After(10 * time.Millisecond):
	}
	close(release)
	if err := <-submitted; err != nil {
		t.Fatalf("Submit(second) error = %v", err)
	}
	if err := executor.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if snapshot := executor.Snapshot(); snapshot != (Snapshot{
		Submitted: 2,
		Completed: 2,
		Closed:    true,
	}) {
		t.Fatalf("Snapshot() = %#v", snapshot)
	}
	observationsMu.Lock()
	defer observationsMu.Unlock()
	if len(observations) != 2 ||
		observations[0].Definition.ID != "first" ||
		observations[1].Definition.ID != "second" {
		t.Fatalf("observations = %#v", observations)
	}
}

func TestShutdownReturnsErrorsInSubmissionOrder(t *testing.T) {
	t.Parallel()
	executor := newTestExecutor(t, context.Background(), 2)
	releaseFirst := make(chan struct{})
	if err := executor.Submit(context.Background(), taskDefinition("first"), func(context.Context) error {
		<-releaseFirst
		return errFirst
	}); err != nil {
		t.Fatalf("Submit(first) error = %v", err)
	}
	if err := executor.Submit(context.Background(), taskDefinition("second"), func(context.Context) error {
		return errSecond
	}); err != nil {
		t.Fatalf("Submit(second) error = %v", err)
	}
	close(releaseFirst)
	err := executor.Shutdown(context.Background())
	if !errors.Is(err, errFirst) || !errors.Is(err, errSecond) {
		t.Fatalf("Shutdown() error = %v, want both task errors", err)
	}
	if strings.Index(err.Error(), "first") > strings.Index(err.Error(), "second") {
		t.Fatalf("Shutdown() error order = %v", err)
	}
}

func TestExecutorContainsPanic(t *testing.T) {
	t.Parallel()
	var observed Result
	executor := newTestExecutor(t, context.Background(), 1, func(_ context.Context, result Result) {
		observed = result
	})
	if err := executor.Submit(context.Background(), taskDefinition("panic"), func(context.Context) error {
		panic("secret panic value")
	}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	err := executor.Shutdown(context.Background())
	var panicErr *PanicError
	if !errors.Is(err, ErrPanicked) || !errors.As(err, &panicErr) {
		t.Fatalf("Shutdown() error = %#v", err)
	}
	if strings.Contains(err.Error(), "secret panic value") {
		t.Fatal("Shutdown() exposed the recovered panic value")
	}
	if !observed.Panicked || !errors.Is(observed.Err, ErrPanicked) {
		t.Fatalf("observation = %#v", observed)
	}
	snapshot := executor.Snapshot()
	if snapshot.Panicked != 1 || snapshot.Failed != 1 || snapshot.Completed != 1 {
		t.Fatalf("Snapshot() = %#v", snapshot)
	}
	if (*PanicError)(nil).Error() != ErrPanicked.Error() ||
		!errors.Is((*PanicError)(nil), ErrPanicked) {
		t.Fatal("nil PanicError contract changed")
	}
}

func TestShutdownDeadlineCancelsExecutionContext(t *testing.T) {
	t.Parallel()
	executor := newTestExecutor(t, context.Background(), 1)
	taskDone := make(chan error, 1)
	if err := executor.Submit(context.Background(), taskDefinition("wait"), func(ctx context.Context) error {
		<-ctx.Done()
		taskDone <- context.Cause(ctx)
		return context.Cause(ctx)
	}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	shutdownContext, cancel := context.WithCancel(context.Background())
	cancel()
	err := executor.Shutdown(shutdownContext)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Shutdown() error = %v, want context.Canceled", err)
	}
	if taskErr := <-taskDone; !errors.Is(taskErr, context.Canceled) {
		t.Fatalf("task error = %v, want context.Canceled", taskErr)
	}
	<-executor.Done()
}

func TestSubmitHonorsAdmissionAndLifetimeCancellation(t *testing.T) {
	t.Parallel()
	executor := newTestExecutor(t, context.Background(), 1)
	release := make(chan struct{})
	if err := executor.Submit(context.Background(), taskDefinition("running"), func(context.Context) error {
		<-release
		return nil
	}); err != nil {
		t.Fatalf("Submit(running) error = %v", err)
	}
	admission, cancelAdmission := context.WithCancel(context.Background())
	cancelAdmission()
	if err := executor.Submit(admission, taskDefinition("blocked"), func(context.Context) error {
		return nil
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Submit(canceled admission) error = %v", err)
	}
	close(release)
	if err := executor.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := executor.Submit(context.Background(), taskDefinition("late"), func(context.Context) error {
		return nil
	}); !errors.Is(err, ErrClosed) {
		t.Fatalf("Submit(after shutdown) error = %v", err)
	}

	lifetime, cancelLifetime := context.WithCancel(context.Background())
	other := newTestExecutor(t, lifetime, 1)
	cancelLifetime()
	if err := other.Submit(context.Background(), taskDefinition("canceled"), func(context.Context) error {
		return nil
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Submit(canceled lifetime) error = %v", err)
	}
}

func TestExecutorValidatesInputs(t *testing.T) {
	t.Parallel()
	if _, err := NewExecutor(nilTestContext(), 1); err == nil {
		t.Fatal("NewExecutor(nil context) error = nil")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewExecutor(canceled, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("NewExecutor(canceled) error = %v", err)
	}
	if _, err := NewExecutor(context.Background(), 0); err == nil {
		t.Fatal("NewExecutor(zero concurrency) error = nil")
	}
	if _, err := NewExecutor(context.Background(), 1, nil); err == nil {
		t.Fatal("NewExecutor(nil observer) error = nil")
	}
	executor := newTestExecutor(t, context.Background(), 1)
	if err := (*Executor)(nil).Submit(
		context.Background(),
		taskDefinition("task"),
		noTask,
	); err == nil {
		t.Fatal("nil executor Submit() error = nil")
	}
	if err := executor.Submit(nilTestContext(), taskDefinition("task"), noTask); err == nil {
		t.Fatal("Submit(nil admission) error = nil")
	}
	tests := []struct {
		name       string
		definition Definition
		task       Task
	}{
		{"missing ID", Definition{Module: "example.com/shop"}, noTask},
		{"missing module", Definition{ID: "task"}, noTask},
		{"nil task", taskDefinition("task"), nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := executor.Submit(context.Background(), test.definition, test.task); err == nil {
				t.Fatal("Submit() error = nil")
			}
		})
	}
	if err := (*Executor)(nil).Shutdown(context.Background()); err == nil {
		t.Fatal("nil Shutdown() error = nil")
	}
	if err := executor.Shutdown(nilTestContext()); err == nil {
		t.Fatal("Shutdown(nil context) error = nil")
	}
	if snapshot := (*Executor)(nil).Snapshot(); !snapshot.Closed {
		t.Fatalf("nil Snapshot() = %#v", snapshot)
	}
	select {
	case <-(*Executor)(nil).Done():
	default:
		t.Fatal("nil Done() is not closed")
	}
}

func TestShutdownIsIdempotent(t *testing.T) {
	t.Parallel()
	executor := newTestExecutor(t, context.Background(), 1)
	results := make(chan error, 2)
	for range 2 {
		go func() {
			results <- executor.Shutdown(context.Background())
		}()
	}
	got := []error{<-results, <-results}
	if !slices.EqualFunc(got, []error{nil, nil}, errors.Is) {
		t.Fatalf("Shutdown() results = %v", got)
	}
}

func newTestExecutor(
	t *testing.T,
	ctx context.Context,
	maxConcurrent int,
	observers ...Observer,
) *Executor {
	t.Helper()
	executor, err := NewExecutor(ctx, maxConcurrent, observers...)
	if err != nil {
		t.Fatalf("NewExecutor() error = %v", err)
	}
	return executor
}

func taskDefinition(id string) Definition {
	return Definition{ID: id, Module: "example.com/shop/orders"}
}

func noTask(context.Context) error {
	return nil
}

func nilTestContext() context.Context {
	return nil
}
