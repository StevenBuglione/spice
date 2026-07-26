// Package oauth2client provides a bounded OAuth 2.0 client-credentials
// integration for service-to-service HTTP clients.
package oauth2client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	defaultMaxTokenResponseBytes int64 = 64 << 10
	maxTokenResponseBytes        int64 = 1 << 20
	maxEndpointParameters              = 32
	maxEndpointValues                  = 32
	maxEndpointValueBytes              = 8 << 10
)

var errTokenResponseTooLarge = errors.New("OAuth2 token response is too large")

// AuthStyle selects how the client authenticates to the token endpoint.
type AuthStyle uint8

const (
	// AuthStyleHeader uses HTTP Basic authentication. It is the secure default
	// and avoids the upstream library's two-request auto-detection.
	AuthStyleHeader AuthStyle = iota
	// AuthStyleParameters sends client credentials in the form body for
	// providers that explicitly require it.
	AuthStyleParameters
)

// Options describes one client-credentials integration. EndpointParameters
// supports provider-specific values such as audience, but cannot replace
// standard OAuth2 parameters.
type Options struct {
	ClientID              string
	ClientSecret          string
	TokenURL              string
	Scopes                []string
	EndpointParameters    url.Values
	AuthStyle             AuthStyle
	MaxTokenResponseBytes int64
}

// TokenError is a safe token-resolution failure. It intentionally does not
// retain or unwrap the upstream response, URL, credentials, or token.
type TokenError struct {
	canceled bool
	timedOut bool
}

// Error returns a stable message safe for logs and HTTP error chains.
func (*TokenError) Error() string {
	return "OAuth2 token acquisition failed"
}

// Is preserves cancellation classification without exposing the upstream
// failure.
func (tokenErr *TokenError) Is(target error) bool {
	return (tokenErr.canceled && target == context.Canceled) ||
		(tokenErr.timedOut && target == context.DeadlineExceeded)
}

// NewClient constructs an HTTP client that resolves and caches service
// credentials. Construction performs no network I/O. tokenClient is used only
// for the token endpoint; resourceClient is cloned and used for API calls.
func NewClient(
	lifetime context.Context,
	options Options,
	tokenClient *http.Client,
	resourceClient *http.Client,
) (*http.Client, error) {
	normalized, err := normalize(options)
	if err != nil {
		return nil, err
	}
	switch {
	case lifetime == nil:
		return nil, errors.New("construct OAuth2 client: lifetime context is nil")
	case tokenClient == nil:
		return nil, errors.New("construct OAuth2 client: token HTTP client is nil")
	case tokenClient.Timeout <= 0:
		return nil, errors.New("construct OAuth2 client: token HTTP client timeout must be positive")
	case resourceClient == nil:
		return nil, errors.New("construct OAuth2 client: resource HTTP client is nil")
	case resourceClient.Timeout <= 0:
		return nil, errors.New("construct OAuth2 client: resource HTTP client timeout must be positive")
	}

	tokenHTTPClient := cloneTokenClient(tokenClient, normalized.MaxTokenResponseBytes)
	tokenContext := context.WithValue(lifetime, oauth2.HTTPClient, tokenHTTPClient)
	configuration := clientcredentials.Config{
		ClientID:       normalized.ClientID,
		ClientSecret:   normalized.ClientSecret,
		TokenURL:       normalized.TokenURL,
		Scopes:         normalized.Scopes,
		EndpointParams: normalized.EndpointParameters,
		AuthStyle:      oauthAuthStyle(normalized.AuthStyle),
	}
	source := &safeTokenSource{source: configuration.TokenSource(tokenContext)}

	client := *resourceClient
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	client.Transport = &oauth2.Transport{Source: source, Base: base}
	return &client, nil
}

func normalize(options Options) (Options, error) {
	switch {
	case options.ClientID == "" || strings.TrimSpace(options.ClientID) != options.ClientID:
		return Options{}, errors.New("construct OAuth2 client: client ID is required")
	case options.ClientSecret == "":
		return Options{}, errors.New("construct OAuth2 client: client secret is required")
	case !validTokenURL(options.TokenURL):
		return Options{}, errors.New("construct OAuth2 client: token URL must be an HTTPS URL")
	case options.AuthStyle != AuthStyleHeader && options.AuthStyle != AuthStyleParameters:
		return Options{}, errors.New("construct OAuth2 client: unsupported authentication style")
	}

	scopes, err := normalizeScopes(options.Scopes)
	if err != nil {
		return Options{}, err
	}
	parameters, err := normalizeParameters(options.EndpointParameters)
	if err != nil {
		return Options{}, err
	}
	maxBytes := options.MaxTokenResponseBytes
	if maxBytes == 0 {
		maxBytes = defaultMaxTokenResponseBytes
	}
	if maxBytes < 1 || maxBytes > maxTokenResponseBytes {
		return Options{}, fmt.Errorf(
			"construct OAuth2 client: token response limit must be between 1 and %d bytes",
			maxTokenResponseBytes,
		)
	}
	options.Scopes = scopes
	options.EndpointParameters = parameters
	options.MaxTokenResponseBytes = maxBytes
	return options, nil
}

func validTokenURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed == nil {
		return false
	}
	return parsed.Scheme == "https" &&
		parsed.Host != "" &&
		parsed.User == nil &&
		parsed.RawQuery == "" &&
		parsed.Fragment == ""
}

func normalizeScopes(scopes []string) ([]string, error) {
	normalized := append([]string(nil), scopes...)
	slices.Sort(normalized)
	for index, scope := range normalized {
		if !validScope(scope) {
			return nil, fmt.Errorf("construct OAuth2 client: scope %d is invalid", index)
		}
		if index > 0 && scope == normalized[index-1] {
			return nil, fmt.Errorf("construct OAuth2 client: scope %q is duplicated", scope)
		}
	}
	return normalized, nil
}

func validScope(scope string) bool {
	if scope == "" {
		return false
	}
	for _, character := range []byte(scope) {
		if character < 0x21 ||
			character == 0x22 ||
			character == 0x5c ||
			character > 0x7e {
			return false
		}
	}
	return true
}

func normalizeParameters(parameters url.Values) (url.Values, error) {
	if len(parameters) > maxEndpointParameters {
		return nil, errors.New("construct OAuth2 client: too many endpoint parameters")
	}
	normalized := make(url.Values, len(parameters))
	keys := make([]string, 0, len(parameters))
	for key := range parameters {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		values := parameters[key]
		if !validParameterName(key) {
			return nil, fmt.Errorf("construct OAuth2 client: endpoint parameter %q is invalid", key)
		}
		if len(values) == 0 || len(values) > maxEndpointValues {
			return nil, fmt.Errorf(
				"construct OAuth2 client: endpoint parameter %q has an invalid value count",
				key,
			)
		}
		normalized[key] = append([]string(nil), values...)
		for _, value := range normalized[key] {
			if len(value) > maxEndpointValueBytes {
				return nil, fmt.Errorf(
					"construct OAuth2 client: endpoint parameter %q value is too large",
					key,
				)
			}
		}
	}
	return normalized, nil
}

func validParameterName(name string) bool {
	switch name {
	case "", "client_id", "client_secret", "grant_type", "scope":
		return false
	}
	for _, character := range []byte(name) {
		if character <= 0x20 || character >= 0x7f || character == '&' || character == '=' {
			return false
		}
	}
	return true
}

func oauthAuthStyle(style AuthStyle) oauth2.AuthStyle {
	if style == AuthStyleParameters {
		return oauth2.AuthStyleInParams
	}
	return oauth2.AuthStyleInHeader
}

func cloneTokenClient(client *http.Client, maxBytes int64) *http.Client {
	clone := *client
	base := clone.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	clone.Transport = boundedTokenTransport{base: base, maxBytes: maxBytes}
	clone.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &clone
}

type safeTokenSource struct {
	source oauth2.TokenSource
}

func (source *safeTokenSource) Token() (*oauth2.Token, error) {
	token, err := source.source.Token()
	if err != nil {
		return nil, safeTokenError(err)
	}
	if token == nil ||
		!validBearerToken(token.AccessToken) ||
		(token.TokenType != "" && !strings.EqualFold(token.TokenType, "Bearer")) ||
		!token.Valid() {
		return nil, &TokenError{}
	}
	return token, nil
}

func validBearerToken(token string) bool {
	if token == "" {
		return false
	}
	padding := false
	for _, character := range []byte(token) {
		if character == '=' {
			padding = true
			continue
		}
		if padding || !isBearerCharacter(character) {
			return false
		}
	}
	return true
}

func isBearerCharacter(character byte) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9' ||
		strings.ContainsRune("-._~+/", rune(character))
}

func safeTokenError(err error) error {
	return &TokenError{
		canceled: errors.Is(err, context.Canceled),
		timedOut: errors.Is(err, context.DeadlineExceeded),
	}
}

type boundedTokenTransport struct {
	base     http.RoundTripper
	maxBytes int64
}

func (transport boundedTokenTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := transport.base.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	if response.ContentLength > transport.maxBytes {
		if closeErr := response.Body.Close(); closeErr != nil {
			return nil, errors.Join(errTokenResponseTooLarge, closeErr)
		}
		return nil, errTokenResponseTooLarge
	}
	response.Body = &boundedBody{body: response.Body, remaining: transport.maxBytes}
	return response, nil
}

type boundedBody struct {
	body      io.ReadCloser
	remaining int64
}

func (body *boundedBody) Read(buffer []byte) (int, error) {
	if body.remaining == 0 {
		var probe [1]byte
		count, err := body.body.Read(probe[:])
		if count > 0 {
			return 0, errTokenResponseTooLarge
		}
		return 0, err
	}
	if int64(len(buffer)) > body.remaining+1 {
		buffer = buffer[:body.remaining+1]
	}
	count, err := body.body.Read(buffer)
	if int64(count) > body.remaining {
		allowed := int(body.remaining)
		body.remaining = 0
		return allowed, errTokenResponseTooLarge
	}
	body.remaining -= int64(count)
	return count, err
}

func (body *boundedBody) Close() error {
	return body.body.Close()
}
