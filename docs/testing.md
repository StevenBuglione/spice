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

## Generated application context

`spicetest.NewContext` owns construction, optional startup, and bounded
idempotent shutdown while preserving the concrete generated application type:

```go
testContext, err := spicetest.NewContext(
    context.Background(),
    func(ctx context.Context) (*commerce.Application, error) {
        return commerce.NewApplicationWithOptions(
            ctx,
            commerce.ApplicationOptions{
                Sources: []config.Source{testConfiguration},
                Logger:  slog.New(slog.DiscardHandler),
            },
        )
    },
    spicetest.ContextOptions{ShutdownTimeout: time.Second},
)
if err != nil {
    t.Fatal(err)
}
t.Cleanup(func() {
    if err := testContext.Close(); err != nil {
        t.Error(err)
    }
})

processor := testContext.Application().Components().StripeProcessor
```

Lifecycle hooks start by default. `SkipStart` retains the constructed state for
focused slices that must not open production listeners. Startup and shutdown
timeouts are finite and validated; cancellation and factory failures preserve
their causes, and a factory that returns a constructed application with an
error is stopped using a fresh bounded context.

The generated `Components` value is a compile-time typed snapshot of
constructed singleton beans. It is intended for tests and explicit embedding;
it performs no string lookup, reflection, package scan, or hidden construction.
Prototype/request/session beans remain available only through their generated
typed scope owners.

## Typed bean overrides

Every generated application also exposes a target-specific `BeanOverrides`
structure. Its fields are generated only for application-constructed singleton
beans whose exact Go types can be named by callers. Tests replace a bean by
assigning a value of that exact type:

```go
replacement := orders.NewViewAudit()
testContext, err := spicetest.NewContext(
    context.Background(),
    func(ctx context.Context) (*commerce.Application, error) {
        return commerce.NewApplicationWithOptions(
            ctx,
            commerce.ApplicationOptions{
                Overrides: commerce.BeanOverrides{
                    ViewAudit: bean.Replace(replacement),
                },
                Sources: []config.Source{testConfiguration},
                Logger:  slog.New(slog.DiscardHandler),
            },
        )
    },
    spicetest.ContextOptions{ShutdownTimeout: time.Second},
)
```

The Go compiler checks the replacement type. There is no bean-name string,
interface conversion, mutable registry, reflection, or runtime lookup.
Downstream generated constructors receive the replacement through the same
direct calls as the production bean.

When a test harness, library fixture, and individual test each contribute
overrides, use generated `BeanOverrideLayer` values and
`ComposeBeanOverrides`. Names must be unique, composition is deterministic,
and the later child layer wins for an enabled exact-type field. Composition
finishes before `NewApplicationWithOptions`; it never mutates a running test
context.

Use `bean.ReplaceFactory` when a replacement needs construction failure or
cleanup behavior:

```go
Repository: bean.ReplaceFactory(func(ctx context.Context) (
    *storage.OrderRepository,
    lifecycle.Cleanup,
    error,
) {
    repository, err := newTestRepository(ctx)
    return repository, cleanupTestRepository(repository), err
}),
```

Generated construction registers a successful factory cleanup immediately
under the original bean's module. Startup failure rolls it back in reverse
order, and normal `Stop` owns the same idempotent lifecycle path. The zero
value of every override is disabled, so ordinary production
`ApplicationOptions` are unchanged.

Spice dogfoods this contract in
`internal/spicegen/spice/application_test.go`. The test constructs the
production generated CLI application through `spicetest.NewContext`, verifies
its typed runtime, 13 handler interface beans, command component, configuration
schema, and lifecycle states, then replaces the exact `VersionHandler` bean to
prove the generated ordered interface collection observes the replacement. A
separate `BeanOverrides.Command` factory proves construction and exactly-once
cleanup through idempotent shutdown, while the real generated command runs an
LSP initialize/shutdown exchange over caller-owned streams. This is a
handwritten test beside the generated target; it does not modify generated
files or introduce a test-only container.

Configuration remains replaceable through explicit test `config.Source`
values. Prototype, request, and session values retain their generated typed
scope/provider contracts rather than being promoted to process-wide
singletons for tests.

Annotation authors use `annotation/sdk/sdktest.RunHandlerCases` for the other
side of the extension boundary. The public-only harness validates descriptor
metadata, allowed targets, typed contributions, diagnostics, cancellation,
timeouts, and deterministic repeated results. The independent annotation
fixture module uses this harness, proving that it does not depend on compiler
internals.

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
failures, caching/events, and the generated management surface.

## Transaction-rollback data slice

`spicetest.NewSQL` begins a transaction on a caller-owned `*sql.DB` and passes
only the restricted `data.Executor` surface to a typed subject factory:

```go
slice, err := spicetest.NewSQL(
    ctx,
    database,
    func(ctx context.Context, executor data.Executor) (*orders.Repository, error) {
        return orders.NewRepository(executor)
    },
    spicetest.SQLOptions{Isolation: sql.LevelSerializable},
)
if err != nil {
    t.Fatal(err)
}
t.Cleanup(func() {
    if err := slice.Close(); err != nil {
        t.Error(err)
    }
})

repository := slice.Value()
```

The factory and subject share the same transaction. `Close` always rolls it
back, is safe under concurrent calls, and returns the same rollback outcome to
every caller. The executor deliberately omits commit and rollback methods.
Factory errors roll back before return; factory panics attempt rollback and
re-panic with the original value. If both panic and rollback fail,
`SQLRollbackPanic` preserves the original value without formatting it and
unwraps the rollback failure.

Isolation and read-only policy are explicit. The harness owns no database,
schema, migration, truncation, or connection-pool lifecycle. Database
sequences and work performed outside the supplied executor are not
transactional and therefore are not reset by the slice.

The PostgreSQL integration suite proves under the race detector that rows are
visible to the transaction-scoped subject and absent after `Close`.
