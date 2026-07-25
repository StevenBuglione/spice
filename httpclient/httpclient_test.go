package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func FuzzResolveReference(f *testing.F) {
	for _, seed := range []string{
		"items/42?verbose=true",
		"",
		"../outside",
		"%2e%2e/outside",
		"//other.example/path",
		`items\other`,
	} {
		f.Add(seed)
	}
	client, err := New(Options{BaseURL: "https://example.com/api"})
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, reference string) {
		if len(reference) > 4<<10 {
			t.Skip()
		}
		request, err := client.NewRequest(context.Background(), http.MethodGet, reference, nil)
		if err != nil {
			return
		}
		if !withinBaseURL(client.baseURL, request.URL) ||
			request.URL.Fragment != "" ||
			hasDotSegment(request.URL.Path) {
			t.Fatalf("escaped request URL %q from reference %q", request.URL, reference)
		}
	})
}

func TestDoJSONScopesRequestsAndDecodesResponses(t *testing.T) {
	type requestBody struct {
		Name string `json:"name"`
	}
	type responseBody struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost ||
			request.URL.Path != "/api/v1/items/42" ||
			request.URL.Query().Get("verbose") != "true" ||
			request.Header.Get("Authorization") != "Bearer token" ||
			request.Header.Get("User-Agent") != "spice-test" ||
			request.Header.Get("Accept") != "application/json" ||
			request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("request = %s %s headers=%#v", request.Method, request.URL, request.Header)
		}
		var body requestBody
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("X-Request-ID", "request-1")
		if _, err := io.WriteString(writer, `{"id":"42","name":"`+body.Name+`","future":true}`); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()

	headers := http.Header{"Authorization": {"Bearer token"}}
	client, err := New(Options{
		BaseURL:        server.URL + "/api/v1",
		DefaultHeaders: headers,
		UserAgent:      "spice-test",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	headers.Set("Authorization", "changed")
	if client.BaseURL() != server.URL+"/api/v1/" {
		t.Fatalf("BaseURL() = %q", client.BaseURL())
	}
	if client.httpClient.Timeout != DefaultTimeout {
		t.Fatalf("default timeout = %s", client.httpClient.Timeout)
	}
	response, err := DoJSON[responseBody](
		context.Background(),
		client,
		http.MethodPost,
		"items/42?verbose=true",
		requestBody{Name: "Spice"},
	)
	if err != nil {
		t.Fatalf("DoJSON() error = %v", err)
	}
	if response.Status != http.StatusOK ||
		response.Header.Get("X-Request-ID") != "request-1" ||
		response.Value != (responseBody{ID: "42", Name: "Spice"}) {
		t.Fatalf("DoJSON() = %#v", response)
	}
}

func TestDoJSONHandlesEmptyProblemStrictAndBoundedResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/empty":
			writer.WriteHeader(http.StatusNoContent)
		case "/api/problem":
			writer.Header().Set("Content-Type", "application/problem+json")
			writer.WriteHeader(http.StatusConflict)
			writeResponse(t, writer, `{"type":"https://example.com/conflict","title":"Conflict","status":409,"detail":"remote secret"}`)
		case "/api/text-error":
			writer.Header().Set("Content-Type", "text/plain")
			writer.WriteHeader(http.StatusBadGateway)
			writeResponse(t, writer, "remote secret")
		case "/api/unknown":
			writer.Header().Set("Content-Type", "application/json")
			writeResponse(t, writer, `{"value":"ok","future":true}`)
		case "/api/large":
			writer.Header().Set("Content-Type", "application/json")
			writeResponse(t, writer, `{"value":"123456789"}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := New(Options{BaseURL: server.URL + "/api", MaxResponseBodyBytes: 16})
	if err != nil {
		t.Fatal(err)
	}
	type responseBody struct {
		Value string `json:"value"`
	}
	empty, err := DoJSON[responseBody](context.Background(), client, http.MethodGet, "empty", nil)
	if err != nil || empty.Status != http.StatusNoContent || empty.Value != (responseBody{}) {
		t.Fatalf("empty response = %#v, %v", empty, err)
	}
	_, err = DoJSON[responseBody](context.Background(), client, http.MethodGet, "problem", nil)
	var responseErr *ResponseError
	if !errors.As(err, &responseErr) || responseErr.Status != http.StatusConflict ||
		strings.Contains(responseErr.Error(), "remote secret") {
		t.Fatalf("problem error = %T %v", err, err)
	}
	if problem, found := responseErr.RemoteProblem(); found {
		t.Fatalf("oversized problem unexpectedly decoded: %#v", problem)
	}
	_, err = DoJSON[responseBody](context.Background(), client, http.MethodGet, "text-error", nil)
	if !errors.As(err, &responseErr) || strings.Contains(err.Error(), "remote secret") {
		t.Fatalf("text error = %T %v", err, err)
	}
	_, err = DoJSON[responseBody](context.Background(), client, http.MethodGet, "large", nil)
	if err == nil || !strings.Contains(err.Error(), "exceeds 16 bytes") {
		t.Fatalf("large response error = %v", err)
	}

	strict, err := New(Options{
		BaseURL:                       server.URL + "/api",
		DisallowUnknownResponseFields: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = DoJSON[responseBody](context.Background(), strict, http.MethodGet, "unknown", nil)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("strict response error = %v", err)
	}
}

func TestResponseErrorReturnsValidatedRemoteProblem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/problem+json")
		writer.WriteHeader(http.StatusConflict)
		if request.URL.Path == "/problem-invalid" {
			writeResponse(t, writer, `{"title":"Bad Request","status":400}`)
			return
		}
		writeResponse(t, writer, `{"type":"https://example.com/conflict","title":"Conflict","status":409,"detail":"safe remote detail"}`)
	}))
	defer server.Close()
	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = DoJSON[struct{}](context.Background(), client, http.MethodGet, "problem", nil)
	var responseErr *ResponseError
	if !errors.As(err, &responseErr) {
		t.Fatalf("error = %T %v", err, err)
	}
	problem, found := responseErr.RemoteProblem()
	if !found || problem.Status != http.StatusConflict ||
		problem.Detail != "safe remote detail" ||
		strings.Contains(responseErr.Error(), problem.Detail) {
		t.Fatalf("remote problem = %#v, found=%t, error=%v", problem, found, responseErr)
	}
	_, err = DoJSON[struct{}](context.Background(), client, http.MethodGet, "problem-invalid", nil)
	if !errors.As(err, &responseErr) {
		t.Fatalf("mismatched problem error = %T %v", err, err)
	}
	if problem, found := responseErr.RemoteProblem(); found {
		t.Fatalf("mismatched remote problem = %#v", problem)
	}
}

func TestClientRejectsEscapesRedirectsAndInvalidConfiguration(t *testing.T) {
	var externalCalls atomic.Int64
	external := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		externalCalls.Add(1)
	}))
	defer external.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/external":
			http.Redirect(writer, request, external.URL+"?token=remote-secret", http.StatusFound)
		case "/api/escape":
			http.Redirect(writer, request, "/outside", http.StatusFound)
		case "/api/inside":
			http.Redirect(writer, request, "/api/final", http.StatusFound)
		case "/api/final":
			writer.Header().Set("Content-Type", "application/json")
			writeResponse(t, writer, `{}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer origin.Close()
	client, err := New(Options{BaseURL: origin.URL + "/api"})
	if err != nil {
		t.Fatal(err)
	}
	for _, reference := range []string{
		external.URL,
		"//example.com/path",
		"/outside",
		"../outside",
		"%2e%2e/outside",
		`..\outside`,
		`%2e%2e%5coutside`,
		"inside#fragment",
	} {
		if _, requestErr := client.NewRequest(context.Background(), http.MethodGet, reference, nil); requestErr == nil {
			t.Fatalf("NewRequest(%q) error = nil", reference)
		}
	}
	for _, reference := range []string{"external", "escape"} {
		if _, redirectErr := DoJSON[struct{}](context.Background(), client, http.MethodGet, reference, nil); redirectErr == nil ||
			!strings.Contains(redirectErr.Error(), "outside base URL") ||
			strings.Contains(redirectErr.Error(), "remote-secret") {
			t.Fatalf("redirect %q error = %v", reference, redirectErr)
		}
	}
	if externalCalls.Load() != 0 {
		t.Fatalf("cross-origin redirect made %d request(s)", externalCalls.Load())
	}
	if _, redirectErr := DoJSON[struct{}](context.Background(), client, http.MethodGet, "inside", nil); redirectErr != nil {
		t.Fatalf("same-base redirect error = %v", redirectErr)
	}
	request, err := http.NewRequest(http.MethodGet, external.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	rawResponse, rawErr := client.Do(request)
	closeResponse(t, rawResponse)
	if rawErr == nil {
		t.Fatal("Do(outside base) error = nil")
	}
	request, err = http.NewRequest(http.MethodGet, origin.URL+"/api/final", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "different.example"
	rawResponse, rawErr = client.Do(request)
	closeResponse(t, rawResponse)
	if rawErr == nil || !strings.Contains(rawErr.Error(), "Host override") {
		t.Fatalf("Do(Host override) error = %v", rawErr)
	}

	tests := []Options{
		{},
		{BaseURL: "ftp://example.com"},
		{BaseURL: "https://user:password@example.com"},
		{BaseURL: "https://example.com?query=true"},
		{BaseURL: "https://example.com/a/../b"},
		{BaseURL: `https://example.com/a\backslash`},
		{BaseURL: "https://example.com/a%2Fb"},
		{BaseURL: "https://example.com", MaxResponseBodyBytes: -1},
		{BaseURL: "https://example.com", MaxResponseBodyBytes: MaxResponseBodyBytes + 1},
		{BaseURL: "https://example.com", UserAgent: "bad\nagent"},
		{BaseURL: "https://example.com", DefaultHeaders: http.Header{"Bad Header": {"value"}}},
		{BaseURL: "https://example.com", DefaultHeaders: http.Header{"Host": {"other.example"}}},
		{BaseURL: "https://example.com", DefaultHeaders: http.Header{"X-Test": {"bad\nvalue"}}},
	}
	for index, options := range tests {
		if _, err := New(options); err == nil {
			t.Fatalf("New(invalid %d) error = nil", index)
		}
	}
}

func TestClientHonorsCancellationAndDefensiveAPIs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
		writer.WriteHeader(http.StatusGatewayTimeout)
	}))
	defer server.Close()
	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := DoJSON[struct{}](ctx, client, http.MethodGet, "wait", nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("canceled DoJSON() error = %v", err)
	}
	if _, err := (*Client)(nil).NewRequest(context.Background(), http.MethodGet, "", nil); err == nil {
		t.Fatal("nil NewRequest() error = nil")
	}
	if _, err := client.NewRequest(nil, http.MethodGet, "", nil); err == nil { //nolint:staticcheck // Verify the defensive public API.
		t.Fatal("nil context error = nil")
	}
	if _, err := client.NewRequest(context.Background(), "get", "", nil); err == nil {
		t.Fatal("lowercase method error = nil")
	}
	response, doErr := (*Client)(nil).Do(nil)
	closeResponse(t, response)
	if doErr == nil {
		t.Fatal("nil Client.Do() error = nil")
	}
	response, doErr = client.Do(nil)
	closeResponse(t, response)
	if doErr == nil {
		t.Fatal("Do(nil request) error = nil")
	}
	if _, err := DoJSON[struct{}](context.Background(), nil, http.MethodGet, "", nil); err == nil {
		t.Fatal("DoJSON(nil client) error = nil")
	}
	var responseErr *ResponseError
	if responseErr.Error() == "" {
		t.Fatal("nil ResponseError.Error() is empty")
	}
	if _, found := responseErr.RemoteProblem(); found {
		t.Fatal("nil ResponseError has a remote problem")
	}
	var nilClient *Client
	if nilClient.BaseURL() != "" {
		t.Fatalf("nil BaseURL() = %q", nilClient.BaseURL())
	}
}

func writeResponse(t *testing.T, writer io.Writer, value string) {
	t.Helper()
	if _, err := io.WriteString(writer, value); err != nil {
		t.Error(err)
	}
}

func closeResponse(t *testing.T, response *http.Response) {
	t.Helper()
	if response == nil || response.Body == nil {
		return
	}
	if err := response.Body.Close(); err != nil {
		t.Error(err)
	}
}
