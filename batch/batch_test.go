package batch

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

var errTestStep = errors.New("test step failed")

func TestNewJobValidatesAndFreezesOrderedSteps(t *testing.T) {
	t.Parallel()

	specs := []StepSpec{
		{ID: "extract", Run: func(context.Context) error { return nil }},
		{ID: "load", Run: func(context.Context) error { return nil }},
	}
	job, err := NewJob(
		Definition{ID: "orders.import", Module: "example.com/shop/orders"},
		specs,
	)
	if err != nil {
		t.Fatalf("NewJob() error = %v", err)
	}
	specs[0].ID = "changed"
	steps := job.Steps()
	steps[0] = Step{}
	if job.Definition() != (Definition{
		ID:     "orders.import",
		Module: "example.com/shop/orders",
	}) || !slices.Equal(
		jobStepIDs(job.Steps()),
		[]string{"extract", "load"},
	) {
		t.Fatalf(
			"job definition=%#v steps=%#v",
			job.Definition(),
			jobStepIDs(job.Steps()),
		)
	}
	if (*Job)(nil).Definition() != (Definition{}) ||
		len((*Job)(nil).Steps()) != 0 {
		t.Fatal("nil job accessors returned nonzero metadata")
	}

	tests := []struct {
		name       string
		definition Definition
		steps      []StepSpec
	}{
		{
			name:       "job ID",
			definition: Definition{Module: "example.com/module"},
			steps:      specs[:1],
		},
		{
			name:       "module",
			definition: Definition{ID: "job"},
			steps:      specs[:1],
		},
		{
			name: "no steps",
			definition: Definition{
				ID:     "job",
				Module: "example.com/module",
			},
		},
		{
			name: "step ID",
			definition: Definition{
				ID:     "job",
				Module: "example.com/module",
			},
			steps: []StepSpec{{
				ID:  "invalid step",
				Run: func(context.Context) error { return nil },
			}},
		},
		{
			name: "duplicate",
			definition: Definition{
				ID:     "job",
				Module: "example.com/module",
			},
			steps: []StepSpec{
				{ID: "step", Run: func(context.Context) error { return nil }},
				{ID: "step", Run: func(context.Context) error { return nil }},
			},
		},
		{
			name: "nil callback",
			definition: Definition{
				ID:     "job",
				Module: "example.com/module",
			},
			steps: []StepSpec{{ID: "step"}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewJob(test.definition, test.steps); err == nil {
				t.Fatal("NewJob() unexpectedly succeeded")
			}
		})
	}
}

func TestNewAttemptValidatesAndFreezesRestartMetadata(t *testing.T) {
	t.Parallel()

	completed := []string{"extract"}
	attempt, err := NewAttempt(AttemptSpec{
		Definition: Definition{
			ID:     "orders.import",
			Module: "example.com/shop/orders",
		},
		Instance:       "2026-07-26",
		Number:         2,
		CompletedSteps: completed,
	})
	if err != nil {
		t.Fatalf("NewAttempt() error = %v", err)
	}
	completed[0] = "changed"
	returned := attempt.CompletedSteps()
	returned[0] = "changed"
	if attempt.Definition().ID != "orders.import" ||
		attempt.Instance() != "2026-07-26" ||
		attempt.Number() != 2 ||
		attempt.Complete() ||
		!slices.Equal(attempt.CompletedSteps(), []string{"extract"}) {
		t.Fatalf("attempt = %#v", attempt)
	}

	valid := AttemptSpec{
		Definition: Definition{ID: "job", Module: "example.com/module"},
		Instance:   "instance",
		Number:     1,
	}
	tests := []struct {
		name   string
		mutate func(*AttemptSpec)
	}{
		{name: "definition", mutate: func(spec *AttemptSpec) {
			spec.Definition.ID = ""
		}},
		{name: "instance", mutate: func(spec *AttemptSpec) {
			spec.Instance = "invalid instance"
		}},
		{name: "number", mutate: func(spec *AttemptSpec) {
			spec.Number = 0
		}},
		{name: "completed invalid", mutate: func(spec *AttemptSpec) {
			spec.CompletedSteps = []string{"invalid step"}
		}},
		{name: "completed duplicate", mutate: func(spec *AttemptSpec) {
			spec.CompletedSteps = []string{"step", "step"}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			spec := valid
			test.mutate(&spec)
			if _, err := NewAttempt(spec); err == nil {
				t.Fatal("NewAttempt() unexpectedly succeeded")
			}
		})
	}
}

func TestRunnerExecutesResumesAndRecognizesCompletedInstances(t *testing.T) {
	t.Parallel()

	definition := Definition{
		ID:     "orders.import",
		Module: "example.com/shop/orders",
	}
	tests := []struct {
		name            string
		completed       []string
		complete        bool
		wantRun         []string
		wantCheckpoints []string
		wantResult      Result
	}{
		{
			name:            "new",
			wantRun:         []string{"extract", "load"},
			wantCheckpoints: []string{"extract", "load"},
			wantResult: Result{
				Attempt:        3,
				StepsCompleted: 2,
			},
		},
		{
			name:            "resume",
			completed:       []string{"extract"},
			wantRun:         []string{"load"},
			wantCheckpoints: []string{"load"},
			wantResult: Result{
				Attempt:        3,
				StepsSkipped:   1,
				StepsCompleted: 1,
				Resumed:        true,
			},
		},
		{
			name:      "already complete",
			completed: []string{"extract", "load"},
			complete:  true,
			wantResult: Result{
				Attempt:         3,
				StepsSkipped:    2,
				Resumed:         true,
				AlreadyComplete: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var ran []string
			job := mustJob(t, definition, func(step string) func(context.Context) error {
				return func(context.Context) error {
					ran = append(ran, step)
					return nil
				}
			})
			store := &recordingStore{attempt: mustAttempt(t, AttemptSpec{
				Definition:     definition,
				Instance:       "instance",
				Number:         3,
				CompletedSteps: test.completed,
				Complete:       test.complete,
			})}
			var observations []Observation
			runner := mustRunner(
				t,
				store,
				func(_ context.Context, observation Observation) {
					observations = append(observations, observation)
				},
			)
			result, err := runner.Run(
				context.Background(),
				job,
				"instance",
			)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			result.Duration = 0
			if result != test.wantResult ||
				!slices.Equal(ran, test.wantRun) ||
				!slices.Equal(store.checkpoints, test.wantCheckpoints) {
				t.Fatalf(
					"result=%#v ran=%#v checkpoints=%#v",
					result,
					ran,
					store.checkpoints,
				)
			}
			if store.beginRequest.Instance != "instance" ||
				!slices.Equal(
					store.beginRequest.Steps,
					[]string{"extract", "load"},
				) {
				t.Fatalf("Begin() request = %#v", store.beginRequest)
			}
			if store.completeCalls != boolCount(!test.complete) {
				t.Fatalf("Complete() calls = %d", store.completeCalls)
			}
			if len(observations) == 0 ||
				observations[len(observations)-1].Operation != OperationJob ||
				!observations[len(observations)-1].Completed {
				t.Fatalf("observations = %#v", observations)
			}
		})
	}
}

func TestRunnerRecordsStepFailuresAndContainsPanics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		run          func(context.Context) error
		wantKind     FailureKind
		wantError    error
		wantPanicked bool
	}{
		{
			name:      "error",
			run:       func(context.Context) error { return errTestStep },
			wantKind:  FailureError,
			wantError: errTestStep,
		},
		{
			name: "canceled",
			run: func(context.Context) error {
				return context.Canceled
			},
			wantKind:  FailureCanceled,
			wantError: context.Canceled,
		},
		{
			name: "panic",
			run: func(context.Context) error {
				panic("sensitive panic value")
			},
			wantKind:     FailurePanic,
			wantError:    ErrPanicked,
			wantPanicked: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			definition := Definition{ID: "job", Module: "example.com/module"}
			job, err := NewJob(definition, []StepSpec{{
				ID:  "step",
				Run: test.run,
			}})
			if err != nil {
				t.Fatalf("NewJob() error = %v", err)
			}
			store := &recordingStore{attempt: mustAttempt(t, AttemptSpec{
				Definition: definition,
				Instance:   "instance",
				Number:     1,
			})}
			var observation Observation
			runner := mustRunner(t, store, func(
				_ context.Context,
				value Observation,
			) {
				observation = value
			})
			result, err := runner.Run(
				context.Background(),
				job,
				"instance",
			)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("Run() error = %v, want %v", err, test.wantError)
			}
			if strings.Contains(err.Error(), "sensitive panic value") {
				t.Fatalf("Run() exposed panic value: %v", err)
			}
			if result.StepsCompleted != 0 ||
				len(store.failures) != 1 ||
				store.failures[0].Kind != test.wantKind ||
				store.failures[0].Step != "step" ||
				observation.Operation != OperationStep ||
				observation.Panicked != test.wantPanicked ||
				observation.Completed {
				t.Fatalf(
					"result=%#v failures=%#v observation=%#v",
					result,
					store.failures,
					observation,
				)
			}
			if !store.failContextUsable {
				t.Fatal("Fail() did not receive a fresh usable context")
			}
		})
	}
}

func TestRunnerRecordsPersistenceFailuresAndJoinsTransitionErrors(t *testing.T) {
	t.Parallel()

	checkpointFailure := errors.New("checkpoint failed")
	completionFailure := errors.New("completion failed")
	recordFailure := errors.New("record failure failed")
	tests := []struct {
		name      string
		configure func(*recordingStore)
		wantError error
		wantStep  string
		completed int
	}{
		{
			name: "checkpoint",
			configure: func(store *recordingStore) {
				store.checkpointErr = checkpointFailure
			},
			wantError: checkpointFailure,
			wantStep:  "extract",
		},
		{
			name: "complete",
			configure: func(store *recordingStore) {
				store.completeErr = completionFailure
			},
			wantError: completionFailure,
			wantStep:  "load",
			completed: 2,
		},
		{
			name: "joined failure",
			configure: func(store *recordingStore) {
				store.checkpointErr = checkpointFailure
				store.failErr = recordFailure
			},
			wantError: checkpointFailure,
			wantStep:  "extract",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			definition := Definition{ID: "job", Module: "example.com/module"}
			store := &recordingStore{attempt: mustAttempt(t, AttemptSpec{
				Definition: definition,
				Instance:   "instance",
				Number:     1,
			})}
			test.configure(store)
			result, err := mustRunner(t, store).Run(
				context.Background(),
				mustJob(t, definition, successfulStep),
				"instance",
			)
			if !errors.Is(err, test.wantError) ||
				(store.failErr != nil && !errors.Is(err, recordFailure)) ||
				result.StepsCompleted != test.completed ||
				len(store.failures) != 1 ||
				store.failures[0].Step != test.wantStep {
				t.Fatalf(
					"result=%#v error=%v failures=%#v",
					result,
					err,
					store.failures,
				)
			}
		})
	}
}

func TestRunnerRejectsInvalidInputsAndStoreAttempts(t *testing.T) {
	t.Parallel()

	definition := Definition{ID: "job", Module: "example.com/module"}
	job := mustJob(t, definition, successfulStep)
	store := &recordingStore{attempt: mustAttempt(t, AttemptSpec{
		Definition: definition,
		Instance:   "instance",
		Number:     1,
	})}
	runner := mustRunner(t, store)
	if _, err := runner.Run(nilTestContext(), job, "instance"); err == nil {
		t.Fatal("Run(nil context) unexpectedly succeeded")
	}
	if _, err := (*Runner)(nil).Run(
		context.Background(),
		job,
		"instance",
	); err == nil {
		t.Fatal("nil Runner.Run() unexpectedly succeeded")
	}
	if _, err := runner.Run(
		context.Background(),
		nil,
		"instance",
	); err == nil {
		t.Fatal("Run(nil job) unexpectedly succeeded")
	}
	if _, err := runner.Run(
		context.Background(),
		job,
		"invalid instance",
	); err == nil {
		t.Fatal("Run(invalid instance) unexpectedly succeeded")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runner.Run(canceled, job, "instance"); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("Run(canceled) error = %v", err)
	}

	attempts := []Attempt{
		mustAttempt(t, AttemptSpec{
			Definition: Definition{ID: "other", Module: definition.Module},
			Instance:   "instance",
			Number:     1,
		}),
		mustAttempt(t, AttemptSpec{
			Definition: definition,
			Instance:   "other",
			Number:     1,
		}),
		{definition: definition, instance: "instance"},
		mustAttempt(t, AttemptSpec{
			Definition:     definition,
			Instance:       "instance",
			Number:         1,
			CompletedSteps: []string{"load"},
		}),
		mustAttempt(t, AttemptSpec{
			Definition:     definition,
			Instance:       "instance",
			Number:         1,
			CompletedSteps: []string{"extract"},
			Complete:       true,
		}),
	}
	for index, attempt := range attempts {
		store := &recordingStore{attempt: attempt}
		if _, err := mustRunner(t, store).Run(
			context.Background(),
			job,
			"instance",
		); err == nil {
			t.Fatalf("malformed attempt %d unexpectedly succeeded", index)
		}
	}
}

func TestNewRunnerAndFailureContextValidation(t *testing.T) {
	t.Parallel()

	var nilStore *recordingStore
	if _, err := NewRunner(nilStore, freshContext); err == nil {
		t.Fatal("NewRunner(nil store) unexpectedly succeeded")
	}
	if _, err := NewRunner(&recordingStore{}, nil); err == nil {
		t.Fatal("NewRunner(nil factory) unexpectedly succeeded")
	}
	if _, err := NewRunner(
		&recordingStore{},
		freshContext,
		nil,
	); err == nil {
		t.Fatal("NewRunner(nil observer) unexpectedly succeeded")
	}

	definition := Definition{ID: "job", Module: "example.com/module"}
	factories := []ContextFactory{
		func() (context.Context, context.CancelFunc) {
			return nil, func() {}
		},
		func() (context.Context, context.CancelFunc) {
			return context.Background(), nil
		},
		func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, func() {}
		},
	}
	for index, factory := range factories {
		store := &recordingStore{attempt: mustAttempt(t, AttemptSpec{
			Definition: definition,
			Instance:   "instance",
			Number:     1,
		})}
		job, err := NewJob(definition, []StepSpec{{
			ID:  "step",
			Run: func(context.Context) error { return errTestStep },
		}})
		if err != nil {
			t.Fatalf("NewJob() error = %v", err)
		}
		runner, err := NewRunner(store, factory)
		if err != nil {
			t.Fatalf("NewRunner() error = %v", err)
		}
		if _, err := runner.Run(
			context.Background(),
			job,
			"instance",
		); !errors.Is(err, errTestStep) {
			t.Fatalf("factory %d error = %v", index, err)
		}
		if len(store.failures) != 0 {
			t.Fatalf("factory %d reached Fail()", index)
		}
	}
}

func mustJob(
	t *testing.T,
	definition Definition,
	step func(string) func(context.Context) error,
) *Job {
	t.Helper()
	job, err := NewJob(definition, []StepSpec{
		{ID: "extract", Run: step("extract")},
		{ID: "load", Run: step("load")},
	})
	if err != nil {
		t.Fatalf("NewJob() error = %v", err)
	}
	return job
}

func successfulStep(string) func(context.Context) error {
	return func(context.Context) error { return nil }
}

func mustAttempt(t *testing.T, spec AttemptSpec) Attempt {
	t.Helper()
	attempt, err := NewAttempt(spec)
	if err != nil {
		t.Fatalf("NewAttempt() error = %v", err)
	}
	return attempt
}

func mustRunner(
	t *testing.T,
	store Store,
	observers ...Observer,
) *Runner {
	t.Helper()
	runner, err := NewRunner(store, freshContext, observers...)
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	return runner
}

func freshContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Second)
}

func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nilTestContext() context.Context {
	return nil
}

type recordingStore struct {
	mu                sync.Mutex
	attempt           Attempt
	beginRequest      BeginRequest
	beginErr          error
	checkpoints       []string
	checkpointErr     error
	completeCalls     int
	completeErr       error
	failures          []Failure
	failContextUsable bool
	failErr           error
}

func (store *recordingStore) Begin(
	_ context.Context,
	request BeginRequest,
) (Attempt, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.beginRequest = request
	return store.attempt, store.beginErr
}

func (store *recordingStore) Checkpoint(
	_ context.Context,
	_ Attempt,
	step string,
) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.checkpoints = append(store.checkpoints, step)
	return store.checkpointErr
}

func (store *recordingStore) Complete(
	_ context.Context,
	_ Attempt,
) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.completeCalls++
	return store.completeErr
}

func (store *recordingStore) Fail(
	ctx context.Context,
	failure Failure,
) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.failContextUsable = ctx != nil && context.Cause(ctx) == nil
	store.failures = append(store.failures, failure)
	return store.failErr
}
