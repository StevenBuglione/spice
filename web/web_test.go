package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
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
