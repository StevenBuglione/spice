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
// @Transactional
```

Third-party or ambiguous annotations may be qualified:

```go
// @security.Authorize(roles=["admin"])
```

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

## Declaration association

Annotations may target packages, types, functions, methods, variables, and constants. Future compiler phases will validate whether a specific annotation is legal on a target.

## Why comments

Raw `@Controller` is not valid Go syntax. Declaration comments preserve standard Go parsing, formatting, tests, debugging, and static tooling while allowing Spice to provide compiler-adjacent diagnostics and generation.
