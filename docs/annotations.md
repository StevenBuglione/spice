# Spice Annotation Syntax

## Canonical form

Spice annotations are declaration comments:

```go
// @Controller(prefix="/users")
type UserController struct{}
```

The parser also accepts `//@Controller`, but `gofmt` inserts a space after `//`, making `// @Controller` the canonical documented form.

## Names

Built-in annotation names are unqualified:

```go
// @Service
// @Configuration
```

The parser accepts qualified names for future starter and third-party definitions:

```go
// @security.Authorize(roles=["admin"])
```

During the current bootstrap, `spice verify` fails closed on unknown annotations. A qualified annotation is syntactically valid, but it must have a registered typed definition before verification succeeds. User-defined definition loading is planned; unknown annotations are not silently accepted because doing so would hide spelling mistakes and missing starters.

## Arguments

Spice supports positional and named arguments:

```go
// @Profile("production")
// @Controller(prefix="/users")
```

Supported bootstrap values:

- Strings: `"users"`
- Integers: `3`
- Booleans: `true`
- Identifiers: `exponential`
- Lists: `["admin", "operator"]`

Named arguments must follow positional arguments, and duplicate named arguments are rejected.

Typed annotation definitions include argument names, accepted value kinds, required status, and positional support. Argument schema enforcement is a later compiler phase; the current verifier enforces annotation existence and declaration targets.

## Built-in definitions and targets

Spice ships the following initial definitions:

| Annotation | Allowed target | Defined arguments |
|---|---|---|
| `@Application` | Function | None |
| `@Configuration` | Type | None |
| `@Controller` | Type | `prefix` string, optional |
| `@Get` | Method | `path` string, required |
| `@Post` | Method | `path` string, required |
| `@Service` | Type | None |

Examples:

```go
// @Application
func main() {}

// @Controller(prefix="/users")
type UserController struct{}

// @Get(path="/{id}")
func (UserController) GetUser() {}

// @Service
type UserService struct{}
```

Invalid placement fails with a source-positioned, actionable diagnostic:

```text
controller.go:3:1: annotation @Controller cannot target function "NewController"; allowed target: type
```

The definition registry is immutable by construction and uses deterministic constant-time name lookup. This model is public so future starters and custom annotation tooling can use the same metadata without a process-global mutable registry.

## Declaration association

Annotations may be discovered on packages, types, functions, methods, variables, and constants. Each annotation definition determines which of those declaration kinds are legal.

## Why comments

Raw `@Controller` is not valid Go syntax. Declaration comments preserve standard Go parsing, formatting, tests, debugging, and static tooling while allowing Spice to provide compiler-adjacent diagnostics and generation.
