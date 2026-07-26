# Encrypted cookie sessions

`session` provides typed, stateless HTTP sessions with no global registry,
background worker, process-environment read, or server-side storage:

```go
type Claims struct {
    Subject string   `json:"subject"`
    Roles   []string `json:"roles"`
}

sessions, err := session.New[Claims](session.Options{
    Name: "__Host-orders",
    Keys: []session.Key{
        {
            ID:     "2026-07",
            Secret: configuration.SessionKey,
        },
        {
            ID:     "2026-06",
            Secret: configuration.PreviousSessionKey,
        },
    },
})
if err != nil {
    return err
}
```

Keys are exactly 32 bytes for AES-256-GCM. Load them from a secret source and
never commit them. The first key seals new cookies; at most seven older keys
may remain for bounded decryption during rotation. A record loaded through an
older key reports `NeedsRefresh`, so the application can save it again with
the primary key before retiring the old key.

```go
record, err := sessions.Load(request, clock())
switch {
case errors.Is(err, session.ErrNotFound):
    // Continue anonymously.
case errors.Is(err, session.ErrExpired):
    // Ask the client to authenticate again.
case err != nil:
    // Treat malformed or unauthenticated state as an authentication failure.
default:
    principal := record.Value
}

if err := sessions.Save(writer, claims, clock()); err != nil {
    return err
}
```

Cookies are host-only, `Secure`, `HttpOnly`, `Path=/`, and `SameSite=Lax` by
default. `__Host-` and `__Secure-` prefix rules are enforced. `SameSite=None`
requires `Secure`. `AllowInsecure` exists only for isolated local HTTP tests;
it is explicit and cannot weaken prefixed or cross-site cookies.

Session JSON is encrypted and authenticated with a fresh cryptographic nonce.
The token version, cookie name, and key ID are authenticated associated data.
Payloads are limited to 2 KiB, complete cookie headers to 4 KiB, lifetimes to
30 days, clock skew to five minutes, and key sets to eight entries. Decoding
rejects duplicate cookies, unknown JSON fields, trailing values, null values,
unknown keys, tampering, future issuance beyond skew, and expiry. Errors never
contain plaintext or token contents.

`Clear` emits deletion with the same path, domain, security, and SameSite
attributes. Managers are immutable and safe for concurrent use.

This is a stateless bearer session: confidentiality does not prevent replay,
and removing a key invalidates every cookie under that key. Applications that
need per-session revocation, large state, or clustered mutable sessions should
store only an opaque identifier in the encrypted cookie and provide an
explicit server-side store. SameSite is a CSRF defense-in-depth control, not a
replacement for origin checks or CSRF tokens on sensitive browser actions.
