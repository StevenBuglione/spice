package retry

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"
	"testing"
	"time"
)

var (
	errTransient = errors.New("transient")
	errPermanent = errors.New("permanent")
	errWait      = errors.New("wait failed")
)

func TestDoRetriesWithCappedBackoffAndReturnsValue(t *testing.T) {
	t.Parallel()
	var attempts []Attempt
	var delays []time.Duration
	var observations []Observation
	policy := validPolicy()
	policy.MaxAttempts = 4
	policy.InitialBackoff = 10 * time.Millisecond
	policy.MaxBackoff = 25 * time.Millisecond
	policy.Wait = func(_ context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		return nil
	}
	policy.Observer = func(_ context.Context, observation Observation) {
		observations = append(observations, observation)
	}

	value, err := Do(context.Background(), policy, func(_ context.Context, attempt Attempt) (string, error) {
		attempts = append(attempts, attempt)
		if attempt.Number < attempt.Max {
			return "", errTransient
		}
		return "ready", nil
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if value != "ready" {
		t.Fatalf("Do() value = %q, want ready", value)
	}
	if want := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		25 * time.Millisecond,
	}; !slices.Equal(delays, want) {
		t.Fatalf("delays = %v, want %v", delays, want)
	}
	if len(attempts) != 4 || len(observations) != 4 {
		t.Fatalf("attempts = %v, observations = %#v", attempts, observations)
	}
	for index, observation := range observations {
		if observation.ID != policy.ID ||
			observation.Module != policy.Module ||
			observation.Attempt.Number != index+1 ||
			observation.Duration < 0 {
			t.Fatalf("observation %d = %#v", index, observation)
		}
	}
	if observations[3].Err != nil || observations[3].NextBackoff != 0 {
		t.Fatalf("final observation = %#v", observations[3])
	}
}

func TestDoStopsAtNonRetryableFailure(t *testing.T) {
	t.Parallel()
	policy := validPolicy()
	var calls int
	_, err := Do(context.Background(), policy, func(context.Context, Attempt) (int, error) {
		calls++
		return 0, errPermanent
	})
	if !errors.Is(err, errPermanent) {
		t.Fatalf("Do() error = %v, want errPermanent", err)
	}
	if calls != 1 {
		t.Fatalf("operation calls = %d, want 1", calls)
	}
	if unexpected, ok := errors.AsType[*ExhaustedError](err); ok {
		t.Fatalf("Do() error = %v, did not want ExhaustedError %#v", err, unexpected)
	}
}

func TestDoReturnsTypedExhaustion(t *testing.T) {
	t.Parallel()
	policy := validPolicy()
	policy.MaxAttempts = 3
	_, err := Do(context.Background(), policy, func(context.Context, Attempt) (int, error) {
		return 0, errTransient
	})
	exhausted, ok := errors.AsType[*ExhaustedError](err)
	if !ok ||
		!errors.Is(err, errTransient) ||
		exhausted.Attempts != policy.MaxAttempts {
		t.Fatalf("Do() error = %#v", err)
	}
	if (*ExhaustedError)(nil).Error() != "retry attempts exhausted" ||
		(*ExhaustedError)(nil).Unwrap() != nil {
		t.Fatal("nil ExhaustedError contract changed")
	}
	if got := exhausted.Error(); !strings.Contains(got, "after 3 attempts") {
		t.Fatalf("ExhaustedError.Error() = %q", got)
	}
}

func TestDoHonorsCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	var observations []Observation
	policy := validPolicy()
	policy.InitialBackoff = time.Second
	policy.MaxBackoff = time.Second
	policy.Observer = func(_ context.Context, observation Observation) {
		observations = append(observations, observation)
	}
	policy.Wait = func(ctx context.Context, _ time.Duration) error {
		cancel()
		return context.Cause(ctx)
	}

	_, err := Do(ctx, policy, func(context.Context, Attempt) (int, error) {
		return 0, errTransient
	})
	if !errors.Is(err, context.Canceled) || !errors.Is(err, errTransient) {
		t.Fatalf("Do() error = %v, want cancellation and attempt error", err)
	}
	if len(observations) != 1 || observations[0].NextBackoff != time.Second {
		t.Fatalf("observations = %#v", observations)
	}
}

func TestDoStopsWhenOperationCancelsContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	policy := validPolicy()
	var calls int
	_, err := Do(ctx, policy, func(context.Context, Attempt) (int, error) {
		calls++
		cancel()
		return 0, errTransient
	})
	if !errors.Is(err, context.Canceled) || !errors.Is(err, errTransient) {
		t.Fatalf("Do() error = %v, want cancellation and attempt error", err)
	}
	if calls != 1 {
		t.Fatalf("operation calls = %d, want 1", calls)
	}

	canceled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := Do(canceled, policy, func(context.Context, Attempt) (int, error) {
		t.Fatal("operation ran for an initially canceled context")
		return 0, nil
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Do(initially canceled) error = %v", err)
	}
}

func TestDoRejectsInvalidJitterAndPreservesAttemptError(t *testing.T) {
	t.Parallel()
	policy := validPolicy()
	policy.InitialBackoff = time.Millisecond
	policy.MaxBackoff = 2 * time.Millisecond
	policy.Jitter = func(Attempt, time.Duration) time.Duration {
		return 3 * time.Millisecond
	}
	_, err := Do(context.Background(), policy, func(context.Context, Attempt) (int, error) {
		return 0, errTransient
	})
	if !errors.Is(err, errTransient) ||
		err == nil ||
		!strings.Contains(err.Error(), "jitter returned backoff") {
		t.Fatalf("Do() error = %v", err)
	}
	if got := err.Error(); got == "" {
		t.Fatal("Do() returned an empty jitter error")
	}
}

func TestDoAppliesJitterAndPreservesWaitFailure(t *testing.T) {
	t.Parallel()
	policy := validPolicy()
	policy.InitialBackoff = 10 * time.Millisecond
	policy.MaxBackoff = 20 * time.Millisecond
	policy.Jitter = func(_ Attempt, base time.Duration) time.Duration {
		return base / 2
	}
	var waited time.Duration
	policy.Wait = func(_ context.Context, delay time.Duration) error {
		waited = delay
		return errWait
	}
	_, err := Do(context.Background(), policy, func(context.Context, Attempt) (int, error) {
		return 0, errTransient
	})
	if !errors.Is(err, errTransient) || !errors.Is(err, errWait) {
		t.Fatalf("Do() error = %v, want attempt and wait errors", err)
	}
	if waited != 5*time.Millisecond {
		t.Fatalf("waited = %v, want 5ms", waited)
	}
}

func TestDoObservesPanicAndRepanics(t *testing.T) {
	t.Parallel()
	panicValue := &struct{ message string }{message: "boom"}
	policy := validPolicy()
	var observed Observation
	policy.Observer = func(_ context.Context, observation Observation) {
		observed = observation
	}

	recovered := recoverDo(func() {
		_, err := Do(context.Background(), policy, func(context.Context, Attempt) (int, error) {
			panic(panicValue)
		})
		t.Fatalf("Do() error = %v, want panic", err)
	})
	if recovered != panicValue {
		t.Fatalf("recovered = %#v, want original panic value", recovered)
	}
	if !observed.Panicked || !errors.Is(observed.Err, ErrPanicked) {
		t.Fatalf("observation = %#v", observed)
	}
}

func TestRunExecutesErrorOnlyOperation(t *testing.T) {
	t.Parallel()
	policy := validPolicy()
	policy.MaxAttempts = 1
	policy.Retryable = nil
	var attempt Attempt
	err := Run(context.Background(), policy, func(_ context.Context, current Attempt) error {
		attempt = current
		return nil
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if attempt != (Attempt{Number: 1, Max: 1}) {
		t.Fatalf("attempt = %#v", attempt)
	}
	if err := Run(context.Background(), policy, nil); err == nil {
		t.Fatal("Run(nil operation) error = nil")
	}
}

func TestPolicyValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		policy Policy
	}{
		{"missing ID", Policy{Module: "example.com/shop", MaxAttempts: 1}},
		{"missing module", Policy{ID: "operation", MaxAttempts: 1}},
		{"zero attempts", Policy{ID: "operation", Module: "example.com/shop"}},
		{
			"missing classifier",
			Policy{ID: "operation", Module: "example.com/shop", MaxAttempts: 2},
		},
		{
			"negative initial",
			Policy{
				ID:             "operation",
				Module:         "example.com/shop",
				MaxAttempts:    1,
				InitialBackoff: -1,
			},
		},
		{
			"max below initial",
			Policy{
				ID:             "operation",
				Module:         "example.com/shop",
				MaxAttempts:    1,
				InitialBackoff: time.Second,
			},
		},
		{
			"multiplier one",
			Policy{
				ID:          "operation",
				Module:      "example.com/shop",
				MaxAttempts: 1,
				Multiplier:  1,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Do(
				context.Background(),
				test.policy,
				func(context.Context, Attempt) (int, error) { return 0, nil },
			); err == nil {
				t.Fatal("Do() error = nil")
			}
		})
	}

	policy := validPolicy()
	if _, err := Do[int](nilTestContext(), policy, func(context.Context, Attempt) (int, error) {
		return 0, nil
	}); err == nil {
		t.Fatal("Do(nil context) error = nil")
	}
	if _, err := Do[int](context.Background(), policy, nil); err == nil {
		t.Fatal("Do(nil operation) error = nil")
	}
}

func TestNextBackoffAvoidsOverflow(t *testing.T) {
	t.Parallel()
	maximum := time.Duration(math.MaxInt64)
	if got := nextBackoff(maximum-1, maximum, 2); got != maximum {
		t.Fatalf("nextBackoff() = %v, want %v", got, maximum)
	}
	if got := nextBackoff(maximum, maximum, 2); got != maximum {
		t.Fatalf("nextBackoff(at maximum) = %v, want %v", got, maximum)
	}
	if got := nextBackoff(0, maximum, 2); got != 0 {
		t.Fatalf("nextBackoff(zero) = %v, want 0", got)
	}
	if got := nextBackoff(2, 10, 2); got != 4 {
		t.Fatalf("nextBackoff(2) = %v, want 4", got)
	}
}

func TestDefaultWaitReturnsCancellationAtZeroDelay(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := wait(ctx, nil, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("wait() error = %v, want context.Canceled", err)
	}
	if err := wait(context.Background(), func(context.Context, time.Duration) error {
		return errWait
	}, time.Second); !errors.Is(err, errWait) {
		t.Fatalf("wait(custom) error = %v, want errWait", err)
	}
	if err := wait(context.Background(), nil, time.Millisecond); err != nil {
		t.Fatalf("wait(timer) error = %v", err)
	}
}

func validPolicy() Policy {
	return Policy{
		ID:          "payments.Charge",
		Module:      "example.com/shop/payments",
		MaxAttempts: 2,
		Retryable: func(err error) bool {
			return errors.Is(err, errTransient)
		},
	}
}

func nilTestContext() context.Context {
	return nil
}

func recoverDo(run func()) (recovered any) {
	defer func() {
		recovered = recover()
	}()
	run()
	return nil
}
