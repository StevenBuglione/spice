# OIDC JWT resource server

`starter/oidc` is the opt-in authentication boundary for RFC 9068 JWT access
tokens. It requires the explicit `at+jwt` type, then uses `go-oidc` to verify
the signature, exact HTTPS issuer, audience, and expiry before constructing an
immutable `security.Principal`. An OIDC ID token is therefore never accepted as
an API access token.

Applications explicitly supply a trusted key set:

```go
server, err := oidc.NewResourceServer(keySet, oidc.Options{
    Issuer:   "https://issuer.example",
    Audience: "orders-api",
})
authentication, err := server.Middleware(reportWriteFailure)
```

Generated routes place `authentication` before `security.Guard`. The default
claims are `roles` and `scope`; both are configurable. Role and scope arrays are
supported, and an OAuth-style space-delimited scope string is supported.

Bearer parsing is strict: exactly one authorization value is accepted, token
whitespace is rejected, and the default encoded-token limit is 16 KiB. Invalid
credentials produce a safe RFC 9457 HTTP 401 plus the RFC 6750 `invalid_token`
challenge. Errors and observations never contain tokens, subjects, verifier
errors, roles, or scopes.

This slice supports signed JWT access tokens. It does not treat an ID token as
an access token, introspect opaque tokens, perform discovery, fetch keys during
construction, install global state, or read environment configuration. OIDC
discovery and refresh are a separate explicit starter contract.
