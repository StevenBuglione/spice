package spicetest

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/StevenBuglione/spice/lifecycle"
)

var (
	errTestFactory = errors.New("factory failed")
	errTestListen  = errors.New("listen failed")
	errTestStop    = errors.New("stop failed")
)

func TestHTTPSliceExecutesBoundedJSONAndProblemRequests(t *testing.T) {
	t.Parallel()

	application := &testHTTPApplication{
		state: lifecycle.StateConstructed,
		handler: http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			switch request.URL.Path {
			case "/echo":
				if request.Method != http.MethodPost ||
					request.Header.Get("X-Test") != "present" ||
					request.Header.Get("Accept") != "application/json" ||
					request.Header.Get("Content-Type") != "application/json" {
					http.Error(response, "unexpected request", http.StatusBadRequest)
					return
				}
				response.Header().Set("Content-Type", "application/json")
				if _, writeErr := response.Write(
					[]byte(`{"accepted":true}`),
				); writeErr != nil {
					t.Errorf("write echo response: %v", writeErr)
				}
			case "/problem":
				response.Header().Set("Content-Type", "application/problem+json")
				response.WriteHeader(http.StatusConflict)
				if _, writeErr := response.Write([]byte(
					`{"type":"urn:problem:conflict","title":"Conflict","status":409,"detail":"already exists"}`,
				)); writeErr != nil {
					t.Errorf("write problem response: %v", writeErr)
				}
			case "/large":
				if _, writeErr := response.Write(
					[]byte(strings.Repeat("x", 257)),
				); writeErr != nil {
					t.Errorf("write large response: %v", writeErr)
				}
			default:
				http.NotFound(response, request)
			}
		}),
	}
	slice, err := NewHTTP(
		context.Background(),
		func(context.Context) (HTTPApplication, error) {
			return application, nil
		},
		HTTPOptions{
			MaxResponseBodyBytes: 256,
			ClientTimeout:        time.Second,
			ShutdownTimeout:      time.Second,
		},
	)
	if err != nil {
		t.Fatalf("NewHTTP() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := slice.Close(); closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
	})
	if !strings.HasPrefix(slice.URL(), "http://127.0.0.1:") ||
		slice.Application() != application {
		t.Fatalf("slice URL=%q application=%v", slice.URL(), slice.Application())
	}

	headers := http.Header{"X-Test": []string{"present"}}
	response, err := slice.Do(context.Background(), HTTPRequest{
		Method: http.MethodPost,
		Path:   "/echo?source=test",
		Header: headers,
		JSON:   map[string]string{"value": "input"},
	})
	if err != nil {
		t.Fatalf("Do(echo) error = %v", err)
	}
	headers.Set("X-Test", "mutated")
	var decoded struct {
		Accepted bool `json:"accepted"`
	}
	if decodeErr := response.DecodeJSON(&decoded); decodeErr != nil {
		t.Fatalf("DecodeJSON() error = %v", decodeErr)
	}
	if response.StatusCode != http.StatusOK || !decoded.Accepted {
		t.Fatalf("echo response = %#v decoded=%#v", response, decoded)
	}

	response, err = slice.Do(context.Background(), HTTPRequest{
		Method: http.MethodGet,
		Path:   "/problem",
	})
	if err != nil {
		t.Fatalf("Do(problem) error = %v", err)
	}
	problem, err := response.Problem()
	if err != nil {
		t.Fatalf("Problem() error = %v", err)
	}
	if problem.Status != http.StatusConflict ||
		problem.Detail != "already exists" {
		t.Fatalf("problem = %#v", problem)
	}
	if _, err := slice.Do(context.Background(), HTTPRequest{
		Method: http.MethodGet,
		Path:   "/large",
	}); err == nil {
		t.Fatal("Do(large) unexpectedly succeeded")
	}
}

func TestHTTPSliceCloseIsConcurrentAndIdempotent(t *testing.T) {
	t.Parallel()

	application := &testHTTPApplication{
		state:   lifecycle.StateReady,
		handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	}
	slice, err := NewHTTP(
		context.Background(),
		func(context.Context) (HTTPApplication, error) {
			return application, nil
		},
		HTTPOptions{},
	)
	if err != nil {
		t.Fatalf("NewHTTP() error = %v", err)
	}
	if err := slice.CloseContext(nilTestContext()); err == nil {
		t.Fatal("CloseContext(nil) unexpectedly succeeded")
	}

	const callers = 16
	start := make(chan struct{})
	results := make(chan error, callers)
	for range callers {
		go func() {
			<-start
			results <- slice.Close()
		}()
	}
	close(start)
	for range callers {
		if err := <-results; err != nil {
			t.Fatalf("concurrent Close() error = %v", err)
		}
	}
	if application.stopCalls.Load() != 1 {
		t.Fatalf("Stop() calls = %d, want 1", application.stopCalls.Load())
	}
	if !application.activeStop.Load() {
		t.Fatal("Stop() did not receive an active shutdown context")
	}
	if _, err := slice.Do(context.Background(), HTTPRequest{
		Method: http.MethodGet,
		Path:   "/",
	}); err == nil {
		t.Fatal("Do() after Close unexpectedly succeeded")
	}
	if (*HTTP)(nil).URL() != "" ||
		(*HTTP)(nil).Application() != nil ||
		(*HTTP)(nil).Close() != nil ||
		(*HTTP)(nil).CloseContext(context.Background()) != nil {
		t.Fatal("nil slice accessors returned nonzero values")
	}
}

func TestHTTPSliceCleansUpConstructionAndListenFailures(t *testing.T) {
	t.Parallel()

	factoryFailure := &testHTTPApplication{
		state:   lifecycle.StateConstructed,
		handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	}
	if _, err := NewHTTP(
		context.Background(),
		func(context.Context) (HTTPApplication, error) {
			return factoryFailure, errTestFactory
		},
		HTTPOptions{},
	); !errors.Is(err, errTestFactory) {
		t.Fatalf("factory failure error = %v", err)
	}
	if factoryFailure.stopCalls.Load() != 1 {
		t.Fatalf("factory failure Stop() calls = %d", factoryFailure.stopCalls.Load())
	}

	listenFailure := &testHTTPApplication{
		state:   lifecycle.StateConstructed,
		handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	}
	if _, err := NewHTTP(
		context.Background(),
		func(context.Context) (HTTPApplication, error) {
			return listenFailure, nil
		},
		HTTPOptions{Listen: func(
			context.Context,
			string,
			string,
		) (net.Listener, error) {
			return nil, errTestListen
		}},
	); !errors.Is(err, errTestListen) {
		t.Fatalf("listen failure error = %v", err)
	}
	if listenFailure.stopCalls.Load() != 1 {
		t.Fatalf("listen failure Stop() calls = %d", listenFailure.stopCalls.Load())
	}

	nilListener := &testHTTPApplication{
		state:   lifecycle.StateConstructed,
		handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	}
	if _, err := NewHTTP(
		context.Background(),
		func(context.Context) (HTTPApplication, error) {
			return nilListener, nil
		},
		HTTPOptions{Listen: func(
			context.Context,
			string,
			string,
		) (net.Listener, error) {
			return nil, nil //nolint:nilnil // Deliberately verifies a broken listener seam.
		}},
	); err == nil {
		t.Fatal("nil listener unexpectedly succeeded")
	}
	if nilListener.stopCalls.Load() != 1 {
		t.Fatalf("nil listener Stop() calls = %d", nilListener.stopCalls.Load())
	}
}

func TestHTTPResponseDecodingFailsClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response HTTPResponse
		problem  bool
	}{
		{name: "empty JSON", response: HTTPResponse{}},
		{name: "multiple JSON", response: HTTPResponse{Body: []byte(`{} {}`)}},
		{
			name: "problem content type",
			response: HTTPResponse{
				StatusCode: http.StatusBadRequest,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       []byte(`{"title":"Bad Request","status":400}`),
			},
			problem: true,
		},
		{
			name: "problem status mismatch",
			response: HTTPResponse{
				StatusCode: http.StatusConflict,
				Header: http.Header{
					"Content-Type": []string{"application/problem+json"},
				},
				Body: []byte(`{"title":"Bad Request","status":400}`),
			},
			problem: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if test.problem {
				if _, err := test.response.Problem(); err == nil {
					t.Fatal("Problem() unexpectedly succeeded")
				}
				return
			}
			var target map[string]any
			if err := test.response.DecodeJSON(&target); err == nil {
				t.Fatal("DecodeJSON() unexpectedly succeeded")
			}
		})
	}
	if err := (HTTPResponse{Body: []byte(`{}`)}).DecodeJSON(nil); err == nil {
		t.Fatal("DecodeJSON(nil) unexpectedly succeeded")
	}
}

func TestHTTPSliceJoinsStopFailure(t *testing.T) {
	t.Parallel()

	application := &testHTTPApplication{
		state:   lifecycle.StateConstructed,
		handler: http.NotFoundHandler(),
		stopErr: errTestStop,
	}
	slice, err := NewHTTP(
		context.Background(),
		func(context.Context) (HTTPApplication, error) {
			return application, nil
		},
		HTTPOptions{},
	)
	if err != nil {
		t.Fatalf("NewHTTP() error = %v", err)
	}
	if err := slice.Close(); !errors.Is(err, errTestStop) {
		t.Fatalf("Close() error = %v, want stop failure", err)
	}
	if err := slice.Close(); !errors.Is(err, errTestStop) {
		t.Fatalf("second Close() error = %v, want stop failure", err)
	}
}

type testHTTPApplication struct {
	state      lifecycle.State
	handler    http.Handler
	stopErr    error
	stopCalls  atomic.Int32
	activeStop atomic.Bool
}

func (application *testHTTPApplication) Handler() http.Handler {
	if application == nil {
		return nil
	}
	return application.handler
}

func (application *testHTTPApplication) Stop(ctx context.Context) error {
	if application == nil {
		return errors.New("test application is nil")
	}
	application.stopCalls.Add(1)
	application.activeStop.Store(ctx != nil && context.Cause(ctx) == nil)
	return application.stopErr
}

func (application *testHTTPApplication) State() lifecycle.State {
	if application == nil {
		return lifecycle.StateInvalid
	}
	return application.state
}

func nilTestContext() context.Context {
	return nil
}
