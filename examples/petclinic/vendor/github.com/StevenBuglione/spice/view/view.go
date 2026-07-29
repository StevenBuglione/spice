// Package view provides deterministic, bounded server-side HTML template
// parsing and atomic HTTP rendering.
package view

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/StevenBuglione/spice/web"
)

const (
	defaultMaxOutputBytes = 1 << 20
	maxOutputBytes        = 8 << 20
	maxTemplateSource     = 4 << 20
	maxTemplatePatterns   = 64
	maxTemplatePathBytes  = 512
	maxFunctionNameBytes  = 64
)

// ErrOutputLimit means template execution exceeded the configured response
// bound.
var ErrOutputLimit = errors.New("template output limit exceeded")

// Options configures an immutable renderer.
type Options struct {
	MaxOutputBytes int
	// ErrorTemplate enables HTML problem rendering through WriteError.
	ErrorTemplate string
	// ErrorModel converts a safe problem and request into template data.
	ErrorModel func(*http.Request, web.Problem) any
}

// Renderer is an immutable parsed HTML template set. It is safe for concurrent
// execution when caller-provided template functions are also safe.
type Renderer struct {
	templates  *template.Template
	names      []string
	nameSet    map[string]struct{}
	maxOutput  int
	errorName  string
	errorModel func(*http.Request, web.Problem) any
}

// Parse expands patterns against source, reads matched regular files within a
// fixed aggregate bound, and parses them in lexical path order. Construction
// performs no process-environment or network access.
func Parse(
	source fs.FS,
	patterns []string,
	functions template.FuncMap,
	options Options,
) (*Renderer, error) {
	if nilInterface(source) {
		return nil, errors.New("parse HTML templates: source filesystem is nil")
	}
	maxOutput, err := normalizeMaxOutput(options.MaxOutputBytes)
	if err != nil {
		return nil, err
	}
	files, err := resolveFiles(source, patterns)
	if err != nil {
		return nil, err
	}
	funcs, err := validateFunctions(functions)
	if err != nil {
		return nil, err
	}

	compiled := template.New("_spice").Option("missingkey=error").Funcs(funcs)
	names := make(map[string]string)
	remaining := int64(maxTemplateSource)
	for _, file := range files {
		content, readErr := readBoundedTemplate(source, file, &remaining)
		if readErr != nil {
			return nil, readErr
		}
		fileNames, inspectErr := inspectTemplate(file, content, funcs)
		if inspectErr != nil {
			return nil, inspectErr
		}
		for _, name := range fileNames {
			if prior, duplicate := names[name]; duplicate {
				return nil, fmt.Errorf(
					"parse HTML templates: template %q is defined by both %q and %q",
					name,
					prior,
					file,
				)
			}
			names[name] = file
		}
		if _, parseErr := compiled.New(path.Base(file)).Parse(string(content)); parseErr != nil {
			return nil, fmt.Errorf("parse HTML template %q: %w", file, parseErr)
		}
	}
	available := make([]string, 0, len(names))
	for name := range names {
		available = append(available, name)
	}
	slices.Sort(available)
	if (options.ErrorTemplate == "") != (options.ErrorModel == nil) {
		return nil, errors.New(
			"parse HTML templates: error template and model must be configured together",
		)
	}
	if options.ErrorTemplate != "" {
		if _, exists := names[options.ErrorTemplate]; !exists {
			return nil, fmt.Errorf(
				"parse HTML templates: error template %q is not defined",
				options.ErrorTemplate,
			)
		}
	}
	return &Renderer{
		templates:  compiled,
		names:      available,
		nameSet:    namesToSet(available),
		maxOutput:  maxOutput,
		errorName:  options.ErrorTemplate,
		errorModel: options.ErrorModel,
	}, nil
}

// TemplateNames returns the sorted exact executable names discovered during
// parsing.
func (renderer *Renderer) TemplateNames() []string {
	if renderer == nil {
		return nil
	}
	return slices.Clone(renderer.names)
}

// Render executes name into a bounded private buffer, then writes one HTML
// response only after execution succeeds. Status must permit a response body.
func (renderer *Renderer) Render(
	ctx context.Context,
	writer http.ResponseWriter,
	name string,
	status int,
	data any,
) error {
	if renderer == nil || renderer.templates == nil {
		return errors.New("render HTML template: renderer is nil")
	}
	if ctx == nil {
		return errors.New("render HTML template: context is nil")
	}
	if nilInterface(writer) {
		return errors.New("render HTML template: response writer is nil")
	}
	if _, exists := renderer.nameSet[name]; !exists {
		return fmt.Errorf("render HTML template: template %q is not defined", name)
	}
	if status < http.StatusOK ||
		status > 599 ||
		status == http.StatusNoContent ||
		status == http.StatusNotModified {
		return fmt.Errorf(
			"render HTML template %q: status %d cannot carry an HTML body",
			name,
			status,
		)
	}
	if cause := context.Cause(ctx); cause != nil {
		return fmt.Errorf("render HTML template %q: %w", name, cause)
	}

	output := &boundedBuffer{
		cause: func() error {
			return context.Cause(ctx)
		},
		limit: renderer.maxOutput,
	}
	if err := renderer.templates.ExecuteTemplate(output, name, data); err != nil {
		return fmt.Errorf("render HTML template %q: %w", name, err)
	}
	if cause := context.Cause(ctx); cause != nil {
		return fmt.Errorf("render HTML template %q: %w", name, cause)
	}
	header := writer.Header()
	header.Set("Content-Type", "text/html; charset=utf-8")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Content-Length", strconv.Itoa(output.Len()))
	writer.WriteHeader(status)
	written, err := writer.Write(output.Bytes())
	if err != nil {
		return fmt.Errorf("write HTML template %q response: %w", name, err)
	}
	if written != output.Len() {
		return fmt.Errorf(
			"write HTML template %q response: wrote %d of %d bytes: %w",
			name,
			written,
			output.Len(),
			io.ErrShortWrite,
		)
	}
	return nil
}

type boundedBuffer struct {
	cause  func() error
	buffer bytes.Buffer
	limit  int
}

func (buffer *boundedBuffer) Write(value []byte) (int, error) {
	if cause := buffer.cause(); cause != nil {
		return 0, cause
	}
	if len(value) > buffer.limit-buffer.buffer.Len() {
		return 0, ErrOutputLimit
	}
	return buffer.buffer.Write(value)
}

func (buffer *boundedBuffer) Len() int {
	return buffer.buffer.Len()
}

func (buffer *boundedBuffer) Bytes() []byte {
	return buffer.buffer.Bytes()
}

func normalizeMaxOutput(value int) (int, error) {
	if value == 0 {
		return defaultMaxOutputBytes, nil
	}
	if value < 1 || value > maxOutputBytes {
		return 0, fmt.Errorf(
			"parse HTML templates: maximum output must be between 1 and %d bytes",
			maxOutputBytes,
		)
	}
	return value, nil
}

func resolveFiles(source fs.FS, patterns []string) ([]string, error) {
	if len(patterns) < 1 || len(patterns) > maxTemplatePatterns {
		return nil, fmt.Errorf(
			"parse HTML templates: patterns must contain between 1 and %d entries",
			maxTemplatePatterns,
		)
	}
	seen := make(map[string]struct{})
	for index, pattern := range patterns {
		if err := validatePattern(index, pattern); err != nil {
			return nil, err
		}
		matches, err := fs.Glob(source, pattern)
		if err != nil {
			return nil, fmt.Errorf(
				"parse HTML templates: pattern %d is invalid: %w",
				index,
				err,
			)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf(
				"parse HTML templates: pattern %q matched no files",
				pattern,
			)
		}
		for _, match := range matches {
			seen[match] = struct{}{}
		}
	}
	files := make([]string, 0, len(seen))
	for file := range seen {
		files = append(files, file)
	}
	slices.Sort(files)
	return files, nil
}

func validatePattern(index int, pattern string) error {
	if pattern == "" ||
		len(pattern) > maxTemplatePathBytes ||
		strings.TrimSpace(pattern) != pattern ||
		strings.Contains(pattern, "\\") ||
		strings.HasPrefix(pattern, "/") {
		return fmt.Errorf(
			"parse HTML templates: pattern %d must be a relative bounded slash path",
			index,
		)
	}
	if slices.Contains(strings.Split(pattern, "/"), "..") {
		return fmt.Errorf(
			"parse HTML templates: pattern %d must not traverse parent directories",
			index,
		)
	}
	return nil
}

func readBoundedTemplate(
	source fs.FS,
	name string,
	remaining *int64,
) ([]byte, error) {
	info, err := fs.Stat(source, name)
	if err != nil {
		return nil, fmt.Errorf("parse HTML template %q: stat: %w", name, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("parse HTML template %q: source is not a regular file", name)
	}
	if info.Size() < 0 || info.Size() > *remaining {
		return nil, fmt.Errorf(
			"parse HTML template %q: aggregate source exceeds %d bytes",
			name,
			maxTemplateSource,
		)
	}
	file, err := source.Open(name)
	if err != nil {
		return nil, fmt.Errorf("parse HTML template %q: open: %w", name, err)
	}
	content, readErr := io.ReadAll(io.LimitReader(file, *remaining+1))
	closeErr := file.Close()
	if readErr != nil {
		return nil, fmt.Errorf("parse HTML template %q: read: %w", name, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("parse HTML template %q: close: %w", name, closeErr)
	}
	if int64(len(content)) > *remaining {
		return nil, fmt.Errorf(
			"parse HTML template %q: aggregate source exceeds %d bytes",
			name,
			maxTemplateSource,
		)
	}
	*remaining -= int64(len(content))
	return content, nil
}

func inspectTemplate(
	file string,
	content []byte,
	functions template.FuncMap,
) ([]string, error) {
	parsed, err := template.New(path.Base(file)).
		Option("missingkey=error").
		Funcs(functions).
		Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse HTML template %q: %w", file, err)
	}
	names := make([]string, 0, len(parsed.Templates()))
	for _, candidate := range parsed.Templates() {
		if candidate.Tree != nil {
			names = append(names, candidate.Name())
		}
	}
	slices.Sort(names)
	return names, nil
}

func validateFunctions(functions template.FuncMap) (template.FuncMap, error) {
	names := make([]string, 0, len(functions))
	for name := range functions {
		names = append(names, name)
	}
	slices.Sort(names)
	validated := make(template.FuncMap, len(functions))
	for _, name := range names {
		function := functions[name]
		if !validFunctionName(name) {
			return nil, fmt.Errorf(
				"parse HTML templates: function name %q is invalid",
				name,
			)
		}
		if function == nil {
			return nil, fmt.Errorf("parse HTML templates: function %q is nil", name)
		}
		reflected := reflect.ValueOf(function)
		if reflected.Kind() != reflect.Func || reflected.IsNil() {
			return nil, fmt.Errorf(
				"parse HTML templates: function %q must be a non-nil function",
				name,
			)
		}
		if !validFunctionSignature(reflected.Type()) {
			return nil, fmt.Errorf(
				"parse HTML templates: function %q must return one value or a value and error",
				name,
			)
		}
		validated[name] = function
	}
	return validated, nil
}

func validFunctionSignature(function reflect.Type) bool {
	switch function.NumOut() {
	case 1:
		return true
	case 2:
		return function.Out(1).Implements(reflect.TypeFor[error]())
	default:
		return false
	}
}

func validFunctionName(value string) bool {
	if len(value) < 1 || len(value) > maxFunctionNameBytes {
		return false
	}
	for index, character := range []byte(value) {
		letter := (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			character == '_'
		if !letter && (index == 0 || character < '0' || character > '9') {
			return false
		}
	}
	return true
}

func namesToSet(names []string) map[string]struct{} {
	result := make(map[string]struct{}, len(names))
	for _, name := range names {
		result[name] = struct{}{}
	}
	return result
}

func nilInterface(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	kind := reflected.Kind()
	nilCapable := kind == reflect.Chan ||
		kind == reflect.Func ||
		kind == reflect.Interface ||
		kind == reflect.Map ||
		kind == reflect.Pointer ||
		kind == reflect.Slice
	return nilCapable && reflected.IsNil()
}
