package schedule

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var (
	errAlpha = errors.New("alpha failed")
	errZulu  = errors.New("zulu failed")
)

func TestSchedulerRunsSerialFixedDelay(t *testing.T) {
	t.Parallel()
	lifetime, cancel := context.WithCancel(context.Background())
	var waitsMu sync.Mutex
	var waits []time.Duration
	waiter := func(context.Context, time.Duration) error {
		waitsMu.Lock()
		defer waitsMu.Unlock()
		waits = append(waits, 5*time.Millisecond)
		return nil
	}
	var concurrent atomic.Int64
	var maximum atomic.Int64
	var runs atomic.Uint64
	scheduler := newTestScheduler(t, lifetime, []Job{{
		Definition:   jobDefinition("refresh"),
		InitialDelay: 5 * time.Millisecond,
		Delay:        5 * time.Millisecond,
		Run: func(context.Context) error {
			current := concurrent.Add(1)
			maximum.Store(max(maximum.Load(), current))
			run := runs.Add(1)
			concurrent.Add(-1)
			if run == 3 {
				cancel()
			}
			return nil
		},
	}}, waiter)
	if err := scheduler.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	<-scheduler.Done()
	if err := scheduler.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if runs.Load() != 3 || maximum.Load() != 1 {
		t.Fatalf("runs = %d, max concurrent = %d", runs.Load(), maximum.Load())
	}
	waitsMu.Lock()
	defer waitsMu.Unlock()
	if len(waits) != 3 {
		t.Fatalf("waits = %v, want initial plus two fixed delays", waits)
	}
}

func TestSchedulerContinuesOnlyWhenExplicit(t *testing.T) {
	t.Parallel()
	lifetime, cancel := context.WithCancel(context.Background())
	var observations []Result
	var runs atomic.Uint64
	scheduler := newTestScheduler(t, lifetime, []Job{{
		Definition:      jobDefinition("continue"),
		Delay:           time.Millisecond,
		ContinueOnError: true,
		Run: func(context.Context) error {
			if runs.Add(1) == 1 {
				return errAlpha
			}
			cancel()
			return nil
		},
	}}, func(context.Context, time.Duration) error { return nil }, func(_ context.Context, result Result) {
		observations = append(observations, result)
	})
	if err := scheduler.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	<-scheduler.Done()
	if err := scheduler.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if len(observations) != 2 || !errors.Is(observations[0].Err, errAlpha) {
		t.Fatalf("observations = %#v", observations)
	}
	snapshot := scheduler.Snapshot()
	if snapshot.Runs != 2 || snapshot.Failed != 1 {
		t.Fatalf("Snapshot() = %#v", snapshot)
	}
}

func TestSchedulerAggregatesTerminalErrorsInJobOrder(t *testing.T) {
	t.Parallel()
	scheduler := newTestScheduler(t, context.Background(), []Job{
		{Definition: jobDefinition("zulu"), Delay: time.Hour, Run: errorJob(errZulu)},
		{Definition: jobDefinition("alpha"), Delay: time.Hour, Run: errorJob(errAlpha)},
	}, nil)
	if err := scheduler.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	<-scheduler.Done()
	err := scheduler.Shutdown(context.Background())
	if !errors.Is(err, errAlpha) || !errors.Is(err, errZulu) {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if strings.Index(err.Error(), "alpha") > strings.Index(err.Error(), "zulu") {
		t.Fatalf("Shutdown() error order = %v", err)
	}
}

func TestSchedulerContainsPanic(t *testing.T) {
	t.Parallel()
	var observed Result
	scheduler := newTestScheduler(t, context.Background(), []Job{{
		Definition: jobDefinition("panic"),
		Delay:      time.Hour,
		Run: func(context.Context) error {
			panic("secret value")
		},
	}}, nil, func(_ context.Context, result Result) {
		observed = result
	})
	if err := scheduler.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	<-scheduler.Done()
	err := scheduler.Shutdown(context.Background())
	panicErr, ok := errors.AsType[*PanicError](err)
	if !ok || !errors.Is(err, ErrPanicked) {
		t.Fatalf("Shutdown() error = %#v", err)
	}
	if strings.Contains(panicErr.Error(), "secret value") ||
		!observed.Panicked ||
		!errors.Is(observed.Err, ErrPanicked) {
		t.Fatalf("panic result = %#v, observation = %#v", panicErr, observed)
	}
	if (*PanicError)(nil).Error() != ErrPanicked.Error() ||
		!errors.Is((*PanicError)(nil), ErrPanicked) {
		t.Fatal("nil PanicError contract changed")
	}
}

func TestShutdownDrainsThenDeadlineCancels(t *testing.T) {
	t.Parallel()
	started := make(chan struct{})
	release := make(chan struct{})
	scheduler := newTestScheduler(t, context.Background(), []Job{{
		Definition: jobDefinition("drain"),
		Delay:      time.Hour,
		Run: func(ctx context.Context) error {
			close(started)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return context.Cause(ctx)
			}
		},
	}}, nil)
	if err := scheduler.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	<-started
	shutdown := make(chan error, 1)
	go func() {
		shutdown <- scheduler.Shutdown(context.Background())
	}()
	select {
	case err := <-shutdown:
		t.Fatalf("Shutdown() returned before current run drained: %v", err)
	case <-time.After(10 * time.Millisecond):
	}
	close(release)
	if err := <-shutdown; err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	otherStarted := make(chan struct{})
	taskCanceled := make(chan error, 1)
	other := newTestScheduler(t, context.Background(), []Job{{
		Definition: jobDefinition("cancel"),
		Delay:      time.Hour,
		Run: func(ctx context.Context) error {
			close(otherStarted)
			<-ctx.Done()
			taskCanceled <- context.Cause(ctx)
			return context.Cause(ctx)
		},
	}}, nil)
	if err := other.Start(context.Background()); err != nil {
		t.Fatalf("other.Start() error = %v", err)
	}
	<-otherStarted
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := other.Shutdown(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("other.Shutdown() error = %v", err)
	}
	if err := <-taskCanceled; !errors.Is(err, context.Canceled) {
		t.Fatalf("task cancellation = %v", err)
	}
	<-other.Done()
}

func TestSchedulerValidatesStateAndInputs(t *testing.T) {
	t.Parallel()
	valid := Job{Definition: jobDefinition("valid"), Delay: time.Second, Run: noJob}
	if _, err := New(nilTestContext(), []Job{valid}, nil); err == nil {
		t.Fatal("New(nil lifetime) error = nil")
	}
	tests := []struct {
		name      string
		jobs      []Job
		observers []Observer
	}{
		{"missing ID", []Job{{Definition: Definition{Module: "module"}, Delay: time.Second, Run: noJob}}, nil},
		{"missing module", []Job{{Definition: Definition{ID: "job"}, Delay: time.Second, Run: noJob}}, nil},
		{"zero delay", []Job{{Definition: jobDefinition("job"), Run: noJob}}, nil},
		{"negative initial", []Job{{Definition: jobDefinition("job"), InitialDelay: -1, Delay: time.Second, Run: noJob}}, nil},
		{"nil function", []Job{{Definition: jobDefinition("job"), Delay: time.Second}}, nil},
		{"duplicate ID", []Job{valid, valid}, nil},
		{"nil observer", []Job{valid}, []Observer{nil}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := New(context.Background(), test.jobs, nil, test.observers...); err == nil {
				t.Fatal("New() error = nil")
			}
		})
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New(canceled, []Job{valid}, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("New(canceled) error = %v", err)
	}
	scheduler := newTestScheduler(t, context.Background(), nil, nil)
	if err := scheduler.Start(nilTestContext()); err == nil {
		t.Fatal("Start(nil context) error = nil")
	}
	if err := scheduler.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := scheduler.Start(context.Background()); !errors.Is(err, ErrStarted) {
		t.Fatalf("Start(twice) error = %v", err)
	}
	if err := scheduler.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := scheduler.Start(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatalf("Start(after shutdown) error = %v", err)
	}
}

func TestUnstartedAndNilSchedulerShutdown(t *testing.T) {
	t.Parallel()
	scheduler := newTestScheduler(t, context.Background(), nil, nil)
	if err := scheduler.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown(unstarted) error = %v", err)
	}
	if err := scheduler.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown(twice) error = %v", err)
	}
	if err := (*Scheduler)(nil).Start(context.Background()); err == nil {
		t.Fatal("nil Start() error = nil")
	}
	if err := (*Scheduler)(nil).Shutdown(context.Background()); err == nil {
		t.Fatal("nil Shutdown() error = nil")
	}
	if err := scheduler.Shutdown(nilTestContext()); err == nil {
		t.Fatal("Shutdown(nil context) error = nil")
	}
	if !(*Scheduler)(nil).Snapshot().Closed {
		t.Fatal("nil Snapshot() is not closed")
	}
	select {
	case <-(*Scheduler)(nil).Done():
	default:
		t.Fatal("nil Done() is not closed")
	}
}

func newTestScheduler(
	t *testing.T,
	lifetime context.Context,
	jobs []Job,
	waiter Waiter,
	observers ...Observer,
) *Scheduler {
	t.Helper()
	scheduler, err := New(lifetime, jobs, waiter, observers...)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return scheduler
}

func jobDefinition(id string) Definition {
	return Definition{ID: id, Module: "example.com/shop"}
}

func errorJob(err error) func(context.Context) error {
	return func(context.Context) error { return err }
}

func noJob(context.Context) error {
	return nil
}

func nilTestContext() context.Context {
	return nil
}
