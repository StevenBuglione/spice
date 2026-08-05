package oauth2client_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spice-framework/spice/starter/oauth2client"
)

func TestClientCredentialsAuthorizesAndCachesResourceRequests(t *testing.T) {
	t.Parallel()

	var tokenRequests atomic.Int64
	tokenServer := httptest.NewTLSServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		tokenRequests.Add(1)
		if clientID, secret, ok := request.BasicAuth(); !ok ||
			clientID != "service" ||
			secret != "credential" {
			http.Error(writer, "invalid client", http.StatusUnauthorized)
			return
		}
		if err := request.ParseForm(); err != nil {
			http.Error(writer, "invalid form", http.StatusBadRequest)
			return
		}
		if request.Form.Get("grant_type") != "client_credentials" ||
			request.Form.Get("scope") != "orders.read orders.write" ||
			request.Form.Get("audience") != "orders-api" {
			http.Error(writer, "invalid parameters", http.StatusBadRequest)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		writeString(
			t,
			writer,
			`{"access_token":"service-token","token_type":"Bearer","expires_in":3600}`,
		)
	}))
	t.Cleanup(tokenServer.Close)

	resourceServer := httptest.NewTLSServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if request.Header.Get("Authorization") != "Bearer service-token" {
			http.Error(writer, "missing token", http.StatusUnauthorized)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(resourceServer.Close)

	tokenClient := tokenServer.Client()
	tokenClient.Timeout = time.Second
	resourceClient := resourceServer.Client()
	resourceClient.Timeout = time.Second
	client, err := oauth2client.NewClient(
		context.Background(),
		validOptions(tokenServer.URL),
		tokenClient,
		resourceClient,
	)
	if err != nil {
		t.Fatalf("construct client: %v", err)
	}
	if tokenRequests.Load() != 0 {
		t.Fatal("construction performed network I/O")
	}

	for range 2 {
		response, requestErr := client.Get(resourceServer.URL)
		if requestErr != nil {
			t.Fatalf("request resource: %v", requestErr)
		}
		if response.StatusCode != http.StatusNoContent {
			t.Fatalf("unexpected status: %d", response.StatusCode)
		}
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Fatalf("close response: %v", closeErr)
		}
	}
	if tokenRequests.Load() != 1 {
		t.Fatalf("expected one cached token request, got %d", tokenRequests.Load())
	}
}

func TestParameterAuthenticationIsExplicit(t *testing.T) {
	t.Parallel()

	tokenServer := httptest.NewTLSServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if request.Header.Get("Authorization") != "" {
			http.Error(writer, "unexpected authorization header", http.StatusBadRequest)
			return
		}
		if err := request.ParseForm(); err != nil {
			http.Error(writer, "invalid form", http.StatusBadRequest)
			return
		}
		if request.Form.Get("client_id") != "service" ||
			request.Form.Get("client_secret") != "credential" {
			http.Error(writer, "missing credentials", http.StatusUnauthorized)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		writeString(t, writer, `{"access_token":"token","token_type":"bearer"}`)
	}))
	t.Cleanup(tokenServer.Close)

	resourceServer := newAuthorizedServer(t)
	options := validOptions(tokenServer.URL)
	options.AuthStyle = oauth2client.AuthStyleParameters
	client := newClient(t, options, tokenServer.Client(), resourceServer.Client())

	response, err := client.Get(resourceServer.URL)
	if err != nil {
		t.Fatalf("request resource: %v", err)
	}
	if closeErr := response.Body.Close(); closeErr != nil {
		t.Fatalf("close response: %v", closeErr)
	}
}

func TestTokenFailuresAreBoundedAndSafe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler http.Handler
		limit   int64
	}{
		{
			name: "upstream body",
			handler: http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				http.Error(writer, "provider-secret-detail", http.StatusBadRequest)
			}),
		},
		{
			name: "oversized content length",
			handler: http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Length", "1000")
				writeString(t, writer, strings.Repeat("x", 1000))
			}),
			limit: 32,
		},
		{
			name: "oversized streamed body",
			handler: http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				flusher, ok := writer.(http.Flusher)
				if !ok {
					t.Error("response writer does not support flushing")
					return
				}
				flusher.Flush()
				writeString(t, writer, strings.Repeat("x", 1000))
			}),
			limit: 32,
		},
		{
			name: "invalid token type",
			handler: http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				writeString(t, writer, `{"access_token":"secret-token","token_type":"MAC"}`)
			}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			tokenServer := httptest.NewTLSServer(test.handler)
			t.Cleanup(tokenServer.Close)
			resourceServer := newAuthorizedServer(t)
			options := validOptions(tokenServer.URL)
			options.MaxTokenResponseBytes = test.limit
			client := newClient(t, options, tokenServer.Client(), resourceServer.Client())

			response, err := client.Get(resourceServer.URL)
			closeResponse(t, response)
			tokenErr, ok := errors.AsType[*oauth2client.TokenError](err)
			if !ok || tokenErr == nil {
				t.Fatalf("expected safe token error, got %v", err)
			}
			message := err.Error()
			for _, secret := range []string{
				"provider-secret-detail",
				"secret-token",
				"credential",
				tokenServer.URL,
			} {
				if strings.Contains(message, secret) {
					t.Fatalf("failure exposed %q: %v", secret, err)
				}
			}
		})
	}
}

func TestTokenEndpointRedirectIsNotFollowed(t *testing.T) {
	t.Parallel()

	var destinationRequests atomic.Int64
	destination := httptest.NewTLSServer(http.HandlerFunc(func(
		http.ResponseWriter,
		*http.Request,
	) {
		destinationRequests.Add(1)
	}))
	t.Cleanup(destination.Close)
	redirect := httptest.NewTLSServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		http.Redirect(writer, request, destination.URL, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(redirect.Close)
	resourceServer := newAuthorizedServer(t)

	client := newClient(
		t,
		validOptions(redirect.URL),
		redirect.Client(),
		resourceServer.Client(),
	)
	response, err := client.Get(resourceServer.URL)
	closeResponse(t, response)
	if err == nil {
		t.Fatal("expected redirect rejection")
	}
	if destinationRequests.Load() != 0 {
		t.Fatal("token client followed a redirect")
	}
}

func TestLifetimeCancellationIsClassified(t *testing.T) {
	t.Parallel()

	tokenServer := httptest.NewTLSServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		http.Error(writer, "unexpected request", http.StatusInternalServerError)
	}))
	t.Cleanup(tokenServer.Close)
	resourceServer := newAuthorizedServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	client, err := oauth2client.NewClient(
		ctx,
		validOptions(tokenServer.URL),
		timedClient(tokenServer.Client()),
		timedClient(resourceServer.Client()),
	)
	if err != nil {
		t.Fatalf("construct client: %v", err)
	}
	cancel()

	response, err := client.Get(resourceServer.URL)
	closeResponse(t, response)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation classification, got %v", err)
	}
}

func TestNewClientValidatesConfiguration(t *testing.T) {
	t.Parallel()

	valid := validOptions("https://identity.example.test/token")
	goodClient := &http.Client{Timeout: time.Second}
	tests := []struct {
		name           string
		mutate         func(*oauth2client.Options)
		nilLifetime    bool
		tokenClient    *http.Client
		resourceClient *http.Client
	}{
		{name: "client ID", mutate: func(options *oauth2client.Options) { options.ClientID = "" }},
		{name: "trimmed client ID", mutate: func(options *oauth2client.Options) { options.ClientID = " x " }},
		{name: "secret", mutate: func(options *oauth2client.Options) { options.ClientSecret = "" }},
		{name: "HTTP URL", mutate: func(options *oauth2client.Options) { options.TokenURL = "http://id/token" }},
		{name: "URL query", mutate: func(options *oauth2client.Options) { options.TokenURL += "?secret=x" }},
		{name: "auth style", mutate: func(options *oauth2client.Options) { options.AuthStyle = 99 }},
		{name: "empty scope", mutate: func(options *oauth2client.Options) { options.Scopes = []string{""} }},
		{name: "invalid scope", mutate: func(options *oauth2client.Options) { options.Scopes = []string{"a b"} }},
		{name: "duplicate scope", mutate: func(options *oauth2client.Options) { options.Scopes = []string{"a", "a"} }},
		{
			name: "reserved parameter",
			mutate: func(options *oauth2client.Options) {
				options.EndpointParameters = url.Values{"grant_type": {"password"}}
			},
		},
		{
			name: "empty parameter values",
			mutate: func(options *oauth2client.Options) {
				options.EndpointParameters = url.Values{"audience": nil}
			},
		},
		{
			name: "response limit",
			mutate: func(options *oauth2client.Options) {
				options.MaxTokenResponseBytes = -1
			},
		},
		{name: "lifetime", nilLifetime: true},
		{name: "token client", tokenClient: nil},
		{name: "token timeout", tokenClient: &http.Client{}},
		{name: "resource client", resourceClient: nil},
		{name: "resource timeout", resourceClient: &http.Client{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			options := valid
			if test.mutate != nil {
				test.mutate(&options)
			}
			lifetime := context.Background()
			if test.nilLifetime {
				lifetime = nilTestContext()
			}
			tokenClient := goodClient
			if test.tokenClient != nil || test.name == "token client" {
				tokenClient = test.tokenClient
			}
			resourceClient := goodClient
			if test.resourceClient != nil || test.name == "resource client" {
				resourceClient = test.resourceClient
			}
			if _, err := oauth2client.NewClient(
				lifetime,
				options,
				tokenClient,
				resourceClient,
			); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func validOptions(tokenURL string) oauth2client.Options {
	return oauth2client.Options{
		ClientID:     "service",
		ClientSecret: "credential",
		TokenURL:     tokenURL,
		Scopes:       []string{"orders.write", "orders.read"},
		EndpointParameters: url.Values{
			"audience": {"orders-api"},
		},
	}
}

func newClient(
	t *testing.T,
	options oauth2client.Options,
	tokenClient *http.Client,
	resourceClient *http.Client,
) *http.Client {
	t.Helper()
	client, err := oauth2client.NewClient(
		context.Background(),
		options,
		timedClient(tokenClient),
		timedClient(resourceClient),
	)
	if err != nil {
		t.Fatalf("construct client: %v", err)
	}
	return client
}

func timedClient(client *http.Client) *http.Client {
	clone := *client
	clone.Timeout = time.Second
	return &clone
}

func newAuthorizedServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if request.Header.Get("Authorization") == "" {
			http.Error(writer, "missing token", http.StatusUnauthorized)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	return server
}

func nilTestContext() context.Context {
	return nil
}

func writeString(t *testing.T, writer io.Writer, value string) {
	t.Helper()
	if _, err := io.WriteString(writer, value); err != nil {
		t.Errorf("write response: %v", err)
	}
}

func closeResponse(t *testing.T, response *http.Response) {
	t.Helper()
	if response == nil {
		return
	}
	if err := response.Body.Close(); err != nil {
		t.Errorf("close response: %v", err)
	}
}

func ExampleNewClient() {
	options := oauth2client.Options{
		ClientID:     "orders-service",
		ClientSecret: "configured-secret",
		TokenURL:     "https://identity.example.test/oauth/token",
		Scopes:       []string{"inventory.read"},
	}
	tokenClient := &http.Client{Timeout: 5 * time.Second}
	resourceClient := &http.Client{Timeout: 10 * time.Second}
	client, err := oauth2client.NewClient(
		context.Background(),
		options,
		tokenClient,
		resourceClient,
	)
	fmt.Println(client != nil, err)
	// Output: true <nil>
}
