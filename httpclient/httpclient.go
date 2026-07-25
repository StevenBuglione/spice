// Package httpclient provides base-URL-scoped, context-owned outbound HTTP
// clients with bounded typed JSON helpers.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/StevenBuglione/spice/web"
)

const (
	// DefaultTimeout bounds requests when a caller does not supply an HTTP
	// client.
	DefaultTimeout = 30 * time.Second
	// DefaultMaxResponseBodyBytes bounds typed response and remote error bodies.
	DefaultMaxResponseBodyBytes int64 = 4 << 20
	// MaxResponseBodyBytes is the largest typed response bound accepted by New.
	// Larger payloads should use the raw streaming API.
	MaxResponseBodyBytes int64 = 64 << 20
)

// Options configures one isolated outbound client.
type Options struct {
	BaseURL                       string
	HTTPClient                    *http.Client
	DefaultHeaders                http.Header
	UserAgent                     string
	MaxResponseBodyBytes          int64
	DisallowUnknownResponseFields bool
}

// Client is one immutable base-URL-scoped outbound HTTP client.
type Client struct {
	baseURL                       *url.URL
	httpClient                    *http.Client
	defaultHeaders                http.Header
	maxResponseBodyBytes          int64
	disallowUnknownResponseFields bool
}

// New validates and copies an outbound client configuration. A nil HTTPClient
// selects a standard client with DefaultTimeout.
func New(options Options) (*Client, error) {
	baseURL, err := parseBaseURL(options.BaseURL)
	if err != nil {
		return nil, err
	}
	maxResponseBodyBytes := options.MaxResponseBodyBytes
	if maxResponseBodyBytes == 0 {
		maxResponseBodyBytes = DefaultMaxResponseBodyBytes
	}
	if maxResponseBodyBytes < 0 || maxResponseBodyBytes > MaxResponseBodyBytes {
		return nil, fmt.Errorf(
			"construct HTTP client: maximum response body bytes must be between 1 and %d",
			MaxResponseBodyBytes,
		)
	}
	defaultHeaders, err := cloneHeaders(options.DefaultHeaders)
	if err != nil {
		return nil, err
	}
	if options.UserAgent != "" {
		if strings.ContainsAny(options.UserAgent, "\r\n") {
			return nil, errors.New("construct HTTP client: user agent contains a newline")
		}
		defaultHeaders.Set("User-Agent", options.UserAgent)
	}
	standardClient := options.HTTPClient
	if standardClient == nil {
		standardClient = &http.Client{Timeout: DefaultTimeout}
	}
	copiedClient := *standardClient
	originalRedirect := copiedClient.CheckRedirect
	copiedClient.CheckRedirect = redirectPolicy(baseURL, originalRedirect)
	return &Client{
		baseURL:                       baseURL,
		httpClient:                    &copiedClient,
		defaultHeaders:                defaultHeaders,
		maxResponseBodyBytes:          maxResponseBodyBytes,
		disallowUnknownResponseFields: options.DisallowUnknownResponseFields,
	}, nil
}

// BaseURL returns the canonical configured base URL.
func (client *Client) BaseURL() string {
	if client == nil || client.baseURL == nil {
		return ""
	}
	return client.baseURL.String()
}

// NewRequest creates a scoped request without sending it. Reference must be
// relative to the configured base path and contain no dot segments or fragment.
func (client *Client) NewRequest(
	ctx context.Context,
	method string,
	reference string,
	body io.Reader,
) (*http.Request, error) {
	if client == nil || client.baseURL == nil {
		return nil, errors.New("create HTTP request: client is nil")
	}
	if ctx == nil {
		return nil, errors.New("create HTTP request: context is nil")
	}
	if method == "" || method != strings.ToUpper(method) {
		return nil, errors.New("create HTTP request: method must be uppercase")
	}
	target, err := client.resolve(reference)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}
	request.Header = client.defaultHeaders.Clone()
	return request, nil
}

// Do sends one raw request and returns a caller-owned response body. The
// request URL must remain within the configured base URL.
func (client *Client) Do(request *http.Request) (*http.Response, error) {
	if client == nil || client.baseURL == nil || client.httpClient == nil {
		return nil, errors.New("send HTTP request: client is nil")
	}
	if request == nil || request.URL == nil {
		return nil, errors.New("send HTTP request: request is nil")
	}
	if !withinBaseURL(client.baseURL, request.URL) {
		return nil, errors.New("send HTTP request: URL is outside configured base")
	}
	if request.Host != "" && !strings.EqualFold(request.Host, request.URL.Host) {
		return nil, errors.New("send HTTP request: Host override does not match configured base")
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		if urlErr, found := errors.AsType[*url.Error](err); found {
			err = urlErr.Err
		}
		return nil, fmt.Errorf("send HTTP request to configured base: %w", err)
	}
	return response, nil
}

// JSONResponse is one successful typed outbound response.
type JSONResponse[Value any] struct {
	Status int
	Header http.Header
	Value  Value
}

// ResponseError reports a non-2xx remote response without retaining its raw
// body or rendering remote details in Error().
type ResponseError struct {
	Status  int
	problem *web.Problem
}

// Error returns stable status context without remote response detail.
func (responseErr *ResponseError) Error() string {
	if responseErr == nil {
		return "outbound HTTP response failed"
	}
	statusText := http.StatusText(responseErr.Status)
	if statusText == "" {
		return "outbound HTTP response status " + strconv.Itoa(responseErr.Status)
	}
	return fmt.Sprintf("outbound HTTP response: %d %s", responseErr.Status, statusText)
}

// RemoteProblem returns a validated copy of a remote RFC 9457 response.
func (responseErr *ResponseError) RemoteProblem() (web.Problem, bool) {
	if responseErr == nil || responseErr.problem == nil {
		return web.Problem{}, false
	}
	return *responseErr.problem, true
}

// DoJSON sends an optional JSON body and decodes one bounded JSON response.
// Empty and 204 success bodies return the zero Value.
func DoJSON[Value any](
	ctx context.Context,
	client *Client,
	method string,
	reference string,
	requestBody any,
) (JSONResponse[Value], error) {
	if client == nil {
		return JSONResponse[Value]{}, errors.New("send JSON request: client is nil")
	}
	var body io.Reader
	if requestBody != nil {
		content, err := json.Marshal(requestBody)
		if err != nil {
			return JSONResponse[Value]{}, fmt.Errorf("encode JSON request: %w", err)
		}
		body = bytes.NewReader(content)
	}
	request, err := client.NewRequest(ctx, method, reference, body)
	if err != nil {
		return JSONResponse[Value]{}, err
	}
	request.Header.Set("Accept", "application/json")
	if requestBody != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request) //nolint:bodyclose // readAndCloseResponse closes the body and preserves its error.
	if err != nil {
		return JSONResponse[Value]{}, err
	}
	content, tooLarge, err := readAndCloseResponse(
		response.Body,
		client.maxResponseBodyBytes,
	)
	if err != nil {
		return JSONResponse[Value]{}, err
	}
	return decodeJSONHTTPResponse[Value](client, response, content, tooLarge)
}

func decodeJSONHTTPResponse[Value any](
	client *Client,
	response *http.Response,
	content []byte,
	tooLarge bool,
) (JSONResponse[Value], error) {
	result := JSONResponse[Value]{
		Status: response.StatusCode,
		Header: response.Header.Clone(),
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseErr := &ResponseError{Status: response.StatusCode}
		if !tooLarge {
			responseErr.problem = decodeProblem(
				response.StatusCode,
				response.Header.Get("Content-Type"),
				content,
			)
		}
		return result, responseErr
	}
	if tooLarge {
		return result, fmt.Errorf(
			"decode HTTP response: body exceeds %d bytes",
			client.maxResponseBodyBytes,
		)
	}
	if response.StatusCode == http.StatusNoContent || len(bytes.TrimSpace(content)) == 0 {
		return result, nil
	}
	if !isJSONMediaType(response.Header.Get("Content-Type")) {
		return result, errors.New("decode HTTP response: content type is not JSON")
	}
	if err := decodeJSONResponse(content, &result.Value, client.disallowUnknownResponseFields); err != nil {
		return result, err
	}
	return result, nil
}

func (client *Client) resolve(reference string) (*url.URL, error) {
	parsed, err := url.Parse(reference)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: parse reference: %w", err)
	}
	if parsed.IsAbs() || parsed.Host != "" || parsed.User != nil ||
		strings.HasPrefix(reference, "//") {
		return nil, errors.New("create HTTP request: reference must be relative")
	}
	if parsed.Fragment != "" {
		return nil, errors.New("create HTTP request: reference must not contain a fragment")
	}
	if strings.HasPrefix(parsed.Path, "/") ||
		strings.Contains(parsed.Path, "\\") ||
		hasDotSegment(parsed.Path) {
		return nil, errors.New("create HTTP request: reference must stay below the base path")
	}
	target := client.baseURL.ResolveReference(parsed)
	if !withinBaseURL(client.baseURL, target) {
		return nil, errors.New("create HTTP request: resolved URL is outside the configured base")
	}
	return target, nil
}

func parseBaseURL(value string) (*url.URL, error) {
	if value == "" {
		return nil, errors.New("construct HTTP client: base URL is empty")
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("construct HTTP client: parse base URL: %w", err)
	}
	if err := validateBaseURL(parsed); err != nil {
		return nil, err
	}
	if !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
		if parsed.RawPath != "" {
			parsed.RawPath += "/"
		}
	}
	return parsed, nil
}

func validateBaseURL(parsed *url.URL) error {
	if (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" || parsed.Opaque != "" {
		return errors.New("construct HTTP client: base URL must be an absolute HTTP(S) URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return errors.New("construct HTTP client: base URL must not contain credentials, query, or fragment")
	}
	if strings.Contains(parsed.Path, "\\") ||
		containsEncodedPathSeparator(parsed.RawPath) ||
		hasDotSegment(parsed.Path) {
		return errors.New("construct HTTP client: base URL must not contain backslashes or dot segments")
	}
	return nil
}

func cloneHeaders(headers http.Header) (http.Header, error) {
	result := make(http.Header, len(headers))
	for name, values := range headers {
		if !validHeaderName(name) {
			return nil, fmt.Errorf("construct HTTP client: invalid default header name %q", name)
		}
		if strings.EqualFold(name, "Host") {
			return nil, errors.New("construct HTTP client: Host must come from the base URL")
		}
		for _, value := range values {
			if strings.ContainsAny(value, "\r\n") {
				return nil, fmt.Errorf("construct HTTP client: default header %q contains a newline", name)
			}
		}
		canonicalName := http.CanonicalHeaderKey(name)
		result[canonicalName] = append(result[canonicalName], values...)
	}
	return result, nil
}

func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for index := range len(name) {
		character := name[index]
		if character <= ' ' || character >= 0x7f ||
			strings.ContainsRune("()<>@,;:\\\"/[]?={}", rune(character)) {
			return false
		}
	}
	return true
}

func redirectPolicy(
	baseURL *url.URL,
	original func(*http.Request, []*http.Request) error,
) func(*http.Request, []*http.Request) error {
	return func(request *http.Request, via []*http.Request) error {
		if request == nil || request.URL == nil || !withinBaseURL(baseURL, request.URL) {
			return fmt.Errorf("redirect URL is outside base URL %q", baseURL)
		}
		if original != nil {
			return original(request, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
}

func withinBaseURL(baseURL, target *url.URL) bool {
	if baseURL == nil || target == nil ||
		!strings.EqualFold(baseURL.Scheme, target.Scheme) ||
		!strings.EqualFold(baseURL.Host, target.Host) ||
		target.User != nil ||
		hasDotSegment(target.Path) {
		return false
	}
	basePath := baseURL.Path
	targetPath := target.Path
	return targetPath == strings.TrimSuffix(basePath, "/") ||
		strings.HasPrefix(targetPath, basePath)
}

func hasDotSegment(value string) bool {
	for segment := range strings.SplitSeq(value, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}

func containsEncodedPathSeparator(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c")
}

func readResponse(body io.Reader, maximum int64) ([]byte, bool, error) {
	if body == nil {
		return nil, false, errors.New("response body is nil")
	}
	content, err := io.ReadAll(io.LimitReader(body, maximum+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(content)) > maximum {
		return content[:maximum], true, nil
	}
	return content, false, nil
}

func readAndCloseResponse(
	body io.ReadCloser,
	maximum int64,
) ([]byte, bool, error) {
	if body == nil {
		return nil, false, errors.New("read HTTP response: body is nil")
	}
	content, tooLarge, readErr := readResponse(body, maximum)
	closeErr := body.Close()
	if readErr != nil || closeErr != nil {
		return nil, false, fmt.Errorf(
			"read HTTP response: %w",
			errors.Join(readErr, closeErr),
		)
	}
	return content, tooLarge, nil
}

func decodeProblem(status int, contentType string, content []byte) *web.Problem {
	if !isJSONMediaType(contentType) || len(bytes.TrimSpace(content)) == 0 {
		return nil
	}
	var problem web.Problem
	if err := json.Unmarshal(content, &problem); err != nil ||
		problem.Validate() != nil ||
		problem.Status != status {
		return nil
	}
	return &problem
}

func decodeJSONResponse(content []byte, destination any, disallowUnknown bool) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	if disallowUnknown {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode HTTP response JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decode HTTP response JSON: body contains multiple values")
		}
		return fmt.Errorf("decode HTTP response JSON: trailing data: %w", err)
	}
	return nil
}

func isJSONMediaType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return false
	}
	return mediaType == "application/json" ||
		strings.HasPrefix(mediaType, "application/") && strings.HasSuffix(mediaType, "+json")
}
