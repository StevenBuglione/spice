# Spice Framework for Go

Spice is an opinionated, compile-time application platform for Go. Its goal is to bring the breadth, productivity, and operational completeness associated with Spring Boot together with Spring Modulith-style architectural enforcement—without importing JVM runtime magic into Go.

## Product direction

Spice is designed around five commitments:

1. **Broad application-platform coverage.** The roadmap intentionally covers web APIs, configuration, dependency injection, validation, security, data access, transactions, messaging, scheduling, observability, testing, and modular architecture.
2. **Excellent developer ergonomics.** Common application behavior should be easy to express, errors should point to source, generated behavior should be inspectable, and the happy path should be obvious.
3. **Valid Go source.** Spice annotations are ordinary Go comments such as `// @Controller(prefix="/users")`, so standard Go tools continue to parse the project.
4. **Compile-time enforcement.** Wiring, annotation validation, and module rules should fail before deployment whenever possible.
5. **Runnable software, not paper architecture.** Every implementation change must compile, execute its relevant smoke path, and pass tests before it is considered complete.

## Current bootstrap

The repository currently provides:

- A parser for Spice annotations.
- A source scanner that associates annotations with Go declarations.
- A runnable `spice` CLI with `version`, `annotations`, and `verify` commands.
- A runnable HTTP example with tests.
- A deterministic verification script and GitHub Actions workflow.
- Product, architecture, annotation, and Spring-coverage documents.
- Three autonomous-agent prompts for research, implementation, and independent verification.

## Annotation syntax

Both of these are accepted:

```go
//@Controller(prefix="/users")
// @Controller(prefix="/users")
type UserController struct{}
```

`gofmt` canonicalizes the second form, so official Spice documentation uses:

```go
// @Controller(prefix="/users")
type UserController struct{}
```

Qualified annotations are available for collisions:

```go
// @security.Authorize(roles=["admin"])
```

## Run it

```bash
make verify
go run ./cmd/spice version
go run ./cmd/spice annotations ./examples/hello-world
go run ./cmd/spice verify ./...
go run ./examples/hello-world -check
```

To start the example HTTP server:

```bash
go run ./examples/hello-world -listen :8080
curl http://localhost:8080/users/42
```

## Repository map

- `annotation/`: public annotation model.
- `compiler/parser/`: annotation parser.
- `compiler/scan/`: Go source scanning and declaration association.
- `cmd/spice/`: CLI entry point.
- `internal/cli/`: CLI implementation.
- `examples/`: executable reference applications.
- `docs/`: user and product documentation.
- `rfcs/`: proposed designs.
- `adrs/`: accepted architectural decisions.
- `agent/prompts/`: scheduled-agent operating prompts.

## Status

Spice is pre-alpha. The immediate goal is to prove the compiler and modular architecture foundations before expanding into the larger Spring Boot capability surface.
