# Spice Annotation Syntax

## Canonical form

Spice annotations are declaration comments:

```go
// @Controller(prefix="/users")
type UserController struct{}
```

The parser also accepts `//@Controller`, but `gofmt` inserts a space after `//`, making `// @Controller` the canonical documented form.

## Names

Core declaration names are unqualified. Bootstrap features use qualified
built-in names such as `@management.Enable` and
`@observability.Logging`. Third-party qualified definitions use the same model
when their complete manifests are explicitly selected in
`.spice/starters.json`. During bootstrap, `spice verify` fails closed on
unknown annotations. Package imports and `go.mod` entries do not register or
activate definitions.

## Arguments

Definitions decide whether arguments are named, positional, required, and which parsed value kinds they accept. Spice does not silently coerce values.

Controller prefixes are optional and named-only:

```go
// @Controller(prefix="/users")
type UserController struct{}

// @Controller
type RootController struct{}
```

Route paths are required strings and support either named or concise positional syntax:

```go
// @Get(path="/{id}")
func (UserController) GetUser() {}

// @Get("/{id}")
func (UserController) GetUserCompact() {}

// @Post("/")
func (UserController) CreateUser() {}
```

Marker annotations such as `@Application`, `@Bean`, `@Service`, `@OnStart`,
`@OnStop`, and `@observability.Logging` accept no arguments.
`@Configuration` accepts an optional named `prefix` string.
`@management.Enable` requires the named `expose` list.

The bootstrap parser supports strings, integers, booleans, identifiers, and
lists. Definitions can constrain list element kinds; management exposure, for
example, requires strings and then applies its endpoint enum validation. The
validator does not yet implement defaults, aliases, composed annotations, or
nested annotations.

## `@Application` marker functions

`@Application` marks an ordinary package-level function whose parameter types
are application roots:

```go
// @Application
func Commerce(server *HTTPServer, worker Worker) {
	panic("compile-time marker; Spice never executes this body")
}
```

The annotation takes no arguments. The function must be non-generic,
non-variadic, and return no results. Every parameter must be the exact Go type
produced by one `@Bean`; aliases preserve exact identity, while implicit
interface implementation, assignability, pointer/value conversion, and
underlying-type equality do not select a provider.

The marker body has no framework semantics and is never executed during
analysis. Packages without a marker remain valid for library verification.
Multiple markers are represented as distinct deterministic application targets;
generation requires an unambiguous selected target before it may write files.

`@Application` also composes safe generated-command conventions. Qualified
companions opt into behavior with exposure or operational consequences:

```go
// @Application
// @management.Enable(expose=["health", "liveness", "readiness", "info", "metrics", "configprops", "modules"])
// @observability.Logging
func Commerce(*Server) {}
```

Both companions are valid only on an `@Application` function. Endpoint names
are exact, duplicates and unknown names fail at their source positions, and
the normalized metadata becomes part of the immutable application IR.

## `@Bean` provider functions

`@Bean` marks an ordinary package-level Go factory function for compile-time provider metadata:

```go
// @Bean
func NewUserService(repository UserRepository) (*UserService, error) {
    return &UserService{repository: repository}, nil
}
```

The catalog accepts these exact forms:

```go
func(dependencies...) T
func(dependencies...) (T, error)
func(dependencies...) (T, lifecycle.Cleanup)
func(dependencies...) (T, lifecycle.Cleanup, error)
```

`lifecycle.Cleanup` is the named context-aware callback `func(context.Context) error`. An alias to that exact type is accepted; unnamed or distinct defined function types are rejected even when their underlying signatures match. Cleanup is metadata only in this release: it must be the second result, `error` must be final, and the first result remains the sole provided value. A one-result provider whose value itself has type `lifecycle.Cleanup` is an ordinary provider of that value.

Every parameter is a required exact-type dependency for the graph phase. Provider methods, generic or variadic functions, annotation arguments, malformed result ordering, multiple cleanup or error results, and extra values are rejected with source-positioned diagnostics.

`spice verify` validates catalog and graph metadata but does not execute providers or cleanup callbacks. The pure generator now renders exported providers as direct calls in graph order and registers cleanup immediately; filesystem application is still a separate explicit command layer. A dedicated generated package cannot import `package main` or call unexported declarations, so render validation requires providers and lifecycle hooks to be exported and declared in importable packages.

Exact output types must have one provider; Spice rejects duplicates rather than choosing by declaration order or implicit interface assignability. Interface bindings, qualifiers, scopes, optional values, groups, and collection injection remain explicit future capabilities.

## Lifecycle hook metadata

Argument-free, method-only `@OnStart` and `@OnStop` select explicit methods for
generated lifecycle orchestration. A hook must have the exact non-variadic form
`func(receiver)(context.Context) error`, and its receiver must be semantically
identical to exactly one valid `@Bean` output.

Aliases to the exact receiver, `context.Context`, and `error` types are accepted. Pointer/value convenience, assignability, interface implementation, structural context lookalikes, method promotion, duplicate roles, and stop-only components are rejected.
The compiler records deterministic typed metadata only. `spice verify` never invokes providers, cleanup callbacks, or lifecycle methods. Generated applications use the public `lifecycle.Coordinator` for the state machine, dependency-order start, reverse stop/cleanup, startup rollback, deterministic error joining, idempotent stop, and run/wait/shutdown composition. Concrete hook calls remain direct generated method values.

Generated `Run` accepts the caller's run context and a caller-supplied shutdown-context factory. This keeps operating-system signals and fresh shutdown deadlines in the command while allowing the framework to stop gracefully after cancellation without inventing a hidden background context.

## Fixed-delay scheduling

`@schedule.FixedDelay` targets an exported method owned by exactly one exact
`@Bean` output. The method contract is
`func(receiver)(context.Context) error`. Its required named `delay` argument
must be a positive Go duration string. Optional `initialDelay` must be
non-negative, and optional `continueOnError` is Boolean.

```go
// @schedule.FixedDelay(delay="30s", initialDelay="5s")
func (*Inventory) Refresh(context.Context) error {
    return nil
}
```

The compiler normalizes durations into immutable scheduling IR and generation
emits a direct method value. A single generated scheduler starts after ordinary
provider hooks and shuts down before them. No method body executes during
analysis, and no runtime annotation lookup or global scheduler exists.

## Typed application events

`@event.Listener` targets an exported method owned by exactly one exact
`@Bean`. Its signature is
`func(receiver)(context.Context, Event) error`; optional named integer `order`
controls deterministic delivery order.

An exported `@event.Topic` marker function selects listener owners through its
exact parameter types and returns one exact `event.Publisher[Event]`:

```go
// @event.Listener(order=10)
func (*Inventory) Reserve(context.Context, OrderPlaced) error {
    // ...
}

// @event.Topic
func OrderEvents(*Inventory) event.Publisher[OrderPlaced] {
    panic("Spice never executes event topic marker bodies")
}
```

The event must be an exported named value. Every marker parameter must select
exactly one listener for that payload, every annotated listener must belong to
one topic, and an ordinary provider may depend on the synthetic exact
`event.Publisher[Event]` node. Provider cycles and duplicate publishers fail in
the normal graph/catalog stages. Generation binds the listener methods directly
to their constructed provider receivers and constructs an instance-owned
`event.Topic[Event]`; it never calls the marker body.

## Cacheable HTTP reads

`@cache.Cacheable` declares an explicit generated cache boundary on a typed
`@Get` method:

```go
// @Get("/products/{id}")
// @cache.Cacheable(name="products.by-id")
func (*Products) Product(
    context.Context,
    ProductRequest,
) (ProductResponse, error) {
    // ...
}
```

The required `name` is a stable architecture identity, not a capacity or TTL.
It must use lowercase alphanumeric segments separated by `.` or `-` and must
be unique in the application. The request must be an exported comparable named
struct value and becomes the exact cache key type. Raw, mutating, no-content,
transactional, and authorization-sensitive routes fail closed. Runtime
capacity and TTL belong to typed configuration. Generation constructs one
bounded in-memory store and emits direct get/call/put logic; method errors are
never cached.

## Transactional HTTP routes

`@data.Transactional` targets an exported typed `@Get` or `@Post` method. The
method must make the transaction dependency explicit:

```go
// @Post("/orders")
// @data.Transactional(isolation="serializable", readOnly=false)
func (*OrdersController) Create(
    context.Context,
    data.Executor,
    CreateOrderRequest,
) (CreateOrderResponse, error) {
    // ...
}
```

The exact route signature is
`func(receiver)(context.Context, data.Executor, RequestDTO) (Response, error)`.
An exact `*data.Manager` provider is required. `isolation` is an optional
named string and `readOnly` is an optional named Boolean. Generation wraps the
direct route call in `Manager.Within`; Spice never places a transaction in the
context or performs runtime annotation lookup. See
[`data.md`](data.md) for isolation values and runtime semantics.

## Application modules

`@Module` is a package-documentation annotation. The annotated package's full
Go import path is the module identity and its root package is the default
public API. Descendant packages belong to the longest matching root and remain
internal unless exposed by a package-level named interface.

```go
// Package orders owns order processing.
//
// @Module(allowedDependencies=["example.com/shop/inventory", "example.com/shop/payments::spi"])
package orders
```

Allowed dependencies use exact module-root import paths. A plain path selects
the root default API; `module::interface` selects a named interface.

`@NamedInterface` is repeatable and accepts one positional or named string.
Names must match `^[a-z][a-z0-9-]*$`.

```go
// Package spi exposes payment contracts.
//
// @NamedInterface("spi")
package spi
```

Spice reports packages in the same Go module that are not owned by any
annotated module root. Short module names, implicit descendant APIs, self
dependencies, duplicate references, and unknown modules/interfaces are
rejected.

## Argument diagnostics

Invalid invocations fail before generation with deterministic source-positioned diagnostics:

```text
controller.go:3:1: annotation @Controller does not define argument "prefx"; available argument: prefix
controller.go:8:1: annotation @Get requires argument "path"
controller.go:13:1: annotation @Get argument "path" requires string, got integer
controller.go:18:1: annotation @Get assigns argument "path" more than once
service.go:3:1: annotation @Service does not accept arguments
```

A positional value is accepted only when exactly one definition argument is explicitly positional. Spice rejects multiple positional values and rejects positional syntax for named-only definitions.

## Built-in definitions and targets

| Annotation | Allowed target | Defined arguments |
|---|---|---|
| `@Application` | Package-level function | None |
| `@Bean` | Package-level function | None |
| `@cache.Cacheable` | Exact typed `@Get` method | `name` string, required and named-only |
| `@Configuration` | Type | `prefix` string, optional, named-only |
| `@Controller` | Type | `prefix` string, optional, named-only |
| `@Get` | Method | `path` string, required, named or positional |
| `@management.Enable` | `@Application` package-level function | `expose` string list, required, named-only |
| `@observability.Logging` | `@Application` package-level function | None |
| `@OnStart` | Method | None |
| `@OnStop` | Method | None |
| `@Post` | Method | `path` string, required, named or positional |
| `@data.Transactional` | Exact typed `@Get` or `@Post` method | `isolation` string and `readOnly` Boolean, optional and named-only |
| `@event.Listener` | Exact provider-owned exported method | `order` integer, optional and named-only |
| `@event.Topic` | Exported package-level marker function | None |
| `@security.Authorize` | `@Get` or `@Post` method | `authenticated` Boolean; `anyRoles`, `allRoles`, and `allScopes` string lists; all optional and named-only, but at least one requirement is mandatory |
| `@schedule.FixedDelay` | Exact provider-owned exported method | `delay` duration string, required; `initialDelay` duration string and `continueOnError` Boolean, optional; all named-only |
| `@Service` | Type | None |

Annotations may be discovered on packages, types, functions, methods, variables, and constants. Each annotation definition determines which declaration kinds and invocation forms are legal.
