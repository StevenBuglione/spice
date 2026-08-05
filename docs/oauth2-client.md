# OAuth2 service clients

The independently versioned
[`github.com/spice-framework/starter-oauth2client`](https://github.com/spice-framework/starter-oauth2client)
module adds RFC 6749 client-credentials tokens to outbound HTTP requests. It is
intended for a service acting on its own behalf, not delegated end-user
authorization.

```text
go get github.com/spice-framework/starter-oauth2client@latest
```

```go
client, err := oauth2client.NewClient(
    applicationContext,
    oauth2client.Options{
        ClientID:     configuration.ClientID,
        ClientSecret: configuration.ClientSecret,
        TokenURL:     "https://identity.example.com/oauth/token",
        Scopes:       []string{"inventory.read"},
    },
    &http.Client{Timeout: 5 * time.Second},  // token endpoint
    &http.Client{Timeout: 15 * time.Second}, // resource API
)
```

Construction makes no network request. The application-lifetime context owns
token refresh, the token source caches valid tokens safely across concurrent
requests, and request contexts still cancel individual resource calls.

The token endpoint must use HTTPS. Token and resource clients are explicit and
must have positive timeouts. The starter disables token-endpoint redirects,
bounds token responses to 64 KiB by default, requires Bearer tokens, rejects
header control characters, and returns a safe `TokenError` without retaining
provider response bodies, credentials, tokens, or endpoint URLs. Cancellation
and deadline errors remain discoverable with `errors.Is`.

HTTP Basic client authentication is the default. Use
`AuthStyleParameters` only when a provider explicitly requires body
credentials. Provider-specific endpoint parameters such as `audience` are
copied and bounded; standard OAuth2 fields cannot be overridden.

The constructor clones both clients but shares their transports. Fully
configure the clients and transports before construction and do not mutate
them concurrently. Standard HTTP/OpenTelemetry transports remain the
observation seam for token and resource requests.

The starter repository owns the canonical [dependency
review](https://github.com/spice-framework/starter-oauth2client/blob/main/docs/dependency-review.md),
[support policy](https://github.com/spice-framework/starter-oauth2client/blob/main/docs/support.md),
compatibility manifest, and verification evidence. This core document remains
an ecosystem usage guide; select and version the integration through ordinary
Go modules.
