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

Marker annotations such as `@Application`, `@Bean`, `@Configuration`, and `@Service` accept no arguments.

The bootstrap parser supports strings, integers, booleans, identifiers, and lists. The current validator checks the outer parsed kind only; it does not yet implement defaults, aliases, composed annotations, nested annotations, enum-like identifiers, or list element schemas.

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

`spice verify` validates catalog and graph metadata but does not execute providers or cleanup callbacks and does not generate application wiring yet. Exact output types must have one provider; Spice rejects duplicates rather than choosing by declaration order or implicit interface assignability. Interface bindings, qualifiers, scopes, cleanup invocation, startup/shutdown hooks, optional values, groups, and collection injection remain explicit future capabilities.

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
| `@Application` | Function | None |
| `@Bean` | Package-level function | None |
| `@Configuration` | Type | None |
| `@Controller` | Type | `prefix` string, optional, named-only |
| `@Get` | Method | `path` string, required, named or positional |
| `@Post` | Method | `path` string, required, named or positional |
| `@Service` | Type | None |

Annotations may be discovered on packages, types, functions, methods, variables, and constants. Each annotation definition determines which declaration kinds and invocation forms are legal.
