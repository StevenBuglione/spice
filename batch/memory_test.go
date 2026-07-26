package batch

import (
	"context"
	"errors"
	"math"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
)

func TestMemoryStoreRestartsAndCompletesOrderedExecution(t *testing.T) {
	t.Parallel()

	store := mustMemoryStore(t, 2)
	request := testBeginRequest("instance")
	attempt, beginErr := store.Begin(context.Background(), request)
	if beginErr != nil {
		t.Fatalf("Begin() error = %v", beginErr)
	}
	request.Steps[0] = "changed"
	if attempt.Number() != 1 ||
		attempt.Complete() ||
		len(attempt.CompletedSteps()) != 0 {
		t.Fatalf("first attempt = %#v", attempt)
	}
	if _, err := store.Begin(
		context.Background(),
		testBeginRequest("instance"),
	); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("duplicate Begin() error = %v, want ErrAlreadyRunning", err)
	}
	if err := store.Checkpoint(
		context.Background(),
		attempt,
		"load",
	); err == nil {
		t.Fatal("out-of-order Checkpoint() unexpectedly succeeded")
	}
	if err := store.Checkpoint(
		context.Background(),
		attempt,
		"extract",
	); err != nil {
		t.Fatalf("Checkpoint(extract) error = %v", err)
	}

	snapshot, exists, snapshotErr := store.Snapshot(
		context.Background(),
		attempt.Definition(),
		attempt.Instance(),
	)
	if snapshotErr != nil {
		t.Fatalf("Snapshot() error = %v", snapshotErr)
	}
	if !exists ||
		snapshot.Attempt != 1 ||
		!snapshot.Running ||
		snapshot.Complete ||
		!slices.Equal(snapshot.CompletedSteps, []string{"extract"}) {
		t.Fatalf("running snapshot = %#v, exists = %t", snapshot, exists)
	}
	snapshot.CompletedSteps[0] = "changed"

	failure := Failure{
		Attempt: attempt,
		Step:    "load",
		Kind:    FailureError,
	}
	if err := store.Fail(context.Background(), failure); err != nil {
		t.Fatalf("Fail() error = %v", err)
	}
	if err := store.Checkpoint(
		context.Background(),
		attempt,
		"load",
	); !errors.Is(err, ErrStaleAttempt) {
		t.Fatalf("stale Checkpoint() error = %v, want ErrStaleAttempt", err)
	}

	resumed, resumeErr := store.Begin(
		context.Background(),
		testBeginRequest("instance"),
	)
	if resumeErr != nil {
		t.Fatalf("resumed Begin() error = %v", resumeErr)
	}
	if resumed.Number() != 2 ||
		!slices.Equal(resumed.CompletedSteps(), []string{"extract"}) {
		t.Fatalf("resumed attempt = %#v", resumed)
	}
	if err := store.Checkpoint(
		context.Background(),
		resumed,
		"load",
	); err != nil {
		t.Fatalf("Checkpoint(load) error = %v", err)
	}
	if err := store.Complete(context.Background(), resumed); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	completed, completedErr := store.Begin(
		context.Background(),
		testBeginRequest("instance"),
	)
	if completedErr != nil {
		t.Fatalf("completed Begin() error = %v", completedErr)
	}
	if !completed.Complete() ||
		completed.Number() != 2 ||
		!slices.Equal(completed.CompletedSteps(), []string{"extract", "load"}) {
		t.Fatalf("completed attempt = %#v", completed)
	}
	if err := store.Delete(
		context.Background(),
		completed.Definition(),
		completed.Instance(),
	); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	_, exists, snapshotErr = store.Snapshot(
		context.Background(),
		completed.Definition(),
		completed.Instance(),
	)
	if snapshotErr != nil || exists {
		t.Fatalf(
			"Snapshot() after delete = exists %t, error %v",
			exists,
			snapshotErr,
		)
	}
}

func TestMemoryStoreRejectsDefinitionDriftAndEnforcesCapacity(t *testing.T) {
	t.Parallel()

	store := mustMemoryStore(t, 1)
	first := testBeginRequest("first")
	attempt, beginErr := store.Begin(context.Background(), first)
	if beginErr != nil {
		t.Fatalf("Begin(first) error = %v", beginErr)
	}
	if err := store.Fail(context.Background(), Failure{
		Attempt: attempt,
		Step:    "extract",
		Kind:    FailureCanceled,
	}); err != nil {
		t.Fatalf("Fail(first) error = %v", err)
	}

	changed := testBeginRequest("first")
	changed.Steps = []string{"extract", "publish"}
	if _, err := store.Begin(
		context.Background(),
		changed,
	); !errors.Is(err, ErrDefinitionChanged) {
		t.Fatalf("drifted Begin() error = %v, want ErrDefinitionChanged", err)
	}
	if _, err := store.Begin(
		context.Background(),
		testBeginRequest("second"),
	); !errors.Is(err, ErrCapacity) {
		t.Fatalf("capacity Begin() error = %v, want ErrCapacity", err)
	}

	resumed, err := store.Begin(context.Background(), first)
	if err != nil {
		t.Fatalf("resume first error = %v", err)
	}
	if err := store.Delete(
		context.Background(),
		resumed.Definition(),
		resumed.Instance(),
	); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("active Delete() error = %v, want ErrAlreadyRunning", err)
	}
	if err := store.Fail(context.Background(), Failure{
		Attempt: resumed,
		Step:    "extract",
		Kind:    FailurePanic,
	}); err != nil {
		t.Fatalf("Fail(resumed) error = %v", err)
	}
	if err := store.Delete(
		context.Background(),
		resumed.Definition(),
		resumed.Instance(),
	); err != nil {
		t.Fatalf("Delete(inactive) error = %v", err)
	}
	if _, err := store.Begin(
		context.Background(),
		testBeginRequest("second"),
	); err != nil {
		t.Fatalf("Begin after capacity release error = %v", err)
	}
}

func TestMemoryStoreAllowsOnlyOneConcurrentBegin(t *testing.T) {
	t.Parallel()

	const callers = 32
	store := mustMemoryStore(t, 1)
	start := make(chan struct{})
	var successes atomic.Int32
	var alreadyRunning atomic.Int32
	var unexpected atomic.Value
	var wait sync.WaitGroup
	wait.Add(callers)
	for range callers {
		go func() {
			defer wait.Done()
			<-start
			_, err := store.Begin(
				context.Background(),
				testBeginRequest("shared"),
			)
			switch {
			case err == nil:
				successes.Add(1)
			case errors.Is(err, ErrAlreadyRunning):
				alreadyRunning.Add(1)
			default:
				unexpected.Store(err)
			}
		}()
	}
	close(start)
	wait.Wait()

	if loaded := unexpected.Load(); loaded != nil {
		t.Fatalf("concurrent Begin() unexpected error = %v", loaded)
	}
	if successes.Load() != 1 || alreadyRunning.Load() != callers-1 {
		t.Fatalf(
			"concurrent Begin() successes=%d already-running=%d",
			successes.Load(),
			alreadyRunning.Load(),
		)
	}
}

func TestMemoryStoreValidatesInputsAndTransitions(t *testing.T) {
	t.Parallel()

	if _, err := NewMemoryStore(0); err == nil {
		t.Fatal("NewMemoryStore(0) unexpectedly succeeded")
	}
	if _, err := NewMemoryStore(maxMemoryStoreCapacity + 1); err == nil {
		t.Fatal("NewMemoryStore(too large) unexpectedly succeeded")
	}

	store := mustMemoryStore(t, 2)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.Begin(
		canceled,
		testBeginRequest("instance"),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("Begin(canceled) error = %v", err)
	}

	beginTests := []struct {
		name    string
		request BeginRequest
	}{
		{name: "invalid definition", request: BeginRequest{
			Definition: Definition{Module: "module"},
			Instance:   "instance",
			Steps:      []string{"step"},
		}},
		{name: "invalid instance", request: BeginRequest{
			Definition: testDefinition(),
			Instance:   "invalid instance",
			Steps:      []string{"step"},
		}},
		{name: "no steps", request: BeginRequest{
			Definition: testDefinition(),
			Instance:   "instance",
		}},
		{name: "invalid step", request: BeginRequest{
			Definition: testDefinition(),
			Instance:   "instance",
			Steps:      []string{"invalid step"},
		}},
		{name: "duplicate step", request: BeginRequest{
			Definition: testDefinition(),
			Instance:   "instance",
			Steps:      []string{"step", "step"},
		}},
	}
	for _, test := range beginTests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := store.Begin(
				context.Background(),
				test.request,
			); err == nil {
				t.Fatal("Begin() unexpectedly succeeded")
			}
		})
	}

	attempt, err := store.Begin(
		context.Background(),
		testBeginRequest("transition"),
	)
	if err != nil {
		t.Fatalf("Begin(transition) error = %v", err)
	}
	if err := store.Complete(
		context.Background(),
		attempt,
	); err == nil {
		t.Fatal("Complete() with pending steps unexpectedly succeeded")
	}
	if err := store.Fail(context.Background(), Failure{
		Attempt: attempt,
		Step:    "load",
		Kind:    "unknown",
	}); err == nil {
		t.Fatal("Fail() with invalid kind unexpectedly succeeded")
	}
	if err := store.Fail(context.Background(), Failure{
		Attempt: attempt,
		Step:    "load",
		Kind:    FailureError,
	}); err == nil {
		t.Fatal("Fail() outside active boundary unexpectedly succeeded")
	}
	if err := store.Fail(context.Background(), Failure{
		Attempt: attempt,
		Step:    "extract",
		Kind:    FailureError,
	}); err != nil {
		t.Fatalf("Fail() error = %v", err)
	}
	if err := store.Fail(context.Background(), Failure{
		Attempt: attempt,
		Step:    "extract",
		Kind:    FailureError,
	}); !errors.Is(err, ErrStaleAttempt) {
		t.Fatalf("stale Fail() error = %v, want ErrStaleAttempt", err)
	}
}

func TestMemoryStoreRejectsAttemptOverflow(t *testing.T) {
	t.Parallel()

	store := mustMemoryStore(t, 1)
	request := testBeginRequest("overflow")
	key := executionKey{
		definition: request.Definition,
		instance:   request.Instance,
	}
	store.executions[key] = &memoryExecution{
		steps:   slices.Clone(request.Steps),
		attempt: math.MaxUint64,
	}
	if _, err := store.Begin(context.Background(), request); err == nil {
		t.Fatal("Begin() at attempt overflow unexpectedly succeeded")
	}
}

func mustMemoryStore(t *testing.T, capacity int) *MemoryStore {
	t.Helper()
	store, err := NewMemoryStore(capacity)
	if err != nil {
		t.Fatalf("NewMemoryStore() error = %v", err)
	}
	return store
}

func testDefinition() Definition {
	return Definition{
		ID:     "orders.import",
		Module: "example.com/shop/orders",
	}
}

func testBeginRequest(instance string) BeginRequest {
	return BeginRequest{
		Definition: testDefinition(),
		Instance:   instance,
		Steps:      []string{"extract", "load"},
	}
}
