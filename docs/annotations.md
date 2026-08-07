# Spice Annotation Syntax

## Canonical form

Spice annotations are declaration comments:

```go
// @Controller(prefix="/users")
type UserController struct{}
```

The parser also accepts `//@Controller`, but `gofmt` inserts a space after `//`, making `// @Controller` the canonical documented form.

`spice lsp` completes this canonical form directly. Typing `@` on an otherwise
empty declaration line may insert the `// ` prefix together with the selected
annotation and required-argument snippet. A raw `@Annotation` line is invalid
Go; the language server reports the ordinary Go/compiler diagnostic and offers
a version-checked prefix insertion. It never stores Java-style syntax or hides
an invalid source representation from `gofmt`, `go test`, or other Go tools.

## Names and imports

New source may explicitly bind descriptor symbols per file:

```go
// @import { Application } from "github.com/spice-framework/spice/annotation/core"
// @import { Controller, Get as GET } from "github.com/spice-framework/spice/annotation/web"
// @import * as web from "github.com/spice-framework/spice/annotation/web"
```

Named bindings permit clean `@Application` and `@Controller` spellings.
Aliases permit `@GET`; namespace bindings permit visibly sourced
`@web.Controller`. Imports apply to the entire file. Every annotation in every
file must resolve through a named or namespace import in that same file. There
is no implicit built-in registry, package-level carryover, or name-based
fallback.

`@import` is the only import directive. The earlier `@spice.import` spelling is
not accepted as an alias: analysis reports
`spice.resolution.annotation-import-legacy` on the retired token, and editor
clients receive a version-checked quick fix replacing exactly that token with
`@import`. The surrounding bindings, aliases, package path, comments, and
physical `// ` prefix are preserved.

A file with an annotation but no matching import fails at the annotation
position. See [`annotation-sdk.md`](annotation-sdk.md) for the static
descriptor contract and offline module behavior.

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

Marker annotations such as `@Application`, `@Bean`, `@Enum`, `@OnStart`,
`@OnStop`, and `@observability.Logging` accept no arguments.
`@ConfigurationProperties` accepts an optional named `prefix` string.
`@Component`, `@Configuration`, `@Repository`, and `@Service` accept optional
`constructor`, `name`, and `aliases` bean-construction metadata.
`@management.Enable` requires the named `expose` list.

The bootstrap parser supports strings, integers, booleans, identifiers, and
lists. Definitions can constrain list element kinds; management exposure, for
example, requires strings and then applies its endpoint enum validation. The
validator does not yet implement defaults, aliases, composed annotations, or
nested annotations.

## `@Application` marker functions

The preferred application declaration is the ordinary Go process entrypoint:

```go
package main

import (
	"os"

	spiceapp "example.com/shop/internal/spicegen/shop"
)

// @Application
// @management.Enable(expose=["health", "readiness", "info"])
// @observability.Logging
func main() {
	os.Exit(spiceapp.Main(os.Args[1:]))
}
```

The preferred marker must be the parameterless, result-free, non-generic
`func main()` in package `main`. It explicitly imports the generated target
package and calls `Main`; Spice writes no generated declarations beside it.
During safe regeneration, the typed loader validates that exact import and
call and supplies an in-memory generated-package stub. Every actual Go load
error remains fatal.

Spice analyzes the selected local Go package scope, discovers
package-documentation `@Module` roots and supported annotated features, builds
one exact provider graph, and generates deterministic direct imports and
calls. `main.go` does not import modules merely to discover them. Package
patterns or an exact target selector bound multi-application repositories.

The pre-1.0 legacy form remains supported:

```go
// @Application
func Commerce(server *HTTPServer, worker Worker) {
	panic("compile-time marker; Spice never executes this body")
}
```

Each legacy parameter must be the exact Go type produced by one `@Bean`;
aliases preserve exact identity, while implicit interface implementation,
assignability, pointer/value conversion, and underlying-type equality do not
select a provider.

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
func main() {
	os.Exit(spiceapp.Main(os.Args[1:]))
}
```

Both companions are valid only on an `@Application` function. Endpoint names
are exact, duplicates and unknown names fail at their source positions, and
the normalized metadata becomes part of the immutable application IR.

## `@Configuration`, `@Component`, and `@Bean` methods

`@Component` fills the generic managed-object role between the specialized
`@Service`, `@Repository`, and `@Controller` stereotypes. All constructible
stereotypes select an ordinary constructor at compile time and are compatible
with explicit interface bindings and bean metadata.

`@Configuration` is a constructible factory type. `@Bean` normally marks one
of its methods, which makes the configuration receiver an explicit provider
dependency in addition to the method parameters:

```go
// @Configuration
type UserConfiguration struct{}

// @Bean
func (*UserConfiguration) UserService(
    repository UserRepository,
) (*UserService, error) {
    return &UserService{repository: repository}, nil
}
```

Package-level `@Bean` functions remain supported for pre-0.2 migration, but
the `java-structured` profile rejects them with
`spice.style.package-bean`. Generated code calls either form directly; no
configuration or provider body executes during analysis.

The catalog accepts these exact forms:

```go
func(dependencies...) T
func(dependencies...) (T, error)
func(dependencies...) (T, lifecycle.Cleanup)
func(dependencies...) (T, lifecycle.Cleanup, error)
```

`lifecycle.Cleanup` is the named context-aware callback `func(context.Context) error`. An alias to that exact type is accepted; unnamed or distinct defined function types are rejected even when their underlying signatures match. Cleanup is metadata only in this release: it must be the second result, `error` must be final, and the first result remains the sole provided value. A one-result provider whose value itself has type `lifecycle.Cleanup` is an ordinary provider of that value.

Every parameter is a required exact-type dependency for the graph phase. A
method provider must have a receiver whose exact named type is a constructible
`@Configuration`; that configuration bean is constructed before the method is
called. Generic or variadic providers, malformed result ordering, multiple
cleanup or error results, and extra values are rejected with source-positioned
diagnostics.

`spice verify` validates catalog and graph metadata but does not execute
providers or cleanup callbacks. The pure generator renders exported providers
as direct calls in graph order and registers cleanup immediately; filesystem
application remains a separate explicit command layer. Providers and lifecycle
hooks must be exported and declared in importable application-module packages,
not the process-only `main` shell.

Exact concrete outputs remain exact-type candidates. A concrete output becomes
an interface candidate only through typed `@Implements(pkg.Interface)`.
Namespace `@import` can bind the interface's package even when it contains no
Spice descriptors. The shared typed compiler verifies the exact method set and
generation emits the corresponding source-owned Go compile-time assertion; a
factory that returns the interface exactly needs no adapter annotation. Qualifiers,
`@Primary`/`@Fallback`, bean names and aliases resolve single values
deterministically. Slices and maps receive every matching bean in stable
`@Order`, name, and source order. `bean.Optional[T]`, `bean.Lazy[T]`, and
`bean.Provider[T]` make absence, deferred resolution, and caller-owned
prototype cleanup explicit. Singleton, prototype, request, and session scopes
retain distinct generated cleanup ownership. See
[`compiler.md`](compiler.md#typed-provider-catalog) for the
complete selection contract.

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

## Asynchronous execution

`@async.Execute` targets an exported method owned by exactly one exact
`@Bean` output. It accepts no annotation arguments. The non-variadic contract
is `func(receiver)(context.Context, arguments...) error`; parameter zero must
be the exact canonical context type, and remaining argument types must be
nameable from the generated application package.

```go
// @async.Execute
func (*Mailer) Send(context.Context, Message) error {
    return nil
}
```

The compiler derives a stable typed submit-method name, rejects collisions,
and stores copied argument types in immutable application IR. It does not
invoke the annotated method. Generation constructs one application-owned
bounded executor and exposes
`Application.Submit<Receiver><Method>(admissionContext, arguments...)`.
Submission requires a ready application and calls the provider method directly
on an accepted worker; there is no proxy or runtime method lookup.

## Typed application events

`@event.Listener` targets an exported method owned by exactly one exact
`@Bean`. Its signature is
`func(receiver)(context.Context, Event) error`; optional named integer `order`
controls deterministic delivery order.

`@event.Topic` belongs on the exported event payload type:

```go
// @event.Topic
type OrderPlaced struct {
    OrderID string
}

// @event.Listener(order=10)
func (*Inventory) Reserve(context.Context, OrderPlaced) error {
    // ...
}
```

The event must be an exported named value. Every annotated listener must belong
to one topic, and an ordinary provider may depend on the synthetic exact
`event.Publisher[Event]` node. Provider cycles and duplicate publishers fail in
the normal graph/catalog stages. Generation discovers listener-owner providers,
binds their methods directly, and constructs an instance-owned
`event.Topic[Event]`. Package-level function markers remain migration-only and
are rejected by `java-structured`.

## Closed enums

`@Enum` marks one named scalar type whose same-file typed constants are the
complete legal value set:

```go
// @Enum
type OrderStatus string

const (
    OrderStatusPending   OrderStatus = "pending"
    OrderStatusCompleted OrderStatus = "completed"
)
```

The compiler rejects missing members, duplicate underlying values, members of
another type, and typed constants for the enum declared in another file. It
generates ordinary `ParseOrderStatus`, `String`, and `Valid` helpers in the
source mirror; it does not create a reflection registry.

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
service.go:3:1: annotation @Service does not define argument "magic"; available arguments: aliases, constructor, name
```

A positional value is accepted only when exactly one definition argument is explicitly positional. Spice rejects multiple positional values and rejects positional syntax for named-only definitions.

## Built-in definitions and targets

| Annotation | Allowed target | Defined arguments |
|---|---|---|
| `@Application` | Package-level function | None |
| `@Bean` | Method on `@Configuration`; package function during migration | `name` string and `aliases` string list, optional and named-only |
| `@Component` | Type | `constructor` identifier, `name` string, and `aliases` string list, optional and named-only |
| `@async.Execute` | Exact provider-owned exported method | None |
| `@cache.Cacheable` | Exact typed `@Get` method | `name` string, required and named-only |
| `@Configuration` | Type | `constructor` identifier, `name` string, and `aliases` string list, optional and named-only |
| `@ConfigurationProperties` | Type | `prefix` string, optional, named-only |
| `@Controller` | Type | `prefix` string, `constructor` identifier, `name` string, and `aliases` string list, optional and named-only |
| `@Get` | Method | `path` string, required, named or positional |
| `@management.Enable` | `@Application` package-level function | `expose` string list, required, named-only |
| `@Module` | Package documentation | `allowedDependencies` string list, optional and named-only |
| `@NamedInterface` | Package documentation | Interface name string, required, named or positional; repeatable |
| `@observability.Logging` | `@Application` package-level function | None |
| `@OnStart` | Method | None |
| `@OnStop` | Method | None |
| `@Post` | Method | `path` string, required, named or positional |
| `@Repository` | Type | `constructor` identifier, `name` string, and `aliases` string list, optional and named-only |
| `@data.Transactional` | Exact typed `@Get` or `@Post` method | `isolation` string and `readOnly` Boolean, optional and named-only |
| `@event.Listener` | Exact provider-owned exported method | `order` integer, optional and named-only |
| `@Enum` | Named scalar type | None |
| `@event.Topic` | Exported event payload type; package function during migration | None |
| `@security.Authorize` | `@Get` or `@Post` method | `authenticated` Boolean; `anyRoles`, `allRoles`, and `allScopes` string lists; all optional and named-only, but at least one requirement is mandatory |
| `@schedule.FixedDelay` | Exact provider-owned exported method | `delay` duration string, required; `initialDelay` duration string and `continueOnError` Boolean, optional; all named-only |
| `@Service` | Type | `constructor` identifier, `name` string, and `aliases` string list, optional and named-only |

Annotations may be discovered on packages, types, functions, methods, variables, and constants. Each annotation definition determines which declaration kinds and invocation forms are legal.
