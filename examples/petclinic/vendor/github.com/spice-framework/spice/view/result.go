package view

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type resultKind uint8

const (
	resultInvalid resultKind = iota
	resultRender
	resultRedirect
)

// Result is one validated server-rendered controller outcome.
type Result struct {
	kind     resultKind
	status   int
	template string
	location string
	model    any
}

// Render constructs an HTTP 200 template result.
func Render[T any](name string, model T) (Result, error) {
	return RenderStatus(http.StatusOK, name, model)
}

// RenderStatus constructs a template result with an explicit body-capable
// success status.
func RenderStatus[T any](
	status int,
	name string,
	model T,
) (Result, error) {
	result := Result{
		kind:     resultRender,
		status:   status,
		template: name,
		model:    model,
	}
	if err := result.Validate(); err != nil {
		return Result{}, err
	}
	return result, nil
}

// SeeOther constructs a safe local HTTP 303 redirect.
func SeeOther(location string) (Result, error) {
	result := Result{
		kind:     resultRedirect,
		status:   http.StatusSeeOther,
		location: location,
	}
	if err := result.Validate(); err != nil {
		return Result{}, err
	}
	return result, nil
}

// AcceptsHTML reports whether an Accept header permits an HTML representation.
// An empty header accepts HTML.
func AcceptsHTML(header string) bool {
	if strings.TrimSpace(header) == "" {
		return true
	}
	for part := range strings.SplitSeq(header, ",") {
		mediaType, parameters, err := mime.ParseMediaType(
			strings.TrimSpace(part),
		)
		if err != nil || qualityRejected(parameters["q"]) {
			continue
		}
		if mediaType == "*/*" ||
			mediaType == "text/*" ||
			mediaType == "text/html" ||
			mediaType == "application/xhtml+xml" {
			return true
		}
	}
	return false
}

// Validate checks the closed result contract.
func (result Result) Validate() error {
	switch result.kind {
	case resultInvalid:
		return errors.New("validate view result: result is uninitialized")
	case resultRender:
		if result.template == "" ||
			strings.TrimSpace(result.template) != result.template {
			return errors.New(
				"validate view result: template name must be non-empty and trimmed",
			)
		}
		if result.status < http.StatusOK ||
			result.status >= http.StatusMultipleChoices ||
			result.status == http.StatusNoContent {
			return fmt.Errorf(
				"validate view result: status %d cannot render an HTML body",
				result.status,
			)
		}
	case resultRedirect:
		if result.status != http.StatusSeeOther {
			return errors.New(
				"validate view result: redirect status must be 303",
			)
		}
		if err := validateLocalLocation(result.location); err != nil {
			return err
		}
	}
	return nil
}

// Respond validates and writes a view result atomically where possible.
func (renderer *Renderer) Respond(
	ctx context.Context,
	writer http.ResponseWriter,
	result Result,
) error {
	if err := result.Validate(); err != nil {
		return err
	}
	switch result.kind {
	case resultInvalid:
		return errors.New("respond with view result: unsupported result")
	case resultRender:
		return renderer.Render(
			ctx,
			writer,
			result.template,
			result.status,
			result.model,
		)
	case resultRedirect:
		if ctx == nil {
			return errors.New("respond with redirect: context is nil")
		}
		if cause := context.Cause(ctx); cause != nil {
			return fmt.Errorf("respond with redirect: %w", cause)
		}
		if writer == nil {
			return errors.New(
				"respond with redirect: response writer is nil",
			)
		}
		writer.Header().Set("Location", result.location)
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Del("Content-Type")
		writer.WriteHeader(result.status)
		return nil
	}
	return errors.New("respond with view result: unsupported result")
}

func validateLocalLocation(location string) error {
	if location == "" ||
		strings.TrimSpace(location) != location ||
		!strings.HasPrefix(location, "/") ||
		strings.HasPrefix(location, "//") ||
		strings.ContainsAny(location, "\x00\r\n#") {
		return errors.New(
			"validate view result: redirect location must be a safe local absolute path",
		)
	}
	parsed, err := url.ParseRequestURI(location)
	if err != nil ||
		parsed.IsAbs() ||
		parsed.Host != "" ||
		parsed.Fragment != "" {
		return errors.New(
			"validate view result: redirect location must be a safe local absolute path",
		)
	}
	return nil
}

func qualityRejected(value string) bool {
	if value == "" {
		return false
	}
	quality, err := strconv.ParseFloat(value, 64)
	return err != nil || quality <= 0 || quality > 1
}
