package spicetest

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spice-framework/spice/lifecycle"
)

var (
	errContextFactory = errors.New("context factory failed")
	errContextStart   = errors.New("context start failed")
	errContextStop    = errors.New("context stop failed")
)

func TestContextPreservesConcreteApplicationAndOwnsLifecycle(t *testing.T) {
	t.Parallel()

	application := newTestContextApplication("typed-component")
	testContext, err := NewContext(
		context.Background(),
		func(context.Context) (*testContextApplication, error) {
			return application, nil
		},
		ContextOptions{},
	)
	if err != nil {
		t.Fatalf("NewContext() error = %v", err)
	}
	if testContext.Application().component != "typed-component" {
		t.Fatalf(
			"Application().component = %q",
			testContext.Application().component,
		)
	}
	if testContext.State() != lifecycle.StateReady ||
		application.startCalls.Load() != 1 {
		t.Fatalf(
			"state=%s start calls=%d",
			testContext.State(),
			application.startCalls.Load(),
		)
	}

	const callers = 16
	start := make(chan struct{})
	results := make(chan error, callers)
	for range callers {
		go func() {
			<-start
			results <- testContext.Close()
		}()
	}
	close(start)
	for range callers {
		if err := <-results; err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}
	if application.stopCalls.Load() != 1 ||
		testContext.State() != lifecycle.StateStopped {
		t.Fatalf(
			"stop calls=%d state=%s",
			application.stopCalls.Load(),
			testContext.State(),
		)
	}
}

func TestContextSupportsConstructedSlices(t *testing.T) {
	t.Parallel()

	application := newTestContextApplication("slice")
	testContext, err := NewContext(
		context.Background(),
		func(context.Context) (*testContextApplication, error) {
			return application, nil
		},
		ContextOptions{SkipStart: true},
	)
	if err != nil {
		t.Fatalf("NewContext() error = %v", err)
	}
	if testContext.State() != lifecycle.StateConstructed ||
		application.startCalls.Load() != 0 {
		t.Fatalf(
			"state=%s start calls=%d",
			testContext.State(),
			application.startCalls.Load(),
		)
	}
	if err := testContext.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestContextCleansUpReturnedFactoryFailures(t *testing.T) {
	t.Parallel()

	application := newTestContextApplication("failure")
	_, err := NewContext(
		context.Background(),
		func(context.Context) (*testContextApplication, error) {
			return application, errContextFactory
		},
		ContextOptions{},
	)
	if !errors.Is(err, errContextFactory) {
		t.Fatalf("NewContext() error = %v", err)
	}
	if application.stopCalls.Load() != 1 {
		t.Fatalf("Stop() calls = %d, want 1", application.stopCalls.Load())
	}
}

func TestContextReportsStartupAndShutdownFailures(t *testing.T) {
	t.Parallel()

	startFailure := newTestContextApplication("start-failure")
	startFailure.startErr = errContextStart
	if _, err := NewContext(
		context.Background(),
		func(context.Context) (*testContextApplication, error) {
			return startFailure, nil
		},
		ContextOptions{},
	); !errors.Is(err, errContextStart) {
		t.Fatalf("startup failure error = %v", err)
	}

	stopFailure := newTestContextApplication("stop-failure")
	stopFailure.stopErr = errContextStop
	testContext, err := NewContext(
		context.Background(),
		func(context.Context) (*testContextApplication, error) {
			return stopFailure, nil
		},
		ContextOptions{},
	)
	if err != nil {
		t.Fatalf("NewContext(stop failure) error = %v", err)
	}
	if err := testContext.Close(); !errors.Is(err, errContextStop) {
		t.Fatalf("Close() error = %v", err)
	}
	if err := testContext.Close(); !errors.Is(err, errContextStop) {
		t.Fatalf("repeated Close() error = %v", err)
	}
}

func TestContextHonorsCancellationTimeoutsAndValidation(t *testing.T) {
	t.Parallel()

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewContext(
		canceled,
		func(context.Context) (*testContextApplication, error) {
			t.Fatal("factory called for canceled context")
			return nil, errContextFactory
		},
		ContextOptions{},
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled NewContext() error = %v", err)
	}

	timeoutApplication := newTestContextApplication("timeout")
	timeoutApplication.waitForStartCancellation = true
	if _, err := NewContext(
		context.Background(),
		func(context.Context) (*testContextApplication, error) {
			return timeoutApplication, nil
		},
		ContextOptions{StartupTimeout: time.Millisecond},
	); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("startup timeout error = %v", err)
	}

	for _, options := range []ContextOptions{
		{StartupTimeout: -1},
		{StartupTimeout: maxTestTimeout + 1},
		{ShutdownTimeout: -1},
		{ShutdownTimeout: maxTestTimeout + 1},
	} {
		if _, err := NewContext(
			context.Background(),
			func(context.Context) (*testContextApplication, error) {
				return newTestContextApplication("invalid"), nil
			},
			options,
		); err == nil {
			t.Fatalf("NewContext(%+v) unexpectedly succeeded", options)
		}
	}
	if _, err := NewContext(
		nilTestContext(),
		func(context.Context) (*testContextApplication, error) {
			return newTestContextApplication("nil"), nil
		},
		ContextOptions{},
	); err == nil {
		t.Fatal("NewContext(nil context) unexpectedly succeeded")
	}
	var nilFactory Factory[*testContextApplication]
	if _, err := NewContext(
		context.Background(),
		nilFactory,
		ContextOptions{},
	); err == nil {
		t.Fatal("NewContext(nil factory) unexpectedly succeeded")
	}
	if (*Context[*testContextApplication])(nil).State() != lifecycle.StateInvalid ||
		(*Context[*testContextApplication])(nil).Application() != nil ||
		(*Context[*testContextApplication])(nil).Close() != nil {
		t.Fatal("nil Context accessors returned nonzero values")
	}
}

type testContextApplication struct {
	mu                       sync.Mutex
	component                string
	state                    lifecycle.State
	startCalls               atomic.Int32
	stopCalls                atomic.Int32
	startErr                 error
	stopErr                  error
	waitForStartCancellation bool
}

func newTestContextApplication(component string) *testContextApplication {
	return &testContextApplication{
		component: component,
		state:     lifecycle.StateConstructed,
	}
}

func (application *testContextApplication) Start(ctx context.Context) error {
	application.startCalls.Add(1)
	if application.waitForStartCancellation {
		<-ctx.Done()
		application.setState(lifecycle.StateFailed)
		return context.Cause(ctx)
	}
	if application.startErr != nil {
		application.setState(lifecycle.StateFailed)
		return application.startErr
	}
	application.setState(lifecycle.StateReady)
	return nil
}

func (application *testContextApplication) Stop(context.Context) error {
	application.stopCalls.Add(1)
	application.setState(lifecycle.StateStopped)
	return application.stopErr
}

func (application *testContextApplication) State() lifecycle.State {
	if application == nil {
		return lifecycle.StateInvalid
	}
	application.mu.Lock()
	defer application.mu.Unlock()
	return application.state
}

func (application *testContextApplication) setState(state lifecycle.State) {
	application.mu.Lock()
	application.state = state
	application.mu.Unlock()
}
