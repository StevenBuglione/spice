# Spice Annotation Syntax

## Canonical form

Spice annotations are declaration comments:

```go
// @Controller(prefix="/users")
type UserController struct{}
```

The parser also accepts `//@Controller`, but `gofmt` inserts a space after `//`, making `// @Controller` the canonical documented form.

## Names

Built-in names are unqualified. Qualified names are reserved for future starters and third-party definitions. During bootstrap, `spice verify` fails closed on unknown annotations.

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

Marker annotations such as `@Application`, `@Bean`, `@Configuration`, `@Service`, `@OnStart`, and `@OnStop` accept no arguments.

The bootstrap parser supports strings, integers, booleans, identifiers, and lists. The current validator checks the outer parsed kind only; it does not yet implement defaults, aliases, composed annotations, nested annotations, enum-like identifiers, or list element schemas.

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

Argument-free, method-only `@OnStart` and `@OnStop` select explicit methods for future generated lifecycle orchestration. A hook must have the exact non-variadic form `func(receiver)(context.Context) error`, and its receiver must be semantically identical to exactly one valid `@Bean` output.

Aliases to the exact receiver, `context.Context`, and `error` types are accepted. Pointer/value convenience, assignability, interface implementation, structural context lookalikes, method promotion, duplicate roles, and stop-only components are rejected.
The compiler records deterministic typed metadata only. `spice verify` never invokes providers, cleanup callbacks, or lifecycle methods. Generated applications use the public `lifecycle.Coordinator` for the state machine, dependency-order start, reverse stop/cleanup, startup rollback, deterministic error joining, idempotent stop, and run/wait/shutdown composition. Concrete hook calls remain direct generated method values.

Generated `Run` accepts the caller's run context and a caller-supplied shutdown-context factory. This keeps operating-system signals and fresh shutdown deadlines in the command while allowing the framework to stop gracefully after cancellation without inventing a hidden background context.

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
| `@Configuration` | Type | None |
| `@Controller` | Type | `prefix` string, optional, named-only |
| `@Get` | Method | `path` string, required, named or positional |
| `@OnStart` | Method | None |
| `@OnStop` | Method | None |
| `@Post` | Method | `path` string, required, named or positional |
| `@Service` | Type | None |

Annotations may be discovered on packages, types, functions, methods, variables, and constants. Each annotation definition determines which declaration kinds and invocation forms are legal.
