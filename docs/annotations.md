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

Marker annotations such as `@Application`, `@Configuration`, and `@Service` accept no arguments.

The bootstrap parser supports strings, integers, booleans, identifiers, and lists. The current validator checks the outer parsed kind only; it does not yet implement defaults, aliases, composed annotations, nested annotations, enum-like identifiers, or list element schemas.

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
| `@Configuration` | Type | None |
| `@Controller` | Type | `prefix` string, optional, named-only |
| `@Get` | Method | `path` string, required, named or positional |
| `@Post` | Method | `path` string, required, named or positional |
| `@Service` | Type | None |

Annotations may be discovered on packages, types, functions, methods, variables, and constants. Each annotation definition determines which declaration kinds and invocation forms are legal.
