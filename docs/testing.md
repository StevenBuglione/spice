# Focused module testing

`spice test` runs ordinary Go tests for one verified application-module graph:

```bash
spice test \
  --module=example.com/shop/orders \
  --race \
  --count=1 \
  ./...
```

The module identity is the full import path from its package-level `@Module`.
Package patterns are used for compiler discovery and default to `./...`; they
should include every module root that can participate in the graph.

Before starting `go test`, Spice resolves and validates annotations, builds the
module model, rejects architecture violations, and focuses it on the requested
module. The executed package list contains exactly:

1. transitively observed dependency modules, in dependency-first order;
2. the focused module;
3. every package owned by those modules, sorted within each module.

Dependents, unrelated modules, declared-but-unused dependencies, and unassigned
packages are excluded. Spice passes full package import paths directly to
`go test -trimpath`; it does not create a test container, scan packages at
runtime, generate files, or change application wiring.

Supported test controls are:

| Option | Effect |
|---|---|
| `--race` | Enable the Go race detector. |
| `--count=N` | Run each test and benchmark `N` times; `N` must be positive. |
| `--run=REGEXP` | Run tests matching the standard Go test expression. |
| `--timeout=DURATION` | Apply a positive Go test timeout. |

Invalid command usage exits with status 2. Package loading, annotation,
architecture, focus, and `go test` failures exit with status 1. A fully passing
focused graph exits with status 0.

This command is the Modulith-style package test slice.

## Generated HTTP application slice

`spicetest.NewHTTP` constructs an actual generated application through a typed
factory and exposes its generated handler on an ephemeral IPv4 loopback
listener:

```go
server, err := spicetest.NewHTTP(
    context.Background(),
    func(ctx context.Context) (spicetest.HTTPApplication, error) {
        return commerce.NewApplicationWithOptions(
            ctx,
            commerce.ApplicationOptions{
                Sources: []config.Source{testConfiguration},
                Logger:  slog.New(slog.DiscardHandler),
            },
        )
    },
    spicetest.HTTPOptions{ShutdownTimeout: time.Second},
)
if err != nil {
    t.Fatal(err)
}
t.Cleanup(func() {
    if err := server.Close(); err != nil {
        t.Error(err)
    }
})
```

The slice accepts the ordinary generated `Handler`, `State`, and `Stop`
surface; it performs no reflection, package scan, provider replacement, signal
capture, or process-environment read. The factory remains responsible for
explicit generated options and test configuration sources.

`HTTPRequest` supports either a JSON value or explicit bytes, cloned headers,
and a root-relative request target. Request and response bodies are explicitly
bounded; responses are completely detached before return. `DecodeJSON` accepts
exactly one JSON value and `Problem` validates both the RFC 9457 document and
agreement between document and HTTP status.

The test listener is loopback-only. Client, response-body, and shutdown limits
are finite and validated. Construction/listen failure stops a successfully
constructed application with a fresh bounded context. `Close` gracefully stops
the server before generated cleanup, is safe under concurrent calls, and joins
all failures. The slice deliberately does not start application lifecycle
hooks: a focused web test should not bind provider-owned production listeners.
Call `Application.Start` explicitly before creating the slice only when a
broader integration test requires lifecycle readiness.

The commerce reference uses this harness for typed controllers, RFC 9457
failures, caching/events, and the generated management surface. A specialized
transaction-rollback data harness remains roadmap work.
