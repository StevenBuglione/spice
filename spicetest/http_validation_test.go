package spicetest

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/StevenBuglione/spice/lifecycle"
)

func TestHTTPSliceRejectsInvalidConstructionAndRequests(t *testing.T) {
	t.Parallel()

	factory := func(context.Context) (HTTPApplication, error) {
		return &testHTTPApplication{
			state:   lifecycle.StateConstructed,
			handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		}, nil
	}
	if _, err := NewHTTP(nilTestContext(), factory, HTTPOptions{}); err == nil {
		t.Fatal("NewHTTP(nil context) unexpectedly succeeded")
	}
	if _, err := NewHTTP(
		context.Background(),
		nil,
		HTTPOptions{},
	); err == nil {
		t.Fatal("NewHTTP(nil factory) unexpectedly succeeded")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewHTTP(canceled, factory, HTTPOptions{}); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("NewHTTP(canceled) error = %v", err)
	}
	for index, options := range []HTTPOptions{
		{ClientTimeout: -1},
		{ClientTimeout: maxTestTimeout + 1},
		{ShutdownTimeout: -1},
		{ShutdownTimeout: maxTestTimeout + 1},
		{MaxRequestBodyBytes: -1},
		{MaxRequestBodyBytes: maxTestBodyBytes + 1},
		{MaxResponseBodyBytes: -1},
		{MaxResponseBodyBytes: maxTestBodyBytes + 1},
	} {
		if _, err := NewHTTP(
			context.Background(),
			factory,
			options,
		); err == nil {
			t.Fatalf("NewHTTP(invalid options %d) unexpectedly succeeded", index)
		}
	}
	for _, application := range []*testHTTPApplication{
		{state: lifecycle.StateStopped, handler: http.NotFoundHandler()},
		{state: lifecycle.StateConstructed},
	} {
		if _, err := NewHTTP(
			context.Background(),
			func(context.Context) (HTTPApplication, error) {
				return application, nil
			},
			HTTPOptions{},
		); err == nil {
			t.Fatalf("NewHTTP(invalid application %#v) unexpectedly succeeded", application)
		}
		if application.stopCalls.Load() != 1 {
			t.Fatalf("invalid application Stop() calls = %d", application.stopCalls.Load())
		}
	}

	slice, err := NewHTTP(
		context.Background(),
		factory,
		HTTPOptions{MaxRequestBodyBytes: 4},
	)
	if err != nil {
		t.Fatalf("NewHTTP() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := slice.Close(); closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
	})
	requests := []HTTPRequest{
		{},
		{Method: http.MethodGet, Path: "relative"},
		{Method: http.MethodGet, Path: "https://example.test/"},
		{
			Method: http.MethodPost,
			Path:   "/",
			JSON:   struct{}{},
			Body:   []byte("{}"),
		},
		{Method: http.MethodPost, Path: "/", Body: []byte("12345")},
	}
	for index, request := range requests {
		if _, err := slice.Do(
			context.Background(),
			request,
		); err == nil {
			t.Fatalf("Do(invalid request %d) unexpectedly succeeded", index)
		}
	}
	if _, err := slice.Do(nilTestContext(), HTTPRequest{}); err == nil {
		t.Fatal("Do(nil context) unexpectedly succeeded")
	}
	if _, err := (*HTTP)(nil).Do(
		context.Background(),
		HTTPRequest{},
	); err == nil {
		t.Fatal("nil Do() unexpectedly succeeded")
	}
	if err := slice.CloseContext(nilTestContext()); err == nil {
		t.Fatal("CloseContext(nil) unexpectedly succeeded")
	}
}
