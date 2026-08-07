# Spice Architecture

## Product thesis

Spice is a Go-native application platform. It provides Spring-style application
outcomes through valid Go source, compile-time analysis, deterministic generated
Go, explicit dependencies, and small typed runtime contracts. It does not use a
runtime service locator, reflection-based dependency lookup, package scanning,
or hidden network access.

The product is intentionally split into independently versioned repositories.
This repository is the public core library, not the compiler distribution.

## Repository boundaries

| Repository | Ownership |
| --- | --- |
| `spice-framework/spice` | Public annotations, SDK/protocol, runtime contracts, test support, and starter-composition metadata |
| `spice-framework/toolchain` | Compiler, generator, CLI, LSP, development supervisor, official annotation tool, bootstrap, release construction, and toolchain dogfooding |
| `spice-framework/goland` | Native GoLand presentation, navigation, completion, Run/Debug, and installed-IDE acceptance |
| `spice-framework/zed` | Secondary Zed/LSP adapter |
| `spice-framework/starter-*` | Independently versioned external-system adapters and real-service verification |
| `spice-framework/petclinic` and `commerce` | Reference applications and black-box developer-workflow acceptance |
| `spice-framework/development` | Cross-repository compatibility orchestration and contributor tooling |

Core never imports toolchain implementation packages. Applications need the
compiler only while analyzing or generating; generated applications build and
run with ordinary Go plus the public core and selected starter modules.

## Core package model

The core module is standard-library-only and has four public layers:

1. `annotation/**` defines canonical declaration descriptors, the public SDK,
   the versioned process protocol, and test helpers for extension authors.
2. Foundation packages such as `bean`, `lifecycle`, `config`,
   `conversion`, `validation`, `security`, and `observability` define
   immutable typed contracts.
3. Capability packages such as `web`, `data`, `event`, `mail`,
   `messaging`, `schedule`, `cache`, and `management` provide
   standard-library-first runtime behavior.
4. `spicetest` and SDK test packages provide black-box application,
   annotation, HTTP, SQL, and generated-context testing seams.

Optional external clients do not enter this module. The small `starter`
package contains portable composition metadata only; network/database
implementations live in dedicated starter modules.

`internal/qualitygate` is the sole internal package retained here. It verifies
the library repository and is not product runtime or toolchain implementation.

## Annotation and tool authorization

Application files remain valid Go:

```go
// @import { Application, Service } from "github.com/spice-framework/spice/annotation/core"

// @Application
func main() {}
```

Descriptor definitions are statically decoded; their bodies are never executed
during analysis. Annotation tools are explicitly authorized by the consuming
application's `go.mod`:

```go
tool (
    github.com/spice-framework/toolchain/cmd/spice
    github.com/spice-framework/toolchain/cmd/spice-annotation-core
)
```

`annotation/coretool.Path` is the canonical bridge to the official annotation
tool. Go's module graph, checksum database, cache, vendor behavior, and
`replace` directives remain the only dependency system.

## Compiler pipeline

The separately versioned toolchain owns one typed-program pipeline:

```text
valid Go + overlays
    -> packages and types
    -> annotation imports and descriptor resolution
    -> generic annotation-tool contributions
    -> providers, interface bindings, configuration, modules, routes, lifecycle
    -> immutable application IR
    -> deterministic generation plan
    -> guarded filesystem application
```

Every diagnostic retains an exact source range. Compiler packages do not switch
on annotation names; official and third-party handlers return typed SDK
contributions that the compiler validates generically.

The LSP, CLI, development supervisor, and GoLand plugin consume the same
analysis service. Editor presentation may fold the physical `// ` prefix, but
must never alter the document bytes that Go tools compile.

## Dependency injection

Providers are exact typed factories. Concrete values become interface
candidates only through an explicit typed `@Implements` contribution or by a
factory returning that exact interface. Qualifiers, primary/fallback selection,
ordered collections, optional/lazy/provider handles, and scopes are resolved at
compile time.

Generated code calls constructors directly, registers cleanup immediately, and
rolls failures back in reverse order. Interface assertions live in generated
source mirrors so application source remains clean while ordinary Go still
checks method sets. There is no runtime BeanFactory mutation or string lookup.

## Generated application layout

Generation belongs to the consuming application, never this core repository.
Each target owns an importable package:

```text
internal/spicegen/<target>/
    spice_contracts_gen.go
    spice_configuration_gen.go
    spice_providers_gen.go
    spice_assembly_gen.go
    spice_features_gen.go
    spice_http_gen.go
    spice_lifecycle_gen.go
    spice_command_gen.go
    sources/<source-directory>/<source>_spice_gen.go
    artifacts/openapi.json
.spice/<target>.manifest.json
```

Stable subsystem files keep assembly readable. Source mirrors provide a
deterministic one-source-file-to-one-generated-file debugging boundary for
assertions and annotation-derived glue. Route files may split further when
needed to preserve useful stack frames and review size.

All generated files have the canonical marker, stable imports and names, no
timestamps or absolute paths, and SHA-256 ownership. The manifest is only a
generated-file ownership record. Guarded application preserves unchanged files
and refuses collisions, stale ownership, symlink escapes, and manual edits.

## Runtime and lifecycle

Generated applications expose typed construction, startup, shutdown, run, and
component-snapshot seams. Construction cleanup is registered immediately.
Startup is dependency-first; shutdown and rollback are reverse dependency
order. Stop and returned leases are idempotent, and callers own contexts,
signals, deadlines, and transport policy.

Public runtime packages use explicit interfaces and immutable values. Global
mutable containers, hidden clients, implicit goroutines, and background module
downloads are forbidden.

## Modules and integrations

Module identity is a Go import path. Root APIs, descendant internals, named
interfaces, allowed dependencies, and cycles are checked by the toolchain.
Generated metadata feeds module canvases, focused tests, observations,
migrations, events, and starter composition.

External integrations remain isolated. Each starter must document its
dependency/security review and prove cancellation, timeout, cleanup,
observability, offline builds, minimum/current core compatibility, and a real
service before it is presented as supported.

## Verification

The core verifier enforces:

- exact Go 1.26.5;
- goimports and gofumpt;
- root and tools `go mod tidy -diff`;
- a standard-library-only root graph and reproducibly empty vendor result;
- vet, allowlisted golangci-lint, NilAway, gosec, and govulncheck;
- one combined shuffled race/coverage pass across exactly 51 public packages;
- at least 85% aggregate public-source coverage;
- bounded parser/decoder fuzz smoke and offline public-package tests;
- API maturity, Spring coverage, documentation, namespace, package-direction,
  and repository-boundary checks.

Toolchain, editor, starter, and reference-application repositories own their
specialized acceptance. Cross-repository release readiness is proven against
exact immutable revisions by the development repository.

## Capability parity policy

Spring Boot and Spring Modulith are the capability map, not an API-shape
mandate. Features are implemented where Go can provide a maintainable typed
contract. Runtime BeanFactory mutation, classloader weaving, universal
proxy-based AOP, and unrestricted expression execution remain deliberately
excluded in favor of compile-time wiring, generated decorators, explicit
adapters, and bounded typed expressions. The disposition of every capability
is tracked in [`docs/spring-coverage.md`](docs/spring-coverage.md).
