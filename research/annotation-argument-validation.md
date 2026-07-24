# Annotation Argument Validation Research

Date: 2026-07-24

## Question

What is the smallest annotation argument-validation contract Spice should implement after target validation so that built-in annotations are safe and ergonomic now without blocking future Spring-style composed annotations, aliases, starters, or custom annotation definitions?

## Context

PR #4 introduces typed annotation definitions with argument names, accepted parsed value kinds, required status, and positional support. The parser already represents strings, integers, booleans, identifiers, and lists. The next compiler step must turn that metadata into deterministic, source-positioned validation.

This work is foundational. Controller routing, configuration, security, data, scheduling, events, caching, and starter annotations all depend on rejecting misspelled arguments and incompatible values before generation.

## Primary sources

### Java annotation contract

The Java Language Specification defines annotation invocations as element-value associations. Every element without a default must be supplied, each named element may appear only once, and a single-element annotation has shorthand syntax conventionally associated with an element named `value`.

- Java Language Specification SE 25, Chapter 9: https://docs.oracle.com/javase/specs/jls/se25/html/jls-9.html
- Java Language Specification SE 25 index: https://docs.oracle.com/javase/specs/jls/se25/html/index.html

Java permits a broader value domain than Spice currently parses, including primitive constants, strings, enum constants, class literals, nested annotations, and arrays. Spice should not copy that domain prematurely. Its current smaller grammar is enough for the first vertical slices and is easier to validate deterministically.

### Spring annotation ergonomics

Spring annotations commonly expose optional attributes with defaults and required semantic attributes. Spring also supports composed annotations and aliases. `@AliasFor` can make two attributes interchangeable or forward an attribute into a meta-annotation, and composed annotations such as `@RestController` combine other annotations.

- Spring Framework 7.0.8 `@Bean`: https://docs.spring.io/spring-framework/docs/current/javadoc-api/org/springframework/context/annotation/Bean.html
- Spring Framework `@AliasFor`: https://docs.spring.io/spring-framework/docs/current/javadoc-api/org/springframework/core/annotation/AliasFor.html
- Spring reference, meta-annotations and composed annotations: https://docs.spring.io/spring-framework/reference/core/beans/classpath-scanning.html#beans-meta-annotations

These features imply that Spice will eventually need an argument-resolution stage that can apply defaults, aliases, and composition before generation. They do not need to be implemented in the first argument validator.

### Spice constraints

Spice source must remain valid Go, diagnostics must preserve source positions, output must be deterministic, and the runtime should not perform late metadata validation. The existing public parser model and typed definition model should remain the single source for this phase.

Relevant repository artifacts:

- `annotation/annotation.go`
- `annotation/definition.go` from PR #4
- `compiler/parser`
- `compiler/scan`
- `compiler/validate`
- `docs/annotations.md`
- RFC 0001

## Recommended immediate contract

The next implementation should validate only the syntax-to-definition relationship. It should not resolve Go symbols or implement composed annotations.

For each parsed annotation:

1. Resolve its typed definition from the registry.
2. Reject arguments when the definition declares none.
3. Reject unknown named arguments.
4. Reject duplicate semantic assignment, including a positional value and a named value that both map to the same definition argument.
5. Match positional values only to an argument explicitly marked `Positional`.
6. Reject positional values when zero or more than one positional argument definition exists.
7. Reject more than one positional value in the bootstrap contract.
8. Require every argument marked `Required` unless it was supplied positionally.
9. Check the parsed value kind against the argument's accepted kinds.
10. Emit all diagnostics in deterministic file, line, column, annotation, and argument order.

The validator should accumulate independent diagnostics rather than stopping at the first error in a file.

## Built-in ergonomic decisions

After PR #4 merges, the built-in definitions should support:

```go
// @Controller(prefix="/users")
type UserController struct{}

// @Get(path="/{id}")
func (UserController) GetUser() {}

// @Get("/{id}")
func (UserController) GetUserCompact() {}
```

Recommendations:

- `Controller.prefix`: optional string, named only. Avoid an ambiguous positional prefix because controllers will likely gain other attributes later.
- `Get.path`: required string, positional or named.
- `Post.path`: required string, positional or named.
- `Application`, `Configuration`, and `Service`: reject all arguments.

This gives route declarations concise Spring-like ergonomics without making every future annotation's first field implicitly positional.

## Diagnostics

Diagnostics should be actionable and stable. Examples:

```text
controller.go:3:1: annotation @Controller does not define argument "path"; available argument: prefix
controller.go:8:1: annotation @Get requires argument "path"
controller.go:13:1: annotation @Get argument "path" requires string, got integer
controller.go:18:1: annotation @Get assigns argument "path" more than once
service.go:3:1: annotation @Service does not accept arguments
```

Do not include unrelated source text or full file contents in an error.

## Deliberate deferrals

The first validator should not implement:

- Default value storage or materialization.
- Alias resolution or `@AliasFor` equivalents.
- Composed/meta-annotations.
- Nested annotation values.
- Go type or symbol references.
- Enum-like identifier validation.
- List element schemas or heterogeneous-list policy.
- Annotation repeatability enforcement.
- Loading third-party annotation definitions.
- Code generation.

These should be separate phases because they change metadata resolution rather than basic invocation validity.

## Compatibility implications

### Defaults

The current `Required` flag is sufficient to validate omission, but it does not preserve an actual default. A later definition-model change should add an optional typed default value. The validator created now should expose a clean boundary so default application can be inserted after validation without rewriting parser rules.

### Aliases and composition

Future aliases should canonicalize supplied arguments before duplicate and required checks. Therefore, keep argument validation behind a function accepting one definition and one parsed annotation instead of spreading checks throughout scanning or CLI code.

### Lists

The parser supports lists, while `ArgumentDefinition.Kinds` currently validates only the outer `list` kind. A later schema may add element-kind constraints. No current built-in needs a list, so the first issue should not invent that API yet.

### Identifiers

Identifiers are currently opaque parsed values. Future phases may interpret them as enum-like constants, type names, package-qualified symbols, or policy names. Basic validation should check only `KindIdentifier` and preserve the raw identifier.

## Security and correctness considerations

- Fail closed on unknown arguments and incompatible kinds.
- Never silently ignore misspellings such as `prefx` or `paht`.
- Do not coerce integers, booleans, or identifiers into strings.
- Avoid process-global mutable registries.
- Avoid nondeterministic diagnostics from map iteration.
- Bound validation work linearly in annotations plus arguments.
- Preserve exact source positions already collected by parsing and scanning.

## Performance expectation

Registry lookup is effectively constant time. Per-annotation validation should be linear in the number of supplied and defined arguments. Annotation definitions are expected to have very small argument counts, so clarity and deterministic behavior matter more than micro-optimization.

## Implementation sequencing

1. Merge PR #4 and close issue #2.
2. Add focused argument validation to `compiler/validate` using the registry already introduced.
3. Add built-in positional metadata for `Get.path` and `Post.path`.
4. Extend CLI tests and invalid fixtures.
5. Update `docs/annotations.md`.
6. Keep default values, aliases, composition, repeatability, and custom-definition loading as later bounded issues.

## Decision

Proceed with one bounded implementation issue for built-in annotation argument validation. This is the highest-value next compiler task because it completes the first typed annotation invocation contract and prevents every later Spring capability from building on unchecked string metadata.
