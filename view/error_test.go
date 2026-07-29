package view

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/StevenBuglione/spice/web"
)

func TestRendererWritesSafeHTMLErrorsAndJSONFallback(t *testing.T) {
	t.Parallel()

	source := fstest.MapFS{
		"error.html": {Data: []byte(
			`{{define "error"}}<h1>{{.Title}}</h1><p>{{.Detail}}</p>{{end}}`,
		)},
	}
	renderer := newTestRenderer(
		t,
		source,
		[]string{"*.html"},
		template.FuncMap{},
		Options{
			ErrorTemplate: "error",
			ErrorModel: func(_ *http.Request, problem web.Problem) any {
				return problem
			},
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	err := renderer.WriteError(
		response,
		request,
		web.NewError(web.Problem{
			Title:  "Missing",
			Status: http.StatusNotFound,
			Detail: "The page is absent.",
		}, errors.New("private cause")),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusNotFound ||
		response.Header().Get("Content-Type") != "text/html; charset=utf-8" ||
		!strings.Contains(response.Body.String(), "<h1>Missing</h1>") ||
		strings.Contains(response.Body.String(), "private cause") {
		t.Fatalf("HTML error = %d %v %q", response.Code, response.Header(), response.Body)
	}

	fallback := newTestRenderer(
		t,
		fstest.MapFS{"page.html": {Data: []byte("page")}},
		[]string{"*.html"},
		nil,
		Options{},
	)
	response = httptest.NewRecorder()
	if err := fallback.WriteError(response, request, errors.New("private"), nil); err != nil {
		t.Fatal(err)
	}
	if response.Header().Get("Content-Type") != "application/problem+json" ||
		strings.Contains(response.Body.String(), "private") {
		t.Fatalf("fallback error = %v %q", response.Header(), response.Body)
	}
}

func TestRendererRejectsInvalidErrorConfiguration(t *testing.T) {
	t.Parallel()

	source := fstest.MapFS{"page.html": {Data: []byte("page")}}
	tests := []Options{
		{ErrorTemplate: "page.html"},
		{ErrorModel: func(*http.Request, web.Problem) any { return nil }},
		{
			ErrorTemplate: "missing",
			ErrorModel:    func(*http.Request, web.Problem) any { return nil },
		},
	}
	for _, options := range tests {
		if _, err := Parse(source, []string{"*.html"}, nil, options); err == nil {
			t.Fatalf("Parse(%+v) error = nil", options)
		}
	}
	if err := (*Renderer)(nil).WriteError(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/", nil),
		errors.New("failure"),
		nil,
	); err == nil {
		t.Fatal("nil WriteError() error = nil")
	}
}
