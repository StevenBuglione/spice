package view

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
)

func TestRendererEscapesHTMLAndWritesOnlyAfterSuccess(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t, fstest.MapFS{
		"templates/00-layout.html": {
			Data: []byte(
				`{{define "layout"}}<!doctype html><body>{{template "orders" .}}</body>{{end}}`,
			),
		},
		"templates/10-orders.html": {
			Data: []byte(
				`{{define "orders"}}<h1>{{upper .Title}}</h1><p>{{.Unsafe}}</p>{{end}}` +
					`{{define "page"}}{{template "layout" .}}{{end}}`,
			),
		},
	}, []string{"templates/*.html"}, template.FuncMap{
		"upper": strings.ToUpper,
	}, Options{})
	names := renderer.TemplateNames()
	if !slices.Equal(names, []string{
		"00-layout.html",
		"10-orders.html",
		"layout",
		"orders",
		"page",
	}) {
		t.Fatalf("template names = %v", names)
	}
	if len(names) == 0 {
		t.Fatal("template names are empty")
	}
	names[0] = "changed"
	currentNames := renderer.TemplateNames()
	if len(currentNames) == 0 || currentNames[0] == "changed" {
		t.Fatal("TemplateNames() exposed mutable state")
	}

	recorder := httptest.NewRecorder()
	err := renderer.Render(
		context.Background(),
		recorder,
		"page",
		http.StatusCreated,
		map[string]string{
			"Title":  "orders",
			"Unsafe": `<script>alert("x")</script>`,
		},
	)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if recorder.Code != http.StatusCreated ||
		recorder.Header().Get("Content-Type") != "text/html; charset=utf-8" ||
		recorder.Header().Get("X-Content-Type-Options") != "nosniff" ||
		recorder.Header().Get("Content-Length") != fmt.Sprint(recorder.Body.Len()) ||
		recorder.Body.String() !=
			`<!doctype html><body><h1>ORDERS</h1><p>&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;</p></body>` {
		t.Fatalf(
			"response code=%d header=%v body=%q",
			recorder.Code,
			recorder.Header(),
			recorder.Body.String(),
		)
	}
}

func TestRendererFailureDoesNotMutateResponse(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t, fstest.MapFS{
		"page.html": {Data: []byte(`<h1>{{.Title}}</h1>`)},
	}, []string{"*.html"}, nil, Options{MaxOutputBytes: 16})
	tests := []struct {
		name string
		data any
		err  error
	}{
		{name: "missing key", data: map[string]string{}, err: nil},
		{name: "output limit", data: map[string]string{
			"Title": strings.Repeat("x", 32),
		}, err: ErrOutputLimit},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			err := renderer.Render(
				context.Background(),
				recorder,
				"page.html",
				http.StatusOK,
				test.data,
			)
			if err == nil {
				t.Fatal("Render() unexpectedly succeeded")
			}
			if test.err != nil && !errors.Is(err, test.err) {
				t.Fatalf("Render() error = %v", err)
			}
			if recorder.Body.Len() != 0 || len(recorder.Header()) != 0 {
				t.Fatalf(
					"failed render wrote header=%v body=%q",
					recorder.Header(),
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestRendererPreservesCancellationAndWriteFailures(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t, fstest.MapFS{
		"page.html": {Data: []byte(`hello {{.}}`)},
	}, []string{"*.html"}, nil, Options{})
	canceled, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("request canceled"))
	recorder := httptest.NewRecorder()
	err := renderer.Render(
		canceled,
		recorder,
		"page.html",
		http.StatusOK,
		"world",
	)
	if err == nil {
		t.Fatal("canceled Render() unexpectedly succeeded")
	}
	if !errors.Is(err, context.Canceled) &&
		!strings.Contains(err.Error(), "request canceled") {
		t.Fatalf("canceled Render() error = %v", err)
	}
	if len(recorder.Header()) != 0 || recorder.Body.Len() != 0 {
		t.Fatal("canceled render mutated response")
	}

	writeFailure := errors.New("network write failed")
	failing := &testResponseWriter{writeErr: writeFailure}
	if err := renderer.Render(
		context.Background(),
		failing,
		"page.html",
		http.StatusOK,
		"world",
	); !errors.Is(err, writeFailure) {
		t.Fatalf("write-failure Render() error = %v", err)
	}
	short := &testResponseWriter{short: true}
	if err := renderer.Render(
		context.Background(),
		short,
		"page.html",
		http.StatusOK,
		"world",
	); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short-write Render() error = %v", err)
	}
}

func TestParseRejectsInvalidSourcesPatternsFunctionsAndDuplicates(t *testing.T) {
	t.Parallel()

	validFS := fstest.MapFS{"page.html": {Data: []byte(`hello`)}}
	var nilFS *nilTemplateFS
	tooManyPatterns := make([]string, maxTemplatePatterns+1)
	for index := range tooManyPatterns {
		tooManyPatterns[index] = "*.html"
	}
	tests := []struct {
		name      string
		source    fs.FS
		patterns  []string
		functions template.FuncMap
		options   Options
	}{
		{name: "nil filesystem", source: nilFS, patterns: []string{"*.html"}},
		{name: "no patterns", source: validFS},
		{name: "too many patterns", source: validFS, patterns: tooManyPatterns},
		{name: "empty pattern", source: validFS, patterns: []string{""}},
		{name: "spaced pattern", source: validFS, patterns: []string{" *.html"}},
		{name: "absolute pattern", source: validFS, patterns: []string{"/page.html"}},
		{name: "backslash pattern", source: validFS, patterns: []string{`templates\*.html`}},
		{name: "traversal pattern", source: validFS, patterns: []string{"../*.html"}},
		{name: "long pattern", source: validFS, patterns: []string{
			strings.Repeat("x", maxTemplatePathBytes+1),
		}},
		{name: "invalid glob", source: validFS, patterns: []string{"["}},
		{name: "no matches", source: validFS, patterns: []string{"*.tmpl"}},
		{name: "negative output", source: validFS, patterns: []string{"*.html"}, options: Options{
			MaxOutputBytes: -1,
		}},
		{name: "large output", source: validFS, patterns: []string{"*.html"}, options: Options{
			MaxOutputBytes: maxOutputBytes + 1,
		}},
		{name: "invalid function name", source: validFS, patterns: []string{"*.html"}, functions: template.FuncMap{
			"bad.name": strings.ToUpper,
		}},
		{name: "empty function name", source: validFS, patterns: []string{"*.html"}, functions: template.FuncMap{
			"": strings.ToUpper,
		}},
		{name: "long function name", source: validFS, patterns: []string{"*.html"}, functions: template.FuncMap{
			strings.Repeat("x", maxFunctionNameBytes+1): strings.ToUpper,
		}},
		{name: "nil function", source: validFS, patterns: []string{"*.html"}, functions: template.FuncMap{
			"helper": nil,
		}},
		{name: "non-function", source: validFS, patterns: []string{"*.html"}, functions: template.FuncMap{
			"helper": 41,
		}},
		{name: "typed nil function", source: validFS, patterns: []string{"*.html"}, functions: template.FuncMap{
			"helper": (func(string) string)(nil),
		}},
		{name: "no function result", source: validFS, patterns: []string{"*.html"}, functions: template.FuncMap{
			"helper": func() {},
		}},
		{name: "three function results", source: validFS, patterns: []string{"*.html"}, functions: template.FuncMap{
			"helper": func() (int, int, int) { return 1, 2, 3 },
		}},
		{name: "invalid second result", source: validFS, patterns: []string{"*.html"}, functions: template.FuncMap{
			"helper": func() (int, string) { return 1, "error" },
		}},
		{name: "malformed template", source: fstest.MapFS{
			"page.html": {Data: []byte(`{{if}}`)},
		}, patterns: []string{"*.html"}},
		{name: "directory source", source: fstest.MapFS{
			"templates": {Mode: fs.ModeDir},
		}, patterns: []string{"*"}},
		{name: "source limit", source: fstest.MapFS{
			"page.html": {Data: []byte(strings.Repeat("x", maxTemplateSource+1))},
		}, patterns: []string{"*.html"}},
		{name: "duplicate definition", source: fstest.MapFS{
			"first.html":  {Data: []byte(`{{define "content"}}first{{end}}`)},
			"second.html": {Data: []byte(`{{define "content"}}second{{end}}`)},
		}, patterns: []string{"*.html"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if _, err := Parse(
				test.source,
				test.patterns,
				test.functions,
				test.options,
			); err == nil {
				t.Fatal("Parse() unexpectedly succeeded")
			}
		})
	}
}

func TestRendererRejectsInvalidInvocation(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t, fstest.MapFS{
		"page.html": {Data: []byte(`hello`)},
	}, []string{"*.html"}, nil, Options{})
	var nilWriter *httptest.ResponseRecorder
	tests := []struct {
		name     string
		renderer *Renderer
		context  func() context.Context
		writer   http.ResponseWriter
		template string
		status   int
	}{
		{name: "nil renderer", context: context.Background, writer: httptest.NewRecorder(), template: "page.html", status: 200},
		{name: "nil context", renderer: renderer, writer: httptest.NewRecorder(), template: "page.html", status: 200},
		{name: "nil writer", renderer: renderer, context: context.Background, writer: nilWriter, template: "page.html", status: 200},
		{name: "unknown template", renderer: renderer, context: context.Background, writer: httptest.NewRecorder(), template: "missing", status: 200},
		{name: "informational", renderer: renderer, context: context.Background, writer: httptest.NewRecorder(), template: "page.html", status: 199},
		{name: "no content", renderer: renderer, context: context.Background, writer: httptest.NewRecorder(), template: "page.html", status: 204},
		{name: "not modified", renderer: renderer, context: context.Background, writer: httptest.NewRecorder(), template: "page.html", status: 304},
		{name: "large status", renderer: renderer, context: context.Background, writer: httptest.NewRecorder(), template: "page.html", status: 600},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var ctx context.Context
			if test.context != nil {
				ctx = test.context()
			}
			if err := test.renderer.Render(
				ctx,
				test.writer,
				test.template,
				test.status,
				nil,
			); err == nil {
				t.Fatal("Render() unexpectedly succeeded")
			}
		})
	}

	var nilRenderer *Renderer
	if nilRenderer.TemplateNames() != nil {
		t.Fatal("nil TemplateNames() was non-nil")
	}
}

func TestRendererIsSafeForConcurrentExecution(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t, fstest.MapFS{
		"page.html": {Data: []byte(`<p>{{.}}</p>`)},
	}, []string{"*.html"}, nil, Options{})
	const workers = 32
	start := make(chan struct{})
	failures := make(chan error, workers)
	var wait sync.WaitGroup
	for index := range workers {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			<-start
			recorder := httptest.NewRecorder()
			value := fmt.Sprintf("worker-%d", worker)
			if err := renderer.Render(
				context.Background(),
				recorder,
				"page.html",
				http.StatusOK,
				value,
			); err != nil {
				failures <- err
				return
			}
			if recorder.Body.String() != "<p>"+value+"</p>" {
				failures <- fmt.Errorf("worker %d body = %q", worker, recorder.Body.String())
			}
		}(index)
	}
	close(start)
	wait.Wait()
	close(failures)
	for err := range failures {
		t.Errorf("concurrent render: %v", err)
	}
}

func FuzzRendererEscaping(f *testing.F) {
	renderer, err := Parse(
		fstest.MapFS{"page.html": {Data: []byte(`<p>{{.}}</p>`)}},
		[]string{"*.html"},
		nil,
		Options{MaxOutputBytes: 4096},
	)
	if err != nil {
		f.Fatalf("Parse() error = %v", err)
	}
	f.Add(`<script>alert("x")</script>`)
	f.Add("orders")
	f.Fuzz(func(t *testing.T, value string) {
		recorder := httptest.NewRecorder()
		err := renderer.Render(
			context.Background(),
			recorder,
			"page.html",
			http.StatusOK,
			value,
		)
		if err != nil && !errors.Is(err, ErrOutputLimit) {
			t.Fatalf("Render() error = %v", err)
		}
		if err == nil && strings.Contains(recorder.Body.String(), "<script") {
			t.Fatalf("rendered unescaped script: %q", recorder.Body.String())
		}
	})
}

func ExampleRenderer() {
	renderer, err := Parse(
		fstest.MapFS{
			"welcome.html": {Data: []byte(`<h1>Hello, {{.Name}}</h1>`)},
		},
		[]string{"*.html"},
		nil,
		Options{},
	)
	if err != nil {
		panic(err)
	}
	recorder := httptest.NewRecorder()
	if err := renderer.Render(
		context.Background(),
		recorder,
		"welcome.html",
		http.StatusOK,
		struct{ Name string }{Name: "Spice"},
	); err != nil {
		panic(err)
	}
	fmt.Println(recorder.Body.String())
	// Output: <h1>Hello, Spice</h1>
}

type testResponseWriter struct {
	header   http.Header
	writeErr error
	short    bool
}

func (writer *testResponseWriter) Header() http.Header {
	if writer.header == nil {
		writer.header = make(http.Header)
	}
	return writer.header
}

func (*testResponseWriter) WriteHeader(int) {}

func (writer *testResponseWriter) Write(value []byte) (int, error) {
	if writer.writeErr != nil {
		return 0, writer.writeErr
	}
	if writer.short {
		return len(value) / 2, nil
	}
	return len(value), nil
}

type nilTemplateFS struct{}

func (*nilTemplateFS) Open(string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

func newTestRenderer(
	tb testing.TB,
	source fs.FS,
	patterns []string,
	functions template.FuncMap,
	options Options,
) *Renderer {
	tb.Helper()
	renderer, err := Parse(source, patterns, functions, options)
	if err != nil {
		tb.Fatalf("Parse() error = %v", err)
	}
	return renderer
}
