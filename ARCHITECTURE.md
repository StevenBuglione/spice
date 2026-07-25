# Spice Architecture

## Product thesis

Spice is a Go-native enterprise application platform inspired by the useful outcomes of Spring Boot and Spring Modulith. It seeks capability parity where that improves developer productivity, but it does not attempt implementation parity with JVM mechanisms.

## Core pipeline

```text
Valid Go source
  -> Go AST and type information
  -> Spice annotation parser
  -> typed Spice intermediate representation
  -> module and dependency graphs
  -> static validation
  -> deterministic generated Go
  -> standard Go compiler
```

## Non-negotiable rules

1. Application source remains valid Go.
2. Generated behavior is ordinary, inspectable Go.
3. Constructor injection is preferred over field injection.
4. Compile-time generation is preferred over reflection-heavy runtime discovery.
5. Interfaces and explicit wrappers replace subclass proxies.
6. Architecture enforcement is a first-class product feature.
7. Each feature must include executable tests and an observable runnable path.
8. The standard library remains the default dependency choice.
9. External dependencies require an explicit rationale.
10. No implementation may claim success without running the repository verification command.

## Major subsystems

### Compiler front end

- Annotation lexical and syntactic parser.
- Go package loading and AST inspection.
- Symbol and type resolution.
- Source-positioned diagnostics.
- Annotation target and argument validation.

### Application model

- Components and providers.
- Dependency graph and lifecycle.
- Application modules and named interfaces.
- Configuration ownership.
- Routes, event contracts, and transaction boundaries.

### Code generation

- Dependency wiring.
- HTTP adapters.
- Cross-cutting decorators.
- Configuration binders.
- Metadata and documentation.

Generation is split into a pure in-memory plan and a guarded filesystem
application. The pure renderer consumes the immutable application IR, emits
canonical target-scoped Go plus SHA-256 ownership metadata, and performs no
filesystem or network mutation.

### Runtime

The runtime should stay small. Its responsibilities are application lifecycle, generated registry execution, request scopes, shutdown, and integrations that cannot be resolved at compile time.

### Verification

`spice verify` will eventually enforce:

- Valid annotation syntax and targets.
- No unresolved dependency providers.
- No dependency cycles.
- No module-level cycles.
- No access to another module's internals.
- Only declared module dependencies.
- Valid route and event contracts.
- Valid transaction and configuration ownership.
- Generated-code freshness.

## Capability parity policy

Spice tracks Spring Boot and Spring Modulith capabilities in `docs/spring-coverage.md`. Each capability must be classified as one of:

- `planned`: a Spice-native implementation is intended.
- `in-progress`: active implementation exists.
- `available`: supported and tested.
- `integration`: delegated to a mature Go library through a Spice starter.
- `not-planned`: intentionally excluded with rationale.

The target is broad practical coverage, not class-by-class porting.
