# Compile-Time Provider Dependency Graph and Diagnostics

Date: 2026-07-24

## Question

After Spice can load one typed Go program, resolve annotations to exact symbols, and build a validated `@Bean` provider catalog, what is the smallest dependency-graph capability that should come next so missing dependencies and constructor cycles fail deterministically before generation or startup?

## Current delivery state

At the time of this research:

- issue #8 / PR #15 is the active delivery lane for type-aware package loading and stable symbols;
- issue #11 is the next typed annotation-resolution slice;
- issue #13 follows it with `@Bean` provider signature analysis and a deterministic provider catalog;
- the ready backlog contains issues #11 and #13, leaving room for one additional bounded issue;
- graph construction must consume issue #13's provider catalog rather than rediscovering functions, reparsing files, or loading another Go type universe.

## Why this is the next coherent capability

Provider metadata alone can say that a function supplies `*Store` and requires `Config`, but it cannot yet answer:

- whether every required type has a provider;
- whether the providers form an acyclic construction graph;
- which deterministic order generated Go should use later;
- which exact provider parameter caused a failure;
- whether the same graph result is reproduced across machines and source ordering.

A graph-validation slice is therefore the final semantic step before generated constructor wiring. It also creates reusable graph metadata for application lifecycle, module dependency verification, test graphs, architecture diagrams, and startup diagnostics.

## Primary sources and status

Sources were accessed on 2026-07-24.

### Spring Framework

- Dependency injection and constructor-cycle behavior:
  - https://docs.spring.io/spring-framework/reference/core/beans/dependencies/factory-collaborators.html
- Bean definitions and dependency metadata:
  - https://docs.spring.io/spring-framework/reference/core/beans/definition.html
- Default eager singleton initialization:
  - https://docs.spring.io/spring-framework/reference/core/beans/dependencies/factory-lazy-init.html
- Bean scopes:
  - https://docs.spring.io/spring-framework/reference/core/beans/factory-scopes.html
- Explicit initialization/destruction ordering through `depends-on`:
  - https://docs.spring.io/spring-framework/reference/core/beans/dependencies/factory-dependson.html

Spring Framework is Apache License 2.0. Spice uses these documents as capability evidence, not as a code dependency or class-by-class porting specification.

### Google Wire

- Public package documentation and injector/provider graph model:
  - https://pkg.go.dev/github.com/google/wire
- Repository and documentation:
  - https://github.com/google/wire
- Provider conflict rationale:
  - https://github.com/google/wire/blob/main/docs/faq.md

Wire is Apache License 2.0 and was archived on 2025-08-25. It remains useful design precedent for generated, reflection-free Go initialization but should not become a Spice dependency.

### Uber Dig

- Directed acyclic container graph, cycle errors, graph visualization, and missing dependency behavior:
  - https://pkg.go.dev/go.uber.org/dig
- Internal generic cycle-detection contract, inspected only as algorithm evidence:
  - https://pkg.go.dev/go.uber.org/dig/internal/graph

Dig is MIT licensed and remains a runtime reflection container. Spice should learn from its diagnostics and graph observability while retaining compile-time analysis and generated ordinary Go.

### Go language

- Type identity and assignability:
  - https://go.dev/ref/spec
- Semantic type comparison:
  - https://pkg.go.dev/go/types#Identical

The graph must consume exact semantic types from the single `compiler/load.Program`. Stable type strings are for diagnostics and serialization, not a replacement for `go/types` identity during one compiler run.

## Findings and decisions

### 1. Validate one active application graph over the complete bootstrap catalog

Spring `ApplicationContext` implementations eagerly create and configure singleton beans by default so configuration failures surface during context initialization. Wire instead starts from explicit injector roots and includes only providers needed to produce those outputs.

Spice does not yet have:

- conditional providers;
- lazy providers;
- scopes;
- starter activation;
- test overrides;
- multiple application roots;
- explicit provider-set selection.

The first graph should therefore treat every provider in the validated catalog as active application-singleton metadata and validate the whole catalog. This gives a simple rule:

```text
Every declared @Bean provider must have all exact dependencies available,
and all declared providers must form one acyclic graph.
```

This is intentionally a bootstrap rule, not a permanent restriction. A later activation phase may filter providers by application, module, condition, profile, starter, or test graph before graph construction. That future filter must happen explicitly before the graph builder; the graph builder should never invent activation policy.

Why not add roots now:

- the current `@Application` annotation has no typed root contract;
- choosing an application entrypoint is a code-generation and lifecycle decision;
- allowing unused broken providers would make verification dependent on an undefined reachability model;
- combining roots, graph validation, code generation, and lifecycle would make the next issue too broad.

### 2. Build graph edges from provider parameter types only

Issue #13 should produce provider records like:

```text
provider NewService
  output: *example.com/app/service.Service
  inputs:
    0: example.com/app/config.Config
    1: *example.com/app/store.Store
```

The graph builder maps each input type to the unique provider with an exact identical output type.

For each provider `P` and dependency `D`:

```text
P requires D
D must be constructed before P
```

The internal graph may store edges as `consumer -> dependency` because that matches diagnostics, but its exported construction order must place every dependency before every consumer.

Do not rescan signatures. Do not infer dependencies from function bodies, global variables, fields, naming conventions, or package imports.

### 3. Keep exact-type matching and fail closed

The provider catalog already rejects duplicate exact output types. The graph should then perform one exact lookup per dependency using the catalog's semantic type index.

It must not:

- use assignability to silently bind concrete values to interfaces;
- flatten named types to their underlying types;
- choose a provider by declaration order;
- search for a “close enough” pointer or value type;
- auto-construct structs with no provider;
- treat zero values as implicit providers.

Future interface bindings, qualifiers, names, primary/fallback selection, groups, and optional dependencies should transform dependency keys before graph construction. They should not complicate the first graph.

### 4. Missing dependencies need one diagnostic per unsatisfied parameter

A missing provider should be attached to the consuming provider's parameter position when available and include:

- diagnostic code;
- consuming provider stable symbol ID;
- required canonical type ID;
- parameter index and name when available;
- source position;
- correction guidance.

Example:

```text
service/provider.go:8:21: provider example.com/app/service.NewService
requires *example.com/app/store.Store for parameter 1 "store", but no @Bean
provider supplies that exact type
```

Independent missing dependencies should accumulate and sort deterministically. The graph builder should still construct all resolvable edges so it can report independent cycles in the same verification run, but it must not return a usable construction order while any graph diagnostic exists.

### 5. Detect strongly connected components rather than stopping at the first cycle

A first-hit depth-first-search error is easy to implement but produces order-dependent results and hides independent cycles. Spice should identify strongly connected components over the resolved provider graph.

A component is cyclic when:

- it contains more than one provider; or
- it contains one provider with a self-edge.

For each cyclic component, render one deterministic representative cycle:

1. sort component nodes by provider stable symbol ID;
2. choose the lexically smallest node as the start;
3. traverse outgoing dependency edges in stable provider-ID order;
4. render a closed path back to the start;
5. list every provider and source position in the component as supporting context.

Example:

```text
constructor dependency cycle:
  example.com/app/a.NewA -> example.com/app/b.NewB
  example.com/app/b.NewB -> example.com/app/c.NewC
  example.com/app/c.NewC -> example.com/app/a.NewA
```

The implementation may use Tarjan or Kosaraju internally. Do not add a graph dependency for this small, deterministic compiler algorithm unless a later need justifies one.

### 6. Produce a stable construction order only for a valid graph

For an acyclic graph, produce a topological construction order with dependencies before consumers.

Multiple valid topological orders often exist. Spice must choose one reproducibly:

- calculate readiness from dependency counts;
- when multiple providers are ready, select by stable provider symbol ID;
- never use map iteration, source scan order, filesystem order, or package load order as a tie-break;
- return immutable ordered provider references from the original catalog.

This stable order becomes the contract for later generated wiring and reverse-order shutdown planning. The graph issue must not invoke providers or generate source yet.

### 7. Keep graph records inspectable and serialization-friendly

A conceptual compiler-internal API is:

```go
package graph

type Result struct {
    Nodes         []Node
    Edges         []Edge
    Construction  []provider.ID
    Diagnostics   []diagnostic.Diagnostic
}

type Node struct {
    Provider provider.ID
    OutputID provider.TypeID
    Position token.Position
}

type Edge struct {
    Consumer       provider.ID
    Dependency     provider.ID
    DependencyType provider.TypeID
    ParameterIndex int
    Position       token.Position
}

func Build(catalog provider.Catalog) Result
```

Exact names may differ, but the boundary should guarantee:

- no package loading or AST parsing;
- no provider execution;
- no global mutable state;
- deterministic node, edge, diagnostic, and construction ordering;
- source-positioned failures;
- references back to exact provider records;
- a serializable stable-ID representation for tests and future diagrams.

### 8. Continue after independent errors but never generate from an invalid graph

Recommended phase behavior:

1. If the typed program is unsafe, do not build annotations, providers, or the graph.
2. If annotation resolution/validation fails, do not build the provider catalog.
3. If the provider catalog has invalid signatures or duplicate outputs, do not build the graph.
4. If the catalog is valid, build every resolvable edge.
5. Accumulate all missing dependency diagnostics.
6. Detect all independent cyclic components among resolved edges.
7. Emit a construction order only when no graph diagnostics exist.

This avoids cascades from invalid provider metadata while still giving developers useful graph feedback in one run.

### 9. Do not introduce implicit external inputs

Wire injector function parameters can act as graph inputs. Spice has not defined an injector-function or application-root signature, so the bootstrap graph should have no implicit external values.

A dependency without an exact provider is an error, even for common types such as:

- `context.Context`;
- `*slog.Logger`;
- configuration structs;
- `*sql.DB`;
- HTTP routers or servers.

Those values should come from explicit providers. Later, Spice may define generated runtime-owned values through explicit built-in providers or application-root inputs, but silently exempting special types would create undocumented service-location behavior.

### 10. Separate graph validation from lifecycle and generated wiring

This issue should stop after validated metadata and deterministic order.

Do not yet decide:

- provider invocation syntax;
- generated variable naming;
- error-return propagation;
- partial-construction rollback;
- cleanup callbacks;
- startup and shutdown interfaces;
- context cancellation;
- root output exposure;
- generated file placement;
- runtime application container shape.

Those concerns need a lifecycle and generation contract informed by the valid graph.

### 11. Graph complexity and performance budget

For `V` providers and `E` dependency parameters:

- exact provider lookup should be effectively constant time per edge;
- node/edge construction should be `O(V + E)` before deterministic sorting;
- strongly connected components should be `O(V + E)`;
- deterministic ordering may add `O(V log V + E log E)`;
- no additional `packages.Load`, AST traversal, or function-body analysis is allowed;
- no persistent cache is needed in this slice.

Tests should include a synthetic graph large enough to catch accidental quadratic scans without freezing a brittle wall-clock benchmark.

### 12. This graph becomes shared architecture infrastructure

The provider graph is not only a DI implementation detail. Stable nodes and edges can later support:

- generated initialization;
- reverse-order shutdown;
- Spring Modulith module dependency derivation;
- forbidden cross-module injection checks;
- architecture diagrams;
- test-application subgraphs;
- controller/service reachability;
- startup explanations such as “why is this provider included?”;
- module-aware traces and health metadata.

Keep the core graph package independent of HTTP, configuration, security, data, and runtime packages.

## Spring and Go comparison

| Concern | Spring behavior | Wire/Dig evidence | Spice bootstrap direction |
|---|---|---|---|
| Graph population | Active bean definitions in the container | Wire injector provider set; Dig registered constructors | Every validated active `@Bean` provider in the catalog |
| Missing dependency | May surface during context/bean creation | Wire generation or Dig invocation fails | `spice verify` fails statically at the provider parameter |
| Constructor cycle | Detected at runtime and rejected | Wire DAG generation; Dig cycle error | Compile-time SCC diagnostics with stable paths |
| Initialization | Eager singleton by default, lazy optional | Wire generates only needed roots; Dig instantiates retained values | Validation covers all bootstrap providers; invocation deferred |
| Candidate selection | Type, qualifier, primary/fallback, name | Wire explicit binding; Dig names/groups | Exact type only; advanced selection deferred |
| Ordering | Dependency order plus explicit `depends-on` | Graph-derived | Stable dependency-first topological order |
| Runtime mechanism | Container metadata and runtime creation | Generated code or reflection container | Typed compiler records now; generated ordinary Go later |

## Recommended implementation sequence

After issues #8, #11, and #13 merge:

1. add `compiler/graph` consuming only a valid provider catalog;
2. create stable nodes and exact-type dependency edges;
3. emit one diagnostic for every missing provider parameter;
4. detect all cyclic strongly connected components, including self-cycles;
5. return a deterministic dependency-first topological order only for a valid graph;
6. integrate graph diagnostics into `spice verify` after provider validation;
7. add valid diamond, missing dependency, self-cycle, multi-node cycle, and multiple-independent-cycle fixtures;
8. prove no providers execute and no package reload occurs;
9. preserve offline vendored execution and Go 1.23 compatibility;
10. update compiler and Spring-coverage documentation without claiming generated DI is available yet.

## Explicitly deferred capabilities

- application/injector roots and reachable-provider pruning;
- generated provider invocation;
- application lifecycle, cleanup, rollback, and shutdown;
- interface bindings and assignability;
- qualifiers, names, primary/fallback, order, groups, and collections;
- optional dependencies and lazy/provider handles;
- conditional providers, profiles, starters, and test overrides;
- scopes beyond future application-singleton semantics;
- module visibility and allowed-dependency enforcement;
- explicit `depends-on` relationships not represented by typed parameters;
- provider methods or configuration-class semantics.

## Recommendation

Create one bounded graph-validation issue after the provider catalog issue. The implementation should validate the complete active bootstrap catalog, reject missing exact dependencies and all constructor cycles, and produce deterministic metadata and construction order without executing or generating anything.

That slice advances Spring-style dependency injection materially while preserving ordinary Go signatures, compile-time feedback, deterministic builds, and a clean boundary for the later lifecycle/code-generation design.