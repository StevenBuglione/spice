# Go-Native Interface Bindings, Qualifiers, and Ambiguity

Date: 2026-07-24

## Question

After Spice has a typed provider catalog and an exact-type dependency graph, how should it support interface-oriented application design, multiple implementations, qualifiers, primary/fallback selection, and collections without importing Spring's runtime candidate-selection complexity or weakening Go-native refactoring and modular boundaries?

## Decision summary

The first production-ready Spice dependency model should keep exact provider-output identity and use ordinary Go provider functions as the explicit binding mechanism.

```go
// @Bean
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
    return &PostgresRepository{db: db}
}

// @Bean
func RepositoryBinding(repository *PostgresRepository) Repository {
    return repository
}
```

The second provider is ordinary valid Go. The Go compiler proves that `*PostgresRepository` implements `Repository`; Spice only records that the provider's exact output type is `Repository`. Issue #17 can then resolve an exact `Repository` dependency without any assignability search or runtime container behavior.

For multiple semantically distinct values of the same third-party or primitive type, prefer named wrapper types and explicit aggregate providers over string qualifiers, parameter-name matching, implicit groups, or primary/fallback priority rules.

This baseline intentionally does **not** introduce `@Bind`, `@Qualifier`, `@Primary`, `@Fallback`, string names, automatic interface projection, or collection injection. Those features should be added only when concrete application evidence shows that ordinary typed providers and wrappers are insufficient.

## Pipeline relationship

This research does not change the active issue #8 / PR #15 delivery lane or any current acceptance criteria.

It reinforces the sequencing already encoded in the ready backlog:

1. issue #11 resolves annotations against exact typed symbols;
2. issue #13 builds exact provider metadata;
3. issue #17 builds an exact-type provider graph and deliberately refuses implicit concrete-to-interface selection.

The findings define the next interface-oriented design boundary after those slices. They also explain why issue #17's fail-closed exact-type rule is the correct bootstrap behavior rather than a temporary correctness defect.

## Primary sources and status

Sources were accessed on 2026-07-24.

### Spring Framework

- Dependency injection and constructor/factory-method collaborators:
  - https://docs.spring.io/spring-framework/reference/core/beans/dependencies/factory-collaborators.html
- Autowiring modes, ambiguity, and the recommendation to prefer explicit wiring when clarity matters:
  - https://docs.spring.io/spring-framework/reference/core/beans/dependencies/factory-autowire.html
- Constructor and factory-method autowiring behavior:
  - https://docs.spring.io/spring-framework/reference/core/beans/annotation-config/autowired.html
- Qualifier narrowing semantics:
  - https://docs.spring.io/spring-framework/reference/core/beans/annotation-config/autowired-qualifiers.html
- `@Primary` and `@Fallback` selection:
  - https://docs.spring.io/spring-framework/reference/core/beans/annotation-config/autowired-primary.html
- `@Bean` candidate, qualifier, ordering, and lifecycle metadata:
  - https://docs.spring.io/spring-framework/docs/current/javadoc-api/org/springframework/context/annotation/Bean.html

Spring Framework is Apache License 2.0. Spice uses Spring as capability evidence, not as an implementation template. Spring's current container performs candidate selection over runtime bean definitions and supports names, qualifiers, primary/fallback precedence, optional dependencies, and multi-value injection. Spice should preserve the useful developer outcomes while using compile-time typed metadata and generated Go.

### Go language and type checker

- Type identity, interface implementation, and assignability:
  - https://go.dev/ref/spec
- `go/types.AssignableTo`, `Implements`, and `Identical`:
  - https://pkg.go.dev/go/types
- Effective Go interface guidance and compile-time interface checks:
  - https://go.dev/doc/effective_go

The Go project uses a BSD-style license. Go interface implementation is structural and implicit. A concrete type can accidentally or intentionally satisfy multiple interfaces without declaring that relationship. That is valuable at ordinary call sites, but dependency-graph activation should not silently turn every assignable provider into a framework binding.

### Google Wire

- Public directives and explicit interface binding:
  - https://pkg.go.dev/github.com/google/wire
- Repository and project status:
  - https://github.com/google/wire

Wire's `Bind` maps one concrete type to an interface explicitly, and its internal binding record requires the provided type to be assignable to the interface. Wire is Apache License 2.0 and was archived by its owner on 2025-08-25. It is useful precedent for explicit compile-time binding, but it should not become a Spice dependency.

### Uber Fx and Dig

- Fx annotations, `As`, `From`, names, groups, and lifecycle:
  - https://pkg.go.dev/go.uber.org/fx
- Fx repository and license:
  - https://github.com/uber-go/fx
- Dig names, groups, optional dependencies, and runtime graph behavior:
  - https://pkg.go.dev/go.uber.org/dig

Fx and Dig are MIT licensed. Fx v1.24.0 and Dig v1.19.0 were released on 2025-05-13. They demonstrate demand for interface projection, string names, groups, decorators, and overrides, but they implement those capabilities through a runtime reflection container. Their string tags are flexible but weaker under rename and static analysis than ordinary Go types.

## Findings and decisions

### 1. Keep the provider graph keyed by exact output type

Issue #13's provider catalog and issue #17's dependency graph should continue to use exact semantic Go types as dependency keys.

A dependency on:

```go
type Repository interface {
    Find(context.Context, string) (User, error)
}
```

must be satisfied by a provider whose declared output type is exactly `Repository`.

A provider whose output is `*PostgresRepository` is not the same graph key, even when `*PostgresRepository` implements `Repository`.

This is stricter than ordinary Go assignment at a call site, but it is the correct framework contract because it makes every application-level binding visible and stable.

Benefits:

- adding an unrelated concrete provider cannot silently change dependency resolution;
- multiple structural implementers do not create order-dependent behavior;
- module-internal implementations are not exposed merely because they happen to satisfy a public interface;
- the graph remains an exact hash lookup rather than a repeated assignability search;
- diagnostics can state precisely which binding is missing;
- generated calls remain inspectable and unsurprising.

### 2. Use ordinary provider adapters for interface bindings

The initial interface-binding mechanism should be an ordinary provider function.

```go
package bootstrap

import (
    "example.com/app/contracts"
    "example.com/app/postgres"
)

// @Bean
func UserRepositoryBinding(
    implementation *postgres.UserRepository,
) contracts.UserRepository {
    return implementation
}
```

This pattern has several important properties:

- it is valid Go and directly unit-testable;
- the Go compiler checks assignability;
- editors understand rename, navigation, completion, and references;
- no annotation argument needs to encode a Go type as a fragile string;
- issue #13 already knows how to model the function signature;
- issue #17 already knows how to create the concrete-to-interface edge;
- generated code does not need a special binding instruction;
- the binding has a source position and stable symbol identity;
- duplicate interface bindings become the existing duplicate exact-output error.

The adapter's runtime cost is one ordinary function call during bootstrap. A future generator may inline trivial adapters only as a transparent optimization, but correctness must never depend on recognizing function bodies.

### 3. Returning the interface directly is also valid, with a trade-off

A constructor may return only the interface:

```go
// @Bean
func NewUserRepository(db *sql.DB) UserRepository {
    return &PostgresRepository{db: db}
}
```

This is concise and exact. It is appropriate when no other provider needs the concrete type.

The trade-off is that the concrete value is not separately available to the graph. When both identities are needed, use two providers:

```go
// @Bean
func NewPostgresRepository(db *sql.DB) *PostgresRepository { ... }

// @Bean
func UserRepositoryBinding(value *PostgresRepository) UserRepository {
    return value
}
```

Spice should document this choice rather than trying to provide one constructor result automatically under every interface it implements.

### 4. Do not automatically search assignable providers

A tempting rule is:

```text
If no exact provider exists for interface I,
find every provider output assignable to I.
If exactly one exists, use it.
```

Spice should not adopt this as the default.

Go interfaces are intentionally implicit. Automatic graph binding would therefore make framework behavior depend on structural relationships that may be unrelated to application composition.

Example:

```go
type Closer interface { Close() error }
```

Many resources can implement `Closer`. Adding a new provider for a cache, file, database, or network client could turn a previously valid graph into an ambiguity, even though the consuming code and intended binding did not change.

Automatic assignability also complicates modular enforcement. A concrete type from another module could become an injection candidate for a consumer interface without an explicit composition decision or public binding package.

Failing closed with a missing exact binding produces a better correction:

```text
provider example.com/app/service.NewService requires
example.com/app/contracts.Repository, but no provider supplies that exact type;
add an ordinary @Bean adapter that returns the interface
```

### 5. Do not add string qualifiers as the first ambiguity mechanism

Spring qualifiers narrow candidates after type matching. Fx and Dig names use string tags. These capabilities solve real problems, but strings are a poor first choice for Spice because they:

- are not ordinary Go identifiers;
- are weaker under rename and code navigation;
- can drift between provider and parameter metadata;
- require a second identity system beside Go types;
- complicate module ownership and compatibility rules;
- create additional collision, escaping, and normalization questions;
- tend to push errors from Go compilation into framework validation.

For two semantically different values of the same external type, prefer distinct named wrapper types.

```go
type ReadDatabase struct {
    *sql.DB
}

type WriteDatabase struct {
    *sql.DB
}

// @Bean
func NewReadDatabase(config ReadDatabaseConfig) (ReadDatabase, error) { ... }

// @Bean
func NewWriteDatabase(config WriteDatabaseConfig) (WriteDatabase, error) { ... }
```

Consumers then request the exact semantic type:

```go
// @Bean
func NewQueryService(database ReadDatabase) *QueryService { ... }
```

This is verbose only where the application truly has multiple semantic identities. In return, the distinction is visible to Go tooling, tests, module APIs, and generated code.

### 6. Prefer semantic wrapper types over aliases

A type alias does not create a distinct dependency key:

```go
type ReadDatabase = *sql.DB
```

This remains identical to `*sql.DB` and cannot distinguish multiple values.

Use a distinct defined type or wrapper struct instead.

A wrapper struct is generally the clearest option for pointer-like third-party values because it can expose the embedded API while retaining a distinct named identity:

```go
type ReadDatabase struct {
    *sql.DB
}
```

Spice documentation should explain this explicitly so developers do not expect aliases to behave like qualifiers.

### 7. Reject ambiguity instead of selecting primary or fallback implicitly

Spring's `@Primary` and `@Fallback` provide useful runtime candidate precedence. Spice should not introduce equivalent priority metadata in the foundational graph.

A graph with two providers for the same exact output type is ambiguous and should remain invalid.

```go
// both return Repository: reject
func PostgresBinding(...) Repository
func MemoryBinding(...) Repository
```

The application must make the choice explicit in ordinary Go composition. Options include:

- remove the inactive binding from the selected application provider set once roots/activation exist;
- expose distinct semantic wrapper types;
- select one implementation in a dedicated binding provider;
- use a future explicit replacement/test-override phase before graph construction.

A primary/fallback feature may be justified later for starter defaults, but it must be an explicit activation transform with deterministic precedence and diagnostics. It must not be hidden inside exact graph lookup.

### 8. Keep collection injection explicit initially

Spring can inject every matching bean into arrays, collections, and maps. Fx and Dig provide value groups. These are convenient for plugin lists, handlers, health contributors, and middleware.

The foundational Spice model should use an ordinary aggregate provider:

```go
type HTTPHandlers []HTTPHandler

// @Bean
func NewHTTPHandlers(
    users *UsersHandler,
    health *HealthHandler,
) HTTPHandlers {
    return HTTPHandlers{users, health}
}
```

Advantages:

- membership is explicit;
- order is explicit;
- the aggregate has one exact type;
- missing members produce normal provider diagnostics;
- module dependencies remain inspectable;
- adding a provider elsewhere does not mutate the collection silently.

Future group support may be valuable for starter ecosystems, but it must define deterministic ordering, duplicate behavior, module visibility, empty-group behavior, and test overrides before implementation. Dig explicitly documents that value groups are unordered, which is not sufficient for Spice's deterministic generation contract.

### 9. Keep optional dependencies explicit in Go types

Do not infer optionality from pointer types, nilability, missing providers, parameter names, or zero values.

A pointer often means mutability or identity, not optional dependency. Silently injecting `nil` would move a configuration error into runtime behavior.

A future optional-dependency feature should use an explicit typed contract, for example a small generic value such as:

```go
type Optional[T any] struct {
    Value T
    Present bool
}
```

That design needs separate research on zero values, interface values, configuration conditions, generated construction, and ergonomics. It is not part of the interface-binding slice.

### 10. Put bindings at the composition boundary

A useful modular-monolith rule is:

- consumer modules define the smallest interfaces they need;
- implementation modules expose concrete constructors or public implementation types only when appropriate;
- the application composition/root package declares which implementation satisfies which consumer contract.

Example layout:

```text
internal/application/bootstrap
internal/orders/api
internal/orders/service
internal/persistence/postgres
```

The binding provider belongs in the bootstrap/composition area, not inside the consumer's domain package and not hidden in a runtime registry.

This supports Spring Modulith-style boundary enforcement:

- the consumer depends on its interface package;
- the concrete implementation dependency is visible at composition time;
- generated wiring can be attributed to the application root;
- changing implementations does not require changing consumer code;
- module diagrams can render the binding as an explicit edge.

The later module verifier must still enforce Go imports and Spice module rules. An interface binding must never authorize imports of another module's internal packages that are otherwise forbidden.

### 11. Preserve both concrete and interface graph identities when declared

When an application declares both:

```go
func NewPostgresRepository(...) *PostgresRepository
func RepositoryBinding(*PostgresRepository) Repository
```

Spice should model two provider nodes and one dependency edge:

```text
NewPostgresRepository -> *PostgresRepository
RepositoryBinding requires *PostgresRepository -> Repository
```

Consumers of the concrete type depend on the first provider. Consumers of the interface depend on the binding provider. Both receive the same underlying instance because the binding provider returns the constructed concrete value.

This is a transparent graph, unlike an implicit projection that makes one provider appear under several keys without a source-level declaration.

### 12. Do not inspect provider bodies to prove bindings

Spice should validate provider signatures and rely on Go type checking. It must not inspect an adapter body's return expression to determine whether the provider is a “real” binding.

The following are all ordinary providers from the compiler's perspective:

```go
func Binding(value *Implementation) Interface { return value }
func Decorated(value *Implementation) Interface { return &loggingWrapper{value} }
func Conditional(value *Implementation, config Config) Interface { ... }
```

Body inspection would create a fragile semantic distinction, complicate overlays and generics, and encourage optimizer logic in the validation phase. The provider's declared input and output types are the contract.

### 13. Keep binding metadata serialization-friendly without replacing live types

A future graph or documentation model may classify a provider as an interface adapter for presentation, but correctness should remain based on live `go/types.Type` values from the owning program.

Stable summaries should use canonical package-path-qualified type IDs, for example:

```text
output: example.com/app/contracts.Repository
input:  *example.com/app/postgres.Repository
```

Do not serialize pointer addresses, temporary fixture paths, local import aliases, or reflection names.

A presentation-only “binding” classification can be inferred from the signature when:

- there is exactly one input;
- the output is an interface;
- the input type is assignable to the output interface;
- the provider has no additional results except optional `error` under the normal provider rules.

That inference must not change graph semantics or skip the provider call.

### 14. Exact lookup is simpler and faster

With exact provider keys, graph resolution remains:

```text
providerByExactType[dependencyType]
```

Expected complexity remains effectively constant per dependency plus deterministic sorting.

Automatic interface candidate search would require either:

- scanning every provider output for every interface dependency; or
- maintaining secondary assignability indexes whose invalidation and generic/interface semantics are more complex.

Performance is not the primary reason for the decision, but the exact model aligns correctness, determinism, and efficiency.

### 15. The baseline migration story is stable

With explicit adapters:

- adding a new concrete provider does not alter existing interface resolution;
- adding methods to a concrete type does not create new framework bindings;
- adding a competing implementation does not break a graph until an explicit duplicate interface provider is declared;
- renaming a binding function does not change its output identity but remains visible in diagnostics and generated source;
- moving an interface package changes canonical type identity and produces ordinary compile-time and Spice diagnostics;
- replacing an implementation is a local composition change.

This is a stronger compatibility story than candidate selection based on “the only assignable provider currently present.”

## Rejected initial designs

### Automatic single-implementer binding

Rejected because the result can change when an unrelated provider is added, structural interface implementation is intentionally implicit, and module boundaries become harder to reason about.

### `@Bind(interface = "example.com/contracts.Repository")`

Rejected for the baseline because a string-encoded Go type is weaker than an ordinary function signature and requires custom resolution, escaping, imports, and rename behavior.

### `@Qualifier("read")` string matching

Rejected for the baseline because wrapper types provide stronger Go-native semantic identity. String qualifiers may be reconsidered only for starter ecosystems where application-defined wrapper types create unacceptable friction.

### Parameter-name matching

Rejected. Renaming a parameter must not change dependency selection, and names are not semantic type identity.

### `@Primary` or declaration-order precedence

Rejected. Ambiguity must fail closed. Source order, registration order, and map iteration are never acceptable selection rules.

### Implicit collection of all assignable providers

Rejected until deterministic membership, ordering, module visibility, empty behavior, and overrides are designed.

### Runtime service locator

Rejected by architecture. Dependencies remain ordinary provider parameters and generated calls.

## Required future regression matrix

The first implementation or documentation slice that claims interface-binding support should prove all of the following.

### Positive

- A provider returning an interface directly satisfies an exact interface dependency.
- A concrete provider plus a one-input adapter provider satisfies both concrete and interface consumers with one concrete construction.
- The Go compiler accepts pointer-receiver and value-receiver implementation cases correctly.
- An interface imported from another package retains the canonical package-path type ID.
- A named wrapper type distinguishes two values whose embedded third-party type is identical.
- An explicit aggregate provider produces deterministic collection order.
- Reordering files and provider declarations does not change the stable graph summary.

### Negative

- A concrete provider that merely implements an interface does not satisfy the interface without an exact interface-returning provider.
- Two providers returning the same interface fail as duplicate exact outputs.
- A type alias does not create a distinct qualifier key.
- A pointer provider does not satisfy a value dependency or vice versa unless ordinary exact provider metadata says so.
- Parameter names do not influence selection.
- No provider body executes during catalog or graph analysis.
- No string qualifier or primary/fallback behavior appears accidentally.
- Module-internal implementation packages remain inaccessible where Go or Spice module rules forbid them.

### Runnable proof

A later bounded issue should include ordinary application fixtures and record at least:

```text
make verify
go test ./compiler/provider -run TestCatalogInterfaceProviders -v
go test ./compiler/graph -run TestGraphRequiresExplicitInterfaceBinding -v
go test ./compiler/graph -run TestGraphRejectsDuplicateInterfaceBindings -v
go tool github.com/spice-framework/toolchain/cmd/spice verify <valid-interface-binding-fixture>
go tool github.com/spice-framework/toolchain/cmd/spice verify <missing-explicit-binding-fixture>
GOPROXY=off go test -mod=vendor ./...
```

The exact paths should follow the fixture conventions established by issues #13 and #17.

## Security and correctness consequences

- No runtime reflection or service-location surface is introduced.
- No concrete type is selected arbitrarily for an interface.
- No string metadata controls code execution or filesystem access.
- No provider is executed during analysis.
- Ambiguity and missing bindings fail closed before generation.
- Explicit composition makes security-sensitive implementation replacement reviewable in source control.
- Module boundary checks remain authoritative; a binding is not an access-control bypass.
- Generated code can call the exact declared provider and preserve normal Go error handling.

## Compatibility and developer ergonomics

The recommended baseline favors slightly more composition code in exchange for stronger tooling and predictability.

Developers gain:

- normal Go signatures instead of framework-specific type strings;
- compile-time interface conformance;
- direct provider unit tests;
- precise navigation and rename support;
- explicit implementation selection;
- stable behavior when the provider set grows;
- inspectable generated construction code.

The main cost is a small adapter function when both concrete and interface identities are needed. That cost is bounded, obvious, and usually concentrated in application composition packages.

## Performance expectations

Interface bindings represented as provider adapters add one graph node and one edge per declared projection. Graph construction remains linear plus stable sorting. Bootstrap adds one trivial function call per adapter unless generated code later performs a semantics-preserving inline optimization.

No additional package load, source parse, reflection scan, persistent cache, or assignability index is justified.

## Recommended future slices

Do not create these issues while the ready backlog remains at three.

### Slice A — Document and test exact interface-returning providers

After issues #13 and #17 are implemented, add focused fixtures proving that ordinary providers can return interfaces and that concrete implementations are not selected implicitly.

This may be a small extension of provider/graph documentation rather than a new public API.

### Slice B — Composition and module-boundary guidance

Document where interface adapter providers belong in a modular application and ensure module verification renders and enforces the explicit binding edge.

### Slice C — Typed semantic wrappers and aggregate providers

Add examples for multiple databases, outbound clients, handlers, and health contributors using named wrapper and aggregate types.

### Slice D — Re-evaluate qualifiers and groups from real starter use cases

Only after starter and web integration work produces concrete friction, research whether Spice needs:

- typed qualifier annotations;
- named providers;
- deterministic groups;
- default/fallback starter candidates;
- explicit application/test replacements.

Any such phase must transform the active provider catalog before exact graph construction and must preserve deterministic, source-positioned diagnostics.

## Final recommendation

Spice should achieve interface-oriented dependency injection through ordinary exact-typed Go providers, not automatic structural candidate discovery.

The foundational rule is:

```text
A dependency is satisfied only by a provider that declares the exact requested Go type.
Use an ordinary @Bean adapter provider to expose a concrete value as an interface.
Use named wrapper and aggregate types for semantic multiplicity.
Reject ambiguity instead of guessing.
```

This covers the practical Spring outcomes of interface-based constructor injection and explicit collaborator selection while remaining more Go-native, deterministic, modular, and refactor-safe than runtime name/qualifier arbitration.
