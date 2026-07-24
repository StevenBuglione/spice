# Generated Application Bootstrap, Rollback, and Lifecycle

Date: 2026-07-24

## Question

After Spice has one typed Go program, resolved annotations, a validated `@Bean` provider catalog, and a deterministic provider dependency graph, what application-construction and lifecycle contract should come next so generated programs start predictably, roll back partial failures, and shut down gracefully without becoming a reflection container?

## Current delivery state

At the time of this research:

- issue #8 / PR #15 is the active delivery lane for type-aware package loading and stable symbols and is waiting on independent verification of exact head `1100a70b14fa5e56e05e7b2c33c426d4d5f06d5e`;
- issue #11 is ready next for resolving annotations against typed Go symbols and migrating CLI loading;
- issue #13 follows with `@Bean` provider signature analysis and a deterministic provider catalog;
- issue #17 follows with missing-dependency diagnostics, cycle detection, and a stable dependency-first construction order;
- the ready backlog is already at its cap of three, so this run creates no additional implementation issue;
- lifecycle design must consume issue #17's validated graph and must not add another package load, rescan provider signatures, or reconstruct dependency ordering at runtime.

## Why this is the next architecture question

A valid provider graph is necessary but not sufficient to run an application. Spice still needs explicit answers for:

- how generated code invokes providers and propagates constructor errors;
- how already-created resources are released when a later constructor fails;
- when long-running components begin accepting work;
- how startup failure rolls back previously started components;
- how cancellation and operating-system signals reach the application;
- what order shutdown follows;
- whether shutdown continues after one component fails;
- how HTTP servers stop accepting traffic while in-flight requests complete;
- which responsibilities belong in generated code versus a small runtime package;
- how all of this remains deterministic, testable, and ordinary Go.

Spring provides strong lifecycle outcomes through its application context, bean callbacks, `Lifecycle`/`SmartLifecycle`, and graceful web-server shutdown. Spice should provide those outcomes with generated wiring and explicit Go contracts rather than a dynamic container.

## Primary sources and status

Sources were accessed on 2026-07-24.

### Spring Framework and Spring Boot

- Bean initialization and destruction callbacks:
  - https://docs.spring.io/spring-framework/reference/core/beans/factory-nature.html
- `LifecycleProcessor` application-context start/stop processing:
  - https://docs.spring.io/spring-framework/docs/current/javadoc-api/org/springframework/context/LifecycleProcessor.html
- `SmartLifecycle` ordering, automatic startup, and stop callbacks:
  - https://docs.spring.io/spring-framework/docs/current/javadoc-api/org/springframework/context/SmartLifecycle.html
- Spring Boot graceful shutdown:
  - https://docs.spring.io/spring-boot/reference/web/graceful-shutdown.html
- Spring application startup and shutdown behavior:
  - https://docs.spring.io/spring-boot/reference/features/spring-application.html

Spring Framework and Spring Boot use Apache License 2.0. Spice uses their documentation as capability evidence, not as source code or a requirement to reproduce JVM container mechanics.

Relevant outcomes:

- managed initialization and destruction;
- dependency-aware startup and reverse shutdown;
- startup failure prevents the application from becoming ready;
- web shutdown rejects new work while allowing active requests a grace period;
- lifecycle timeouts are configurable;
- context close coordinates lifecycle stopping before final destruction.

Spring's phase model and pause/restart support are broader than Spice's first lifecycle slice needs.

### Go standard library

- Cancellation, deadlines, and cancellation causes:
  - https://pkg.go.dev/context
- Signal-derived cancellation:
  - https://pkg.go.dev/os/signal#NotifyContext
- HTTP graceful shutdown:
  - https://pkg.go.dev/net/http#Server.Shutdown
- Deterministic aggregation of multiple errors:
  - https://pkg.go.dev/errors#Join

These packages are BSD-3-Clause standard-library facilities and require no external dependency.

Relevant constraints:

- contexts should be passed explicitly rather than stored as hidden global state;
- cancellation functions must be called to release resources;
- `signal.NotifyContext` changes signal behavior until its stop function is called;
- `http.Server.Shutdown` stops listeners, closes idle connections, and waits for active connections, but does not manage hijacked connections such as WebSockets;
- process exit must wait for shutdown completion;
- `errors.Join` preserves all non-nil shutdown errors for `errors.Is` and `errors.As` inspection.

### Uber Fx

- Application and lifecycle documentation:
  - https://pkg.go.dev/go.uber.org/fx

Fx is MIT licensed and remains a runtime dependency-injection and lifecycle framework. It is comparison evidence, not a proposed Spice dependency.

Useful lifecycle behavior:

- startup hooks execute serially in dependency registration order;
- stop hooks execute serially in reverse order;
- startup failure triggers stop hooks for successfully started components;
- hook calls receive deadline-bearing contexts;
- stop continues through registered hooks and reports errors.

Spice should keep these predictable lifecycle properties while generating direct calls and avoiding runtime reflection or mutable registration as its primary application model.

### Google Wire

- Generated, reflection-free initialization rationale:
  - https://github.com/google/wire
- Design comparison with runtime DI and other languages:
  - https://github.com/google/wire/blob/main/docs/faq.md

Wire is Apache License 2.0 and was archived on 2025-08-25. It remains useful precedent for explicit generated initialization, but it should not become a Spice dependency.

## Findings and decisions

### 1. Split construction, startup, running, and shutdown into distinct phases

The application lifecycle should be explicit:

```text
validate and generate
  -> construct providers
  -> start managed components
  -> report ready and run
  -> receive cancellation
  -> stop managed components
  -> release constructed resources
```

Each phase has different failure and ordering rules. Combining all work inside constructors or a single `Run` callback would make partial rollback, diagnostics, tests, and future readiness reporting ambiguous.

### 2. Generated code owns the concrete application graph

A generated application package should contain direct fields and direct provider calls. Conceptually:

```go
package spicegen

type Application struct {
    config  app.Config
    store   *store.Store
    service *service.Service
    server  *server.Server

    lifecycle *runtime.Lifecycle
}

func NewApplication(ctx context.Context) (*Application, error)
func (a *Application) Start(ctx context.Context) error
func (a *Application) Stop(ctx context.Context) error
func (a *Application) Run(ctx context.Context) error
```

Exact names and generated-file placement require a later code-generation ADR, but the boundary should guarantee:

- ordinary typed Go fields and function calls;
- no reflection-based service lookup;
- no global application singleton;
- no hidden second provider graph;
- no package loading at application runtime;
- generated code that can be read, debugged, and tested without Spice tooling present;
- one small runtime coordinator only for generic lifecycle mechanics.

The runtime must not become a dynamic bean registry. Application-specific provider identities, dependency ordering, and hook calls belong in generated code.

### 3. Construction follows issue #17's exact stable order

`NewApplication` must invoke providers in the dependency-first order produced by the validated graph. It must not recalculate topological order from maps, source files, registration order, or runtime types.

For independent providers, issue #17's stable provider-symbol ordering remains the tie-break. Generated variable and field naming may differ, but observable invocation order must remain deterministic.

Provider invocation errors should include:

- provider stable symbol ID;
- source position;
- output type;
- original wrapped error;
- the construction phase in which failure occurred.

The generated constructor should return no partially usable `Application` after a provider error.

### 4. Construction cleanup is different from lifecycle stop

A constructor can acquire a resource before the application has started: an open file, temporary directory, database handle, client transport, or background allocation. An `OnStop` method alone cannot safely roll that resource back when a later constructor fails.

Spice therefore needs an explicit construction-cleanup contract in a later provider-signature extension. The recommended named runtime type is:

```go
package runtime

type Cleanup func(context.Context) error
```

A future provider-catalog slice may add these forms after issue #13's initial catalog is stable:

```go
func(...) T
func(...) (T, error)
func(...) (T, runtime.Cleanup)
func(...) (T, runtime.Cleanup, error)
```

Why a named cleanup return:

- it is explicit in ordinary Go signatures;
- it is refactor-safe and type-checkable;
- it cannot be confused with an arbitrary callback result;
- it supports deadlines and cancellation;
- it avoids implicitly treating every `io.Closer` as framework-owned;
- it can be called during constructor rollback even when startup never begins.

Issue #13 should not be expanded during its implementation. The cleanup forms should be a focused follow-up after the base catalog and graph are available.

A nil cleanup may be treated as no cleanup, but documentation should discourage returning a typed nil when a resource was actually acquired.

### 5. Cleanup is armed immediately after each successful provider call

When a provider returns a cleanup callback, generated code records it immediately after the provider succeeds and before invoking the next provider.

If construction later fails:

1. stop invoking new providers;
2. execute all armed cleanups in reverse construction order;
3. continue through every cleanup even when one fails;
4. return the constructor error joined with rollback errors in deterministic execution order.

Reverse order ensures consumers release their resources before dependencies.

Construction rollback must use a caller-provided context or a derived deadline. It must never silently switch to an unbounded `context.Background()` after the caller cancels.

### 6. Lifecycle hooks should be explicit typed methods, not implicit interface discovery

Automatically treating every value that happens to implement `Start` or `Close` as managed would create surprising behavior when a type gains a method. String-valued method names inside `@Bean` would also weaken refactoring support.

The recommended developer API is explicit method annotations resolved through issue #11's typed annotation pipeline:

```go
// @OnStart
func (s *Server) Start(ctx context.Context) error {
    return s.Listen(ctx)
}

// @OnStop
func (s *Server) Stop(ctx context.Context) error {
    return s.Shutdown(ctx)
}
```

Initial signature contract:

```text
func (receiver) Method(context.Context) error
```

The compiler should require:

- a method receiver whose exact type is supplied by one provider;
- at most one `@OnStart` and one `@OnStop` per provider output;
- no generic method declaration outside normal Go rules;
- no additional parameters;
- exactly one `error` result;
- `@OnStop` only when a corresponding `@OnStart` exists for that provider;
- source-positioned diagnostics for duplicate, unresolved, or invalid hooks.

A component needing release but no active startup should return `runtime.Cleanup` from its provider rather than declaring a stop-only lifecycle hook.

This keeps lifecycle participation intentional and makes generated calls visible.

### 7. Start hooks execute serially in dependency-first order

The first lifecycle implementation should be serial and deterministic:

1. use the graph construction order;
2. skip providers without `@OnStart`;
3. call each hook with the startup context;
4. record the hook as successfully started only after it returns nil;
5. stop immediately on the first startup error.

Serial startup is easier to reason about and matches dependency ordering. Parallel lifecycle phases can be designed later only with explicit phase/group metadata and demonstrated startup benefit.

When unrelated components require a specific order, the first model should require an explicit dependency edge rather than hidden source or annotation order.

### 8. Startup failure performs deterministic rollback

When an `@OnStart` hook fails:

1. do not report the application ready;
2. invoke `@OnStop` for previously successful start hooks in reverse successful-start order;
3. invoke all armed construction cleanups in reverse construction order;
4. continue through all rollback steps despite errors;
5. return the startup error joined with stop and cleanup errors in execution order.

Do not invoke the failing component's `@OnStop` because its start did not complete successfully. Its construction cleanup still runs if one was returned.

The initial implementation should not recover panics. A panic remains a programmer failure with ordinary Go semantics. Panic-safe rollback may be considered separately, but silently converting every panic into an error could hide defects and change debugging behavior.

### 9. Normal shutdown reverses successful startup and construction

`Application.Stop` should be idempotent and safe when called after:

- successful startup;
- partial startup failure;
- construction followed by no startup;
- a prior stop call.

For normal running applications:

1. invoke successful `@OnStop` hooks in reverse startup order;
2. invoke construction cleanups in reverse construction order;
3. continue after individual errors;
4. return `errors.Join` over errors in deterministic execution order.

The lifecycle coordinator should use `sync.Once`-equivalent state protection internally so repeated stop calls do not invoke hooks twice. Concurrent `Start` and `Stop` calls should return a clear state error rather than race.

### 10. The caller owns cancellation and signal policy

The reusable application API should accept a context and should not register process signals itself:

```go
func (a *Application) Run(ctx context.Context) error
```

A generated command entrypoint may provide the convenient process policy:

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), supportedShutdownSignals()...)
    defer stop()

    app, err := spicegen.NewApplication(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if err := app.Run(ctx); err != nil {
        log.Fatal(err)
    }
}
```

The exact logging and exit policy belongs to the generated command, not the reusable runtime. The application/runtime packages must never call `os.Exit` or `log.Fatal`.

Separating signal policy provides:

- embeddability in tests and other programs;
- explicit ownership of signal handler registration and release;
- easier Windows and Unix support;
- programmatic shutdown without synthetic operating-system signals;
- future support for non-process hosts.

### 11. `Run` is a convenience composition, not hidden magic

Conceptually, `Run` should:

```text
Start(startContext)
wait for ctx.Done()
Stop(shutdownContext)
return joined lifecycle errors
```

Timeout policy must remain explicit. The core lifecycle methods receive contexts supplied by the caller. A later configuration slice may generate startup and shutdown timeout options, but the runtime must not create an unbounded background context or hide a non-configurable deadline.

The initial runnable example may define documented constants for startup and shutdown deadlines until typed external configuration exists.

### 12. HTTP graceful shutdown falls naturally out of reverse dependency order

An HTTP server is normally a consumer of controllers and services. If its `@OnStart` hook starts serving and its `@OnStop` hook calls `http.Server.Shutdown`, reverse dependency shutdown means:

1. stop accepting new HTTP work;
2. wait for active requests up to the shutdown deadline;
3. then release service, store, and infrastructure dependencies.

That is the desired Spring Boot graceful-shutdown outcome without a web-server-specific application container.

Spice HTTP integration must document that hijacked or upgraded connections such as WebSockets need their own shutdown handling because `http.Server.Shutdown` does not close them automatically.

### 13. Readiness begins only after every start hook succeeds

The application is not ready while providers are constructing or hooks are starting. A future health subsystem should expose:

```text
starting -> ready -> stopping -> stopped
                 \-> failed
```

The first lifecycle slice does not need an Actuator-equivalent endpoint, but it should expose an inspectable application state so M2 health/readiness adapters can report it without guessing.

No HTTP listener should be considered ready before its own start succeeds and all earlier dependency hooks have succeeded.

### 14. Keep phases, parallelism, pause/restart, and dynamic registration out initially

Spring `SmartLifecycle` supports phases, automatic startup, concurrent shutdown within phases, pause, and restart. Those are valuable evidence but excessive for the first Go-native contract.

Defer:

- numeric lifecycle phases and priorities;
- parallel startup or shutdown;
- pause/restart semantics;
- runtime hook registration from arbitrary constructors;
- dynamic provider addition or removal;
- lazy provider construction;
- multiple application contexts;
- per-module child containers;
- hot reload;
- automatic interface-based lifecycle discovery;
- finalizers or garbage-collector-driven cleanup.

Dependency edges and stable provider IDs are the only initial ordering mechanism.

### 15. Generated and runtime responsibilities must remain sharply separated

Generated code owns:

- concrete provider calls;
- concrete values and types;
- exact hook method calls;
- provider and source metadata;
- stable construction/start/stop order;
- registration of returned cleanup callbacks;
- application-specific error context.

The small runtime owns:

- lifecycle state transitions;
- once-only stop protection;
- ordered callback execution helpers;
- deadline-aware callback invocation;
- deterministic multi-error aggregation;
- generic application-state observation.

The runtime must not own:

- package scanning;
- type resolution;
- annotation parsing;
- provider selection;
- dependency graph construction;
- reflection-based invocation;
- service lookup by type or string.

### 16. Error output must preserve both the primary failure and rollback failures

Examples:

```text
construct provider example.com/app/store.NewStore: dial database: connection refused
rollback provider example.com/app/config.NewWatcher: stop watcher: context deadline exceeded
```

```text
start component example.com/app/http.(*Server).Start: bind :8080: address already in use
stop component example.com/app/jobs.(*Worker).Stop: drain jobs: context deadline exceeded
cleanup provider example.com/app/store.NewStore: close database: broken pipe
```

Errors should wrap their underlying causes so `errors.Is` and `errors.As` remain useful. Aggregation order should follow actual deterministic rollback/shutdown order rather than map or goroutine completion order.

### 17. Security and operational defaults

The lifecycle design must prevent common unsafe states:

- no traffic readiness before successful startup;
- no continued provider construction after a fatal constructor error;
- no silent omission of cleanup after partial construction;
- no unbounded shutdown unless the caller explicitly supplies an unbounded context;
- no swallowing of secondary cleanup errors;
- no automatic execution of arbitrary methods based only on naming;
- no global signal handler that survives application shutdown;
- no shutdown order that destroys dependencies before consumers;
- no concurrent hook execution before its semantics are explicitly designed.

Generated code should avoid logging secrets or provider argument values in lifecycle diagnostics. Stable symbol IDs, output types, source positions, and wrapped errors are sufficient context.

## Recommended future implementation sequence

The backlog is full, so no issue is created now. When capacity opens, lifecycle should be delivered in bounded vertical slices after issues #8, #11, #13, and #17 merge.

### Slice A — Construction cleanup metadata

- Add `runtime.Cleanup`.
- Extend the provider catalog with the two cleanup-return forms.
- Validate exact signatures.
- Represent cleanup capability in provider records.
- Do not generate or invoke providers yet.

### Slice B — Typed lifecycle-hook metadata

- Add `@OnStart` and `@OnStop` definitions.
- Resolve hooks to exact receiver/provider records.
- Validate signatures and uniqueness.
- Produce deterministic hook metadata.
- Do not run hooks yet.

### Slice C — Generated application constructor and rollback

- Generate direct provider invocation in graph order.
- Register cleanups immediately.
- Roll back construction failures in reverse order.
- Compile and run a generated fixture.

### Slice D — Lifecycle runtime and runnable application

- Generate direct start and stop hook calls.
- Implement startup rollback and idempotent shutdown.
- Add `Application.Run(context.Context)`.
- Add signal-driven example command behavior.
- Exercise real `net/http` graceful shutdown and in-flight request completion.

Each slice should remain small enough for one implementation run and must include `make verify`, focused unit tests, generated-code compilation, and executable integration behavior.

## Required future test matrix

### Construction

- dependency-first provider invocation;
- stable order for independent providers;
- constructor error stops later construction;
- cleanups execute in exact reverse order;
- all rollback errors are retained;
- no partially usable application escapes;
- repeated construction produces byte-identical generated code and stable traces.

### Lifecycle metadata

- valid pointer and value receivers where output identity matches exactly;
- missing provider for a hook receiver;
- duplicate start or stop hook;
- invalid parameters or results;
- `@OnStop` without `@OnStart`;
- deterministic diagnostics from reversed source declaration order.

### Startup and shutdown

- dependency-first start order;
- reverse successful-start stop order;
- start failure prevents later starts;
- start failure triggers stop and construction cleanup;
- stop continues after errors;
- stop is idempotent;
- concurrent invalid state transitions fail clearly;
- canceled and expired contexts propagate;
- process signal cancellation stops a runnable example;
- no signal handler remains after command exit.

### HTTP integration

- application becomes ready only after listener startup succeeds;
- shutdown stops new requests;
- an in-flight request completes within the deadline;
- deadline expiry is reported;
- service dependencies remain alive until the HTTP server finishes shutdown;
- upgraded connection limitations are documented and tested by the later WebSocket integration.

## Spring and Go comparison

| Concern | Spring outcome | Go/Fx evidence | Spice direction |
|---|---|---|---|
| Construction | Container instantiates active beans | Direct constructors; Fx constructors | Generated direct provider calls in stable graph order |
| Initialization | Bean callbacks and context refresh | Fx serial start hooks | Explicit typed `@OnStart` methods, generated direct calls |
| Destruction | Bean destruction and context close | Fx reverse stop hooks | Explicit `@OnStop` plus separate provider cleanup |
| Partial construction | Context creation fails and destroys created beans | Explicit Go cleanup required | Named cleanup returns armed immediately and reversed on failure |
| Startup failure | Context does not become ready | Fx stops previously started hooks | Reverse successful starts, then reverse construction cleanups |
| Graceful web stop | Reject new work and drain active requests | `http.Server.Shutdown` | HTTP server stop hook before service dependency cleanup |
| Ordering | Dependencies plus phases | Fx dependency registration order | Graph order first; phases deferred |
| Cancellation | Container shutdown and configured timeout | `context`, `signal.NotifyContext` | Caller-owned contexts and signal policy |
| Error aggregation | Framework lifecycle exceptions/logging | `errors.Join` | Wrapped deterministic joined errors |
| Runtime model | Dynamic application context | Fx runtime container | Small generic lifecycle coordinator plus generated typed application |

## Decision summary

Spice should provide Spring-quality startup and shutdown outcomes through a generated typed application, not a reflection container.

The intended contract is:

1. issue #17 supplies the one stable dependency-first graph order;
2. generated code invokes providers directly in that order;
3. providers may later return a named context-aware cleanup callback;
4. cleanups are armed immediately and run in reverse order on construction failure or final shutdown;
5. lifecycle participation is explicit through typed `@OnStart` and `@OnStop` methods;
6. start hooks run serially dependency-first;
7. stop hooks run serially in reverse successful-start order;
8. startup failure rolls back started hooks and all constructed resources;
9. callers own contexts, deadlines, signal policy, logging, and process exit;
10. a small runtime coordinates state and callbacks but never discovers or resolves application dependencies;
11. HTTP graceful shutdown uses ordinary `net/http` behavior and naturally precedes dependency cleanup;
12. phases, parallelism, pause/restart, dynamic registration, and implicit lifecycle interfaces remain deferred.

No implementation issue is created in this run because issues #11, #13, and #17 already fill the three-item ready-backlog cap.