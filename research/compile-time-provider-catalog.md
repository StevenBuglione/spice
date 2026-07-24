# Compile-Time Provider Catalog and Constructor Injection Semantics

Date: 2026-07-24

## Question

After Spice has type-aware package loading and resolved annotations, what is the smallest dependency-injection foundation it should build so Spring-style constructor and factory injection is productive for Go developers without importing Spring's runtime container complexity or hiding generated behavior?

## Current delivery state

- There is no active implementation pull request.
- Issue #8 is the next `[agent-ready]` compiler slice: one typed Go program and stable symbol records.
- Issue #11 follows #8: annotations resolved against those exact typed symbols and CLI migration to the loaded program.
- The ready backlog contains two issues, leaving room for one additional bounded issue.
- Provider discovery must depend on both #8 and #11. Starting it earlier would duplicate package loading or rely on textual declaration names.

## Why this is the next coherent capability

Spring dependency injection derives collaborators from constructor or factory-method arguments and resolves them from bean definitions. Spice needs the same developer outcome, but it can perform provider discovery, signature validation, duplicate detection, and later graph construction before generated Go is compiled or the application starts.

A typed provider catalog is the bridge between the compiler front end and:

- generated constructor wiring;
- application lifecycle and shutdown;
- controllers, services, repositories, configuration, and starters;
- missing-provider and duplicate-provider diagnostics;
- dependency and module cycle detection;
- module ownership and allowed-dependency validation;
- test-specific application graphs.

Sources, accessed 2026-07-24:

- https://docs.spring.io/spring-framework/reference/core/beans/dependencies/factory-collaborators.html
- https://docs.spring.io/spring-framework/reference/core/beans/annotation-config/autowired.html
- https://docs.spring.io/spring-framework/reference/core/beans/definition.html

## Primary-source findings

### 1. Constructor and factory signatures should be the source of dependency edges

Spring describes dependency injection as objects declaring collaborators through constructor arguments, factory-method arguments, or mutable properties. It recommends constructor injection for required dependencies and reports constructor cycles as unresolvable.

Spice should deliberately support only constructor/factory-style required dependencies in its first graph. Field injection, setter injection, self injection, and runtime service-location would weaken static guarantees and make generated initialization harder to inspect.

Recommended rule:

```text
provider input parameter type -> required dependency key
provider primary result type  -> provided value key
```

A provider function remains ordinary Go and can be called directly in unit tests.

Sources:

- https://docs.spring.io/spring-framework/reference/core/beans/dependencies/factory-collaborators.html
- https://docs.spring.io/spring-framework/reference/core/beans/annotation-config/autowired.html

### 2. Use an explicit `@Bean` provider annotation rather than constructor-name inference

Go has no constructor declaration in the language. Inferring every `NewX` function would create accidental providers, make package refactors surprising, and prevent multiple intentional construction paths.

Spice should introduce the familiar Spring-facing marker:

```go
// @Bean
func NewUserService(repository UserRepository) *DefaultUserService {
    return &DefaultUserService{repository: repository}
}
```

The first provider issue should accept `@Bean` only on package-level functions. Methods require an instance receiver and therefore another provider and lifecycle decision before the method itself can run. Supporting `@Bean` methods immediately would recreate Spring configuration-class semantics before Spice has an application graph.

`@Service`, `@Controller`, and `@Configuration` should remain stereotype and policy metadata. They must not silently synthesize constructors or select a `New<Type>` function by naming convention.

Spring's `@Bean` declares one bean from a factory method and attaches scope, lifecycle, primary, qualifier, and ordering metadata. Spice can preserve that recognizable developer concept while implementing it as compile-time function analysis.

Source:

- https://docs.spring.io/spring-framework/docs/current/javadoc-api/org/springframework/context/annotation/Bean.html

### 3. Start with two unambiguous provider signature forms

The bootstrap catalog should accept exactly:

```go
func(dependencies...) T
func(dependencies...) (T, error)
```

Where:

- `T` is one non-error provided value;
- every input is a required dependency;
- a trailing `error` reports construction failure;
- zero-input providers are valid;
- parameter names do not affect resolution.

Reject initially:

- no result;
- more than one provided value;
- `error` as the only result;
- `error` in a non-final position;
- multiple error results;
- variadic providers;
- generic provider functions;
- provider methods;
- cleanup callbacks;
- parameter structs, result structs, groups, and optional dependencies.

This narrow contract is enough to build and test provider metadata while avoiding premature lifecycle and qualifier APIs.

Google Wire supports provider functions whose first result is the provided value, with optional cleanup and error results. Uber Fx/Dig support broader runtime forms, named values, groups, interface projection, scopes, and lifecycle annotations. Those capabilities are useful evidence for later slices, but implementing all of them in the first catalog would combine provider analysis, lifecycle, selection, graph generation, and runtime scopes into one unsafe issue.

Sources:

- https://pkg.go.dev/github.com/google/wire
- https://pkg.go.dev/go.uber.org/fx
- https://pkg.go.dev/go.uber.org/dig

### 4. Match provider values by exact Go type first

Spring initially selects candidates by type, then uses qualifiers, primary/fallback markers, names, and ordering to resolve or organize multiple candidates. Plain by-type injection fails when no unique candidate exists.

For the first Spice provider catalog:

- key providers and dependencies by exact `go/types.Type` identity in the single loaded program;
- use `types.Identical` for semantic matching;
- serialize a deterministic Spice type ID only for diagnostics, metadata, and generated artifacts;
- do not silently provide every interface implemented by a concrete result;
- do not select a candidate by declaration order.

Implicit interface projection is attractive but fragile. Adding a second implementation can unexpectedly change or break resolution. Wire requires an explicit interface binding for this reason. Spice should later add a typed binding mechanism rather than encode interface types in strings inside comments.

Sources:

- https://docs.spring.io/spring-framework/reference/core/beans/dependencies/factory-autowire.html
- https://docs.spring.io/spring-framework/docs/current/javadoc-api/org/springframework/context/annotation/Bean.html
- https://github.com/google/wire/blob/main/docs/faq.md

### 5. Duplicate exact outputs must fail deterministically

Two providers for the same exact type are ambiguous before qualifiers or named bindings exist. The catalog should collect both declarations and return one actionable diagnostic containing:

- the conflicting canonical type;
- both provider symbol IDs;
- both source positions;
- guidance that qualifier/binding support is not yet available.

It must never pick the first provider based on package, file, scan, or registration order.

Wire treats multiple providers for one type as an error. Spring likewise requires a unique candidate for a singular by-type dependency unless primary, qualifier, fallback, or explicit selection resolves it.

Sources:

- https://pkg.go.dev/github.com/google/wire
- https://github.com/google/wire/blob/main/docs/faq.md
- https://docs.spring.io/spring-framework/reference/core/beans/dependencies/factory-autowire.html

### 6. Reject constructor cycles; do not add lazy escape hatches yet

Spring can support some mutable-injection cycles but reports constructor-injection cycles as unresolvable. Spice's generated constructor graph should reject all cycles because every bootstrap dependency is required and immutable at construction time.

Cycle detection belongs to the following graph issue, not the provider-catalog issue. The catalog should preserve dependency type records and source positions so a later graph can render a complete cycle path.

Do not add lazy providers, provider functions as dependencies, service locators, or proxy-backed self injection until a real capability requires them and their effect on module verification is understood.

Source:

- https://docs.spring.io/spring-framework/reference/core/beans/dependencies/factory-collaborators.html

### 7. Default lifetime should eventually be one generated application singleton

Spring's default scope is singleton per container. Dig also instantiates a retained type at most once in its container. A generated Spice application graph should eventually construct one instance per provider for the application lifetime unless an explicit later scope says otherwise.

The first catalog should record no configurable scope and assume `application` as semantic metadata. It should not construct anything. Request, prototype/transient, session, WebSocket, or custom scopes require runtime ownership boundaries and should be separate features.

Sources:

- https://docs.spring.io/spring-framework/reference/core/beans/factory-scopes.html
- https://pkg.go.dev/go.uber.org/dig

### 8. Lifecycle cleanup should be designed separately from provider discovery

Wire supports cleanup functions and guarantees dependency-aware cleanup order. Fx provides startup hooks in registration order and shutdown hooks in reverse order with timeouts. Both show that resource cleanup is part of application lifecycle, not merely provider type discovery.

Spice should first build provider metadata for `T` and `(T, error)`. A later lifecycle RFC should choose an explicit Go-native contract, likely involving generated reverse dependency order and context-aware start/stop interfaces or hooks.

Do not accept cleanup return values in the first issue and then freeze an accidental public signature before lifecycle cancellation, timeouts, error aggregation, and partial-construction failure are specified.

Sources:

- https://pkg.go.dev/github.com/google/wire
- https://uber-go.github.io/fx/lifecycle.html
- https://pkg.go.dev/go.uber.org/fx

### 9. Keep annotations for metadata and typed Go for references

Wire's FAQ records that early comment directives were rejected because identifiers inside unstructured comments are opaque to ordinary Go rename and navigation tools. Spice uses comments for declaration metadata, but provider dependencies and outputs remain actual typed Go function signatures, so Go tooling continues to understand the important references.

This is the correct boundary:

```text
comment annotation: declares framework role (`@Bean`)
Go signature:       declares all dependency and result type references
Spice compiler:     validates metadata and builds deterministic records
```

Qualifiers or interface bindings should later use typed declarations or generated metadata where practical. Avoid public APIs that require package-qualified type names as annotation strings.

Source:

- https://github.com/google/wire/blob/main/docs/faq.md

### 10. Provider metadata must be immutable, deterministic, and source-positioned

A useful compiler-internal boundary after issue #11 is conceptually:

```go
package provider

type Catalog struct {
    Providers   []Provider
    Diagnostics []diagnostic.Diagnostic
}

type Provider struct {
    SymbolID     load.SymbolID
    Position     token.Position
    Output       types.Type
    OutputID     TypeID
    Dependencies []Dependency
    ReturnsError bool
}

type Dependency struct {
    Index    int
    Type     types.Type
    TypeID   TypeID
    Position token.Position
}

func Build(program *load.Program, annotations resolve.Result) Catalog
```

Exact exported names may change, but the contract should require:

- one existing per-command `load.Program`;
- resolved `@Bean` occurrences from issue #11;
- no package reload or AST reparse;
- no mutable process-global registry;
- deterministic provider and diagnostic ordering;
- live `types.Type` values limited to the owning program;
- stable readable type IDs for output and tests;
- all invalid signatures reported without panics or direct printing.

### 11. Canonical type IDs must not depend on local aliases or filesystem paths

Matching uses live `types.Type` identity. A stable display/serialization ID should use Go syntax qualified by package import path, not source aliases or absolute directories.

Examples:

```text
example.com/app/config.Config
*example.com/app/store.Store
map[string]example.com/app/users.User
[]example.com/app/events.Event
```

Use `types.TypeString` with a qualifier that returns `pkg.Path()`. Preserve named aliases as the semantic type information returned by the active Go version; do not flatten every named type to its underlying type for matching.

Source:

- https://pkg.go.dev/go/types#TypeString

### 12. Provider discovery should fail closed but accumulate independent errors

Required catalog diagnostics include:

- `@Bean` on a non-function target;
- `@Bean` on a method in the bootstrap contract;
- generic provider function;
- variadic provider function;
- zero results;
- multiple provided results;
- invalid error placement;
- duplicate exact output type;
- unresolved or ill-typed function signature inherited from the typed program.

Independent invalid providers should accumulate and sort by physical source position, annotation name, and diagnostic code. An ill-typed root package must block provider analysis entirely.

## Spring and Go capability comparison

| Concern | Spring behavior | Spice bootstrap direction |
|---|---|---|
| Provider declaration | `@Bean`, components, factory metadata | Explicit `@Bean` on a package-level Go function |
| Dependency declaration | Constructor/factory parameters, fields, setters | Function parameters only |
| Provided value | Bean/factory return type | One exact primary result type |
| Construction failure | Container/factory exception | Optional trailing Go `error` |
| Candidate resolution | Type plus primary, qualifier, fallback, name | Exact type only; ambiguity fails |
| Interface binding | Assignable bean candidates and qualifiers | Explicit typed binding deferred |
| Scope | Singleton default plus prototype/web/custom | Application singleton semantic default; scopes deferred |
| Cycles | Constructor cycles fail; mutable cycles sometimes possible | All constructor cycles will fail statically |
| Lifecycle | Init/destroy callbacks and context lifecycle | Separate Go-native lifecycle contract |
| Runtime mechanism | Bean definitions, post-processors, reflection/proxies | Typed compiler records and generated ordinary Go |

## Recommended implementation sequence

After issues #8 and #11 merge:

1. add built-in marker annotation `@Bean`, valid only on package-level functions;
2. add `compiler/provider` consuming the existing typed program and resolved annotations;
3. accept only `func(...) T` and `func(...) (T, error)`;
4. retain exact input and output `types.Type` values plus canonical import-path type IDs;
5. reject methods, generics, variadics, unsupported result forms, and duplicate exact outputs;
6. sort providers and diagnostics deterministically;
7. integrate provider validation into `spice verify` without generating or executing providers;
8. add positive, negative, deterministic, and CLI integration fixtures;
9. leave graph resolution, missing dependencies, cycles, roots, bindings, qualifiers, scopes, lifecycle, and code generation for later issues.

## Recommended next issue

Create one bounded issue dependent on #8 and #11:

> Add `@Bean` provider signature analysis and a deterministic compile-time provider catalog.

This issue should stop at trustworthy provider metadata and diagnostics. The following issue can resolve application roots and construct the dependency graph from that catalog.

## Deferred decisions

- application root declaration and eager versus reachable-only providers;
- interface bindings and assignability;
- qualifiers, names, primary, fallback, order, and groups;
- optional and collection dependencies;
- multiple result providers;
- provider methods and configuration receiver semantics;
- generic provider instantiation;
- lifecycle start/stop and cleanup return signatures;
- scopes beyond the application singleton;
- decorators, replacements, conditional providers, and starter contributions;
- test graph overrides;
- generated wiring layout and file naming;
- cross-module provider visibility;
- public stability of compiler-internal provider types.

## Recommendation

Adopt explicit `@Bean` package-level provider functions, derive required dependencies and one provided value from typed Go signatures, support only `T` and `(T, error)` results initially, match by exact type, reject duplicates deterministically, and defer lifecycle, scopes, qualifiers, interface bindings, and graph generation. This yields familiar Spring-style ergonomics while keeping dependencies visible to Go tooling and generated initialization statically inspectable.