# Dependency review: x/oauth2

- Decision: approved for the isolated `starter/oauth2client` package.
- Version: `golang.org/x/oauth2` v0.36.0.
- Upstream: <https://go.googlesource.com/oauth2>.
- License: BSD-3-Clause; retained in the vendored module license.
- Maintenance: maintained by the Go project and already present through the
  approved OIDC dependency; this slice makes it a direct dependency.
- Security: Spice requires HTTPS, explicit authentication style, bounded
  provider parameters and responses, Bearer tokens, and safe non-unwrapping
  token errors. The complete graph remains subject to gosec and govulncheck.
- Cancellation: a caller-owned application-lifetime context controls token
  acquisition; individual resource requests retain their own contexts.
- Observability: caller-owned HTTP transports are the instrumentation seam.
  Spice errors never contain tokens, credentials, provider bodies, or token
  endpoint URLs.
- Configuration: credentials and scopes are explicit constructor inputs.
  There is no environment lookup, global client, discovery, or network I/O
  during construction.

Primary references:

- <https://pkg.go.dev/golang.org/x/oauth2/clientcredentials>
- <https://www.rfc-editor.org/rfc/rfc6749#section-4.4>
