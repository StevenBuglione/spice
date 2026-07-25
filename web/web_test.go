package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

func FuzzDecodeJSON(f *testing.F) {
	for _, seed := range []struct {
		contentType string
		body        string
	}{
		{contentType: "application/json", body: `{"name":"Spice","count":3}`},
		{contentType: "application/problem+json", body: `{}`},
		{contentType: "text/plain", body: `{}`},
		{contentType: "application/json", body: `{} {}`},
	} {
		f.Add(seed.contentType, []byte(seed.body))
	}
	f.Fuzz(func(t *testing.T, contentType string, body []byte) {
		if len(body) > 64<<10 || len(contentType) > 1<<10 {
			t.Skip()
		}
		request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		request.Header.Set("Content-Type", contentType)
		var destination struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}
		if err := DecodeJSON(request, &destination, 64<<10); err != nil {
			return
		}
	})
}

func TestProblemMappingAndWriters(t *testing.T) {
	cause := errors.New("database password secret")
	explicit := NewError(Problem{
		Type:     "https://example.com/problems/conflict",
		Title:    "Conflict",
		Status:   http.StatusConflict,
		Detail:   "resource changed",
		Instance: "/orders/42",
	}, cause)
	if !errors.Is(explicit, cause) || explicit.Error() != "Conflict: resource changed" {
		t.Fatalf("HTTPError = %v, unwrap=%v", explicit, errors.Unwrap(explicit))
	}
	problem := DefaultErrorMapper(context.Background(), explicit)
	if problem.Status != http.StatusConflict || problem.Detail != "resource changed" {
		t.Fatalf("mapped problem = %#v", problem)
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/orders/42", nil)
	if err := WriteError(response, request, explicit, nil); err != nil {
		t.Fatalf("WriteError() error = %v", err)
	}
	if response.Code != http.StatusConflict ||
		response.Header().Get("Content-Type") != problemMediaType ||
		response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("problem response = %d, %#v", response.Code, response.Header())
	}
	var decoded Problem
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded != problem || strings.Contains(response.Body.String(), cause.Error()) {
		t.Fatalf("decoded problem = %#v, body=%s", decoded, response.Body.String())
	}

	internal := DefaultErrorMapper(context.Background(), cause)
	if internal.Status != http.StatusInternalServerError || internal.Detail != "" {
		t.Fatalf("internal mapping = %#v", internal)
	}
	invalid := NewError(Problem{Status: http.StatusOK, Detail: "leak"}, nil)
	if got := DefaultErrorMapper(context.Background(), invalid); got != internalProblem() {
		t.Fatalf("invalid mapping = %#v", got)
	}
}

func TestRegisterConvertsServeMuxPanicsToErrors(t *testing.T) {
	mux := http.NewServeMux()
	var order []string
	named := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				order = append(order, name+":before")
				next.ServeHTTP(writer, request)
				order = append(order, name+":after")
			})
		}
	}
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		order = append(order, "handler")
	})
	if err := Register(mux, "GET /items/{id}", handler, named("first"), named("second")); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/items/42", nil))
	if got, want := order, []string{"first:before", "second:before", "handler", "second:after", "first:after"}; !slices.Equal(got, want) {
		t.Fatalf("middleware order = %v, want %v", got, want)
	}
	if err := Register(mux, "GET /items/{id}", handler); err == nil ||
		!strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("Register(duplicate) error = %v", err)
	}
	if err := Register(mux, "invalid pattern", handler); err == nil {
		t.Fatal("Register(invalid) error = nil")
	}
	if err := Register(nil, "GET /", handler); err == nil {
		t.Fatal("Register(nil mux) error = nil")
	}
	if err := Register(http.NewServeMux(), "GET /", nil); err == nil {
		t.Fatal("Register(nil handler) error = nil")
	}
	if err := Register(http.NewServeMux(), "GET /", handler, nil); err == nil ||
		!strings.Contains(err.Error(), "middleware 0 is nil") {
		t.Fatalf("Register(nil middleware) error = %v", err)
	}
	returnsNil := func(http.Handler) http.Handler { return nil }
	if err := Register(http.NewServeMux(), "GET /", handler, returnsNil); err == nil ||
		!strings.Contains(err.Error(), "middleware 0 returned nil") {
		t.Fatalf("Register(nil middleware result) error = %v", err)
	}
}

func TestObservationMiddlewarePreservesContextOrderAndResult(t *testing.T) {
	type contextKey struct{}
	route := RouteMetadata{
		ID:      "route-id",
		Module:  "example.com/orders",
		Method:  http.MethodPost,
		Pattern: "/orders",
	}
	var events []string
	var firstResult, secondResult HTTPResult
	first := HTTPObserverFunc(func(ctx context.Context, got RouteMetadata) (context.Context, func(HTTPResult)) {
		if got != route {
			t.Fatalf("first route = %#v", got)
		}
		events = append(events, "first:begin")
		return context.WithValue(ctx, contextKey{}, "observed"), func(result HTTPResult) {
			firstResult = result
			events = append(events, "first:finish")
		}
	})
	second := HTTPObserverFunc(func(ctx context.Context, got RouteMetadata) (context.Context, func(HTTPResult)) {
		if got != route || ctx.Value(contextKey{}) != "observed" {
			t.Fatalf("second begin = %#v, context=%v", got, ctx.Value(contextKey{}))
		}
		events = append(events, "second:begin")
		return nil, func(result HTTPResult) {
			secondResult = result
			events = append(events, "second:finish")
		}
	})
	middleware, err := ObservationMiddleware(route, first, second)
	if err != nil {
		t.Fatalf("ObservationMiddleware() error = %v", err)
	}
	handler := middleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Context().Value(contextKey{}) != "observed" {
			t.Fatal("handler did not receive observed context")
		}
		writer.WriteHeader(http.StatusCreated)
		if _, err := writer.Write([]byte("body")); err != nil {
			t.Fatal(err)
		}
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/orders", nil))
	if response.Code != http.StatusCreated || response.Body.String() != "body" {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
	if got, want := events, []string{"first:begin", "second:begin", "second:finish", "first:finish"}; !slices.Equal(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for _, result := range []HTTPResult{firstResult, secondResult} {
		if result.Status != http.StatusCreated || result.Bytes != 4 ||
			result.Duration < 0 || result.Panicked {
			t.Fatalf("HTTPResult = %#v", result)
		}
	}
}

func TestObservationMiddlewareReportsPanicsAndValidatesInputs(t *testing.T) {
	route := RouteMetadata{ID: "route", Method: http.MethodGet, Pattern: "/"}
	var result HTTPResult
	observer := HTTPObserverFunc(func(ctx context.Context, _ RouteMetadata) (context.Context, func(HTTPResult)) {
		return ctx, func(got HTTPResult) { result = got }
	})
	middleware, err := ObservationMiddleware(route, observer)
	if err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("observed handler did not panic")
			}
		}()
		middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic("broken")
		})).ServeHTTP(
			httptest.NewRecorder(),
			httptest.NewRequest(http.MethodGet, "/", nil),
		)
	}()
	if result.Status != http.StatusInternalServerError || !result.Panicked ||
		result.Bytes != 0 {
		t.Fatalf("panic result = %#v", result)
	}

	var nilObserver HTTPObserverFunc
	tests := []struct {
		name      string
		route     RouteMetadata
		observers []HTTPObserver
	}{
		{name: "missing ID", route: RouteMetadata{Method: http.MethodGet, Pattern: "/"}},
		{name: "lowercase method", route: RouteMetadata{ID: "id", Method: "get", Pattern: "/"}},
		{name: "relative pattern", route: RouteMetadata{ID: "id", Method: http.MethodGet, Pattern: "relative"}},
		{name: "nil observer", route: route, observers: []HTTPObserver{nil}},
		{name: "typed nil observer", route: route, observers: []HTTPObserver{nilObserver}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ObservationMiddleware(test.route, test.observers...); err == nil {
				t.Fatal("ObservationMiddleware() error = nil")
			}
		})
	}
}

func TestObservedResponseWriterSupportsHTTPInterfaces(t *testing.T) {
	response := httptest.NewRecorder()
	writer := &observedResponseWriter{ResponseWriter: response}
	if writer.Unwrap() != response {
		t.Fatal("Unwrap() did not return the underlying writer")
	}
	if _, _, err := writer.Hijack(); !errors.Is(err, http.ErrNotSupported) {
		t.Fatalf("Hijack() error = %v", err)
	}
	if err := writer.Push("/asset", nil); !errors.Is(err, http.ErrNotSupported) {
		t.Fatalf("Push() error = %v", err)
	}
	written, err := writer.ReadFrom(io.LimitReader(strings.NewReader("stream"), 6))
	if err != nil || written != 6 || writer.bytes != 6 ||
		writer.status != http.StatusOK {
		t.Fatalf("ReadFrom() = %d, %v; writer=%#v", written, err, writer)
	}
	writer.Flush()
	if !response.Flushed {
		t.Fatal("Flush() did not flush the underlying writer")
	}
}

func TestRegisterObservedMeasuresMiddlewareShortCircuits(t *testing.T) {
	var result HTTPResult
	observation, err := ObservationMiddleware(
		RouteMetadata{ID: "route", Method: http.MethodGet, Pattern: "/secure"},
		HTTPObserverFunc(func(ctx context.Context, _ RouteMetadata) (context.Context, func(HTTPResult)) {
			return ctx, func(got HTTPResult) { result = got }
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})
	reject := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusUnauthorized)
		})
	}
	mux := http.NewServeMux()
	if err := RegisterObserved(mux, "GET /secure", handler, observation, reject); err != nil {
		t.Fatalf("RegisterObserved() error = %v", err)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/secure", nil))
	if called || response.Code != http.StatusUnauthorized ||
		result.Status != http.StatusUnauthorized {
		t.Fatalf("observed short circuit: called=%t response=%d result=%#v", called, response.Code, result)
	}
	if err := RegisterObserved(http.NewServeMux(), "GET /", handler, observation, nil); err == nil ||
		!strings.Contains(err.Error(), "middleware 0 is nil") {
		t.Fatalf("nil caller middleware error = %v", err)
	}
	panics := func(http.Handler) http.Handler { panic("construction panic") }
	if err := RegisterObserved(http.NewServeMux(), "GET /", handler, observation, panics); err == nil ||
		!strings.Contains(err.Error(), "construction panic") {
		t.Fatalf("panicking caller middleware error = %v", err)
	}
	if err := RegisterObserved(nil, "GET /", handler, observation); err == nil {
		t.Fatal("nil mux error = nil")
	}
}

func TestValidateMapsOrdinaryErrorsAndPreservesProblems(t *testing.T) {
	cause := errors.New("secret validator detail")
	err := Validate(context.Background(), func(context.Context) error {
		return cause
	})
	var binding *BindingError
	if !errors.As(err, &binding) || !errors.Is(err, cause) ||
		strings.Contains(binding.Error(), cause.Error()) ||
		binding.Location != LocationRequest {
		t.Fatalf("Validate() error = %T %v", err, err)
	}
	explicit := NewError(Problem{Title: "Conflict", Status: http.StatusConflict}, cause)
	if got := Validate(context.Background(), func(context.Context) error {
		return explicit
	}); !errors.Is(got, explicit) {
		t.Fatalf("Validate(problem) = %v, want same error", got)
	}
	if err := Validate(context.Background(), func(context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("Validate(success) error = %v", err)
	}
	if err := Validate(nil, func(context.Context) error { return nil }); err == nil { //nolint:staticcheck // Verify the defensive public API.
		t.Fatal("Validate(nil context) error = nil")
	}
	if err := Validate(context.Background(), nil); err == nil {
		t.Fatal("Validate(nil validator) error = nil")
	}
}

func TestProblemValidationAndFallbacks(t *testing.T) {
	tests := []Problem{
		{Status: 399},
		{Status: 600},
		{Status: 599},
		{Status: 400, Type: "https://example.com/%zz"},
		{Status: 400, Instance: "https://example.com/%zz"},
	}
	for _, problem := range tests {
		if err := problem.Validate(); err == nil {
			t.Fatalf("Problem.Validate(%#v) error = nil", problem)
		}
	}
	if err := (Problem{Status: 400}).Validate(); err != nil {
		t.Fatalf("valid Problem.Validate() error = %v", err)
	}
	if err := (Problem{Status: 599, Title: "Custom failure"}).Validate(); err != nil {
		t.Fatalf("custom Problem.Validate() error = %v", err)
	}

	response := httptest.NewRecorder()
	if err := WriteProblem(response, Problem{Status: 200, Detail: "must not leak"}); err != nil {
		t.Fatalf("WriteProblem() error = %v", err)
	}
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "must not leak") {
		t.Fatalf("invalid problem response = %d %s", response.Code, response.Body.String())
	}
	if err := WriteProblem(nil, Problem{}); err == nil {
		t.Fatal("WriteProblem(nil) error = nil")
	}

	var nilHTTPError *HTTPError
	if nilHTTPError.Error() != "HTTP error" || nilHTTPError.Unwrap() != nil ||
		nilHTTPError.Problem() != internalProblem() {
		t.Fatalf("nil HTTPError methods are unsafe")
	}
}

func TestJSONAndNoContentWriters(t *testing.T) {
	response := httptest.NewRecorder()
	if err := WriteJSON(response, http.StatusCreated, map[string]string{"status": "created"}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	if response.Code != http.StatusCreated ||
		response.Header().Get("Content-Type") != jsonMediaType ||
		response.Body.String() != "{\"status\":\"created\"}\n" {
		t.Fatalf("JSON response = %d %#v %q", response.Code, response.Header(), response.Body.String())
	}
	if err := WriteJSON(nil, 200, nil); err == nil {
		t.Fatal("WriteJSON(nil) error = nil")
	}
	if err := WriteJSON(httptest.NewRecorder(), 199, nil); err == nil {
		t.Fatal("WriteJSON(invalid status) error = nil")
	}

	response = httptest.NewRecorder()
	if err := WriteNoContent(response); err != nil {
		t.Fatalf("WriteNoContent() error = %v", err)
	}
	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Fatalf("no-content response = %d %q", response.Code, response.Body.String())
	}
	if err := WriteNoContent(nil); err == nil {
		t.Fatal("WriteNoContent(nil) error = nil")
	}
}

func TestAcceptsJSON(t *testing.T) {
	tests := []struct {
		header string
		want   bool
	}{
		{header: "", want: true},
		{header: "*/*", want: true},
		{header: "application/*", want: true},
		{header: "application/json", want: true},
		{header: "application/problem+json", want: true},
		{header: "text/plain, application/json;q=0.8", want: true},
		{header: "application/json;q=0", want: false},
		{header: "application/json;q=2", want: false},
		{header: "application/json;q=broken", want: false},
		{header: "text/plain", want: false},
		{header: "not a media type", want: false},
	}
	for _, test := range tests {
		if got := AcceptsJSON(test.header); got != test.want {
			t.Errorf("AcceptsJSON(%q) = %t, want %t", test.header, got, test.want)
		}
	}
}

func TestDecodeJSONStrictBoundedContract(t *testing.T) {
	type requestBody struct {
		Name string `json:"name"`
	}
	valid := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Spice"}`))
	valid.Header.Set("Content-Type", "application/json; charset=utf-8")
	var decoded requestBody
	if err := DecodeJSON(valid, &decoded, 0); err != nil || decoded.Name != "Spice" {
		t.Fatalf("DecodeJSON(valid) = %#v, %v", decoded, err)
	}

	tests := []struct {
		name        string
		contentType string
		body        string
		maxBytes    int64
	}{
		{name: "missing content type", body: `{}`},
		{name: "wrong content type", contentType: "text/plain", body: `{}`},
		{name: "unknown field", contentType: "application/json", body: `{"other":1}`},
		{name: "invalid", contentType: "application/json", body: `{`},
		{name: "two values", contentType: "application/json", body: `{} {}`},
		{name: "trailing invalid", contentType: "application/json", body: `{} x`},
		{name: "too large", contentType: "application/json", body: `{"name":"Spice"}`, maxBytes: 4},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			var destination requestBody
			err := DecodeJSON(request, &destination, test.maxBytes)
			var binding *BindingError
			if !errors.As(err, &binding) || binding.Problem().Status != http.StatusBadRequest {
				t.Fatalf("DecodeJSON() error = %T %v", err, err)
			}
		})
	}

	if err := DecodeJSON(nil, &decoded, 1); err == nil {
		t.Fatal("DecodeJSON(nil request) error = nil")
	}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	if err := DecodeJSON(request, nil, 1); err == nil {
		t.Fatal("DecodeJSON(nil destination) error = nil")
	}
	if err := DecodeJSON(request, &decoded, -1); err == nil {
		t.Fatal("DecodeJSON(negative bound) error = nil")
	}
	request = &http.Request{Header: http.Header{"Content-Type": []string{"application/json"}}}
	if err := DecodeJSON(request, &decoded, 1); err == nil {
		t.Fatal("DecodeJSON(nil body) error = nil")
	}
}

func TestDecodeJSONReadFailure(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Content-Type", "application/json")
	request.Body = io.NopCloser(failingReader{})
	var destination map[string]any
	err := DecodeJSON(request, &destination, 100)
	var binding *BindingError
	if !errors.As(err, &binding) || !errors.Is(err, errRead) {
		t.Fatalf("DecodeJSON(read failure) error = %T %v", err, err)
	}
}

func TestParameterAndScalarBinding(t *testing.T) {
	if value, present, err := Parameter(LocationQuery, "name", nil, false); err != nil || present || value != "" {
		t.Fatalf("optional Parameter() = %q, %t, %v", value, present, err)
	}
	if value, present, err := Parameter(LocationQuery, "name", []string{"spice"}, true); err != nil ||
		!present || value != "spice" {
		t.Fatalf("required Parameter() = %q, %t, %v", value, present, err)
	}
	for _, values := range [][]string{nil, {"one", "two"}} {
		_, _, err := Parameter(LocationQuery, "name", values, true)
		var binding *BindingError
		if !errors.As(err, &binding) {
			t.Fatalf("Parameter(%v) error = %T %v", values, err, err)
		}
		for _, raw := range values {
			if strings.Contains(binding.Error(), raw) {
				t.Fatalf("Parameter(%v) leaked raw value in %v", values, err)
			}
		}
	}

	if value, err := Boolean(LocationQuery, "enabled", "true"); err != nil || !value {
		t.Fatalf("Boolean() = %t, %v", value, err)
	}
	if _, err := Boolean(LocationQuery, "enabled", "secret-raw"); err == nil ||
		strings.Contains(err.Error(), "secret-raw") {
		t.Fatalf("Boolean(invalid) error = %v", err)
	}
	if value, err := Integer(LocationPath, "id", "127", 8); err != nil || value != 127 {
		t.Fatalf("Integer() = %d, %v", value, err)
	}
	if _, err := Integer(LocationPath, "id", "128", 8); err == nil || strings.Contains(err.Error(), "128") {
		t.Fatalf("Integer(overflow) error = %v", err)
	}
	if value, err := Duration(LocationHeader, "Timeout", "5s"); err != nil || value != 5*time.Second {
		t.Fatalf("Duration() = %s, %v", value, err)
	}
	if _, err := Duration(LocationHeader, "Timeout", "raw-secret"); err == nil ||
		strings.Contains(err.Error(), "raw-secret") {
		t.Fatalf("Duration(invalid) error = %v", err)
	}

	var nilBinding *BindingError
	if nilBinding.Error() != "bind request" || nilBinding.Unwrap() != nil {
		t.Fatal("nil BindingError methods are unsafe")
	}
}

var errRead = errors.New("read failed")

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errRead
}
