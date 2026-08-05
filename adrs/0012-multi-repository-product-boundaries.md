# ADR 0012: Multi-repository product boundaries

Status: Accepted

## Context

Spice began in one repository so its source model, compiler, runtime, editor
integrations, starters, and examples could change atomically. That bootstrap
phase succeeded, but the resulting repository is no longer one coherent
release unit.

The measured repository now contains more than one hundred Go packages, two
editor implementations, two substantial reference applications, a native
compiler toolchain, and integrations for independently versioned external
systems. At the time of this decision, the root Go module selected dependencies
for PostgreSQL, MySQL, Redis,
Kafka, OpenTelemetry, OAuth/OIDC, gRPC, and WebSocket even when an application
uses only the standard-library-first runtime. Go package pruning avoids
compiling unused packages, but it does not remove module graph, checksum,
license, vulnerability, upgrade, or release coupling.

The products also have materially different acceptance requirements:

- the core runtime and annotation SDK require Go API compatibility;
- the compiler and CLI require Go toolchain and generated-code compatibility;
- GoLand and Zed require editor-specific compatibility and packaging;
- each external starter requires its own service compatibility matrix;
- reference applications consume published artifacts and must not be part of
  the framework's production dependency graph.

These are the independent compatibility, ownership, scale, and release
lifecycles anticipated by ADR 0001 and ADR 0011.

## Decision

The canonical publisher is the
[`spice-framework`](https://github.com/spice-framework) GitHub organization.
Spice adopts a bounded multi-repository ecosystem with the following dependency
tiers.

### Foundation repositories

| Repository | Owns | Release contract |
| --- | --- | --- |
| `spice-framework/.github` | Organization profile, community health, security policy, and reusable CI workflows | No product artifact |
| `spice-framework/development` | Clone/bootstrap tooling, a native `go.work` development workspace, ecosystem verification, and coordinated release metadata | No runtime dependency |
| `spice-framework/spice` | Standard-library-first runtime capabilities, annotation descriptors, public annotation SDK/protocol, `spicetest`, architecture documentation, and compatibility policy | Go module `github.com/spice-framework/spice` |
| `spice-framework/toolchain` | Compiler, generator, CLI, LSP, development supervisor, official annotation tool, release construction, and toolchain dogfooding | Go module `github.com/spice-framework/toolchain`; installs `cmd/spice` and `cmd/spice-annotation-core` |

The `spice` module is the only foundation dependency of starters. Runtime
packages never import `toolchain`. The toolchain depends on published public
contracts from `spice`; it does not receive privileged access to core
implementation internals.

### Editor repositories

| Repository | Owns |
| --- | --- |
| `spice-framework/goland` | IntelliJ Platform plugin, installed-IDE fixtures, visual goldens, Plugin Verifier, Run/Debug integration, and editor release artifacts |
| `spice-framework/zed` | Zed extension, editor fixture, and Zed-specific acceptance within the public editor API ceiling |

Editors execute a compatible released `spice` CLI/LSP. They do not duplicate
the compiler model and are versioned independently from Go modules.

### Starter repositories

External systems own independent modules and release evidence:

| Repository | Initial scope |
| --- | --- |
| `spice-framework/starter-postgres` | pgx-backed SQL pool and PostgreSQL integration acceptance |
| `spice-framework/starter-mysql` | MySQL driver pool and MySQL integration acceptance |
| `spice-framework/starter-redis` | Redis cache integration and real Redis acceptance |
| `spice-framework/starter-kafka` | Kafka producer/consumer integration and real broker acceptance |
| `spice-framework/starter-observability` | OpenTelemetry integration and exporter compatibility |
| `spice-framework/starter-security` | OAuth2 client and OIDC resource-server integration |
| `spice-framework/starter-grpc` | gRPC server/client integration and interoperability acceptance |
| `spice-framework/starter-websocket` | WebSocket server/client integration and interoperability acceptance |
| `spice-framework/starter-smtp` | Secure SMTP transport, test server acceptance, retry, and cancellation |

One repository may own multiple packages when they share the same external
dependency and compatibility lifecycle. It must not aggregate unrelated
drivers merely to reduce the repository count. Adding a starter repository
requires a named maintainer, dependency review, real-system verification, and
documented support matrix.

[ADR 0013](0013-independent-starter-release-lifecycles.md) refines this initial
inventory after implementation proved that OpenTelemetry, OAuth2 client, and
OIDC resource-server integrations have independent dependency and acceptance
lifecycles.

### Reference application repositories

| Repository | Owns |
| --- | --- |
| `spice-framework/petclinic` | The Spring Petclinic parity application and Windows/Linux developer-proof workflow |
| `spice-framework/commerce` | Modular commerce, security, transaction, event, mail, and failure-path integration proof |

Reference applications consume released module versions. Their normal builds
must not use unpublished relative replacements. A separate explicit
development workspace may use `go.work` to test coordinated source changes.

## Dependency and version rules

1. Repositories publish independent semantic versions; the ecosystem does not
   use artificial lockstep versioning.
2. `go.mod`, `go.sum`, Go module proxies, and normal Go tags remain the only
   library dependency and integrity mechanism.
3. There is no Java-style BOM or custom Spice dependency resolver. The
   development repository may publish a tested compatibility catalog and the
   CLI may suggest those versions, but applications retain ordinary Go module
   ownership.
4. Starter modules declare the minimum compatible `spice` version and test
   both that floor and the current supported version.
5. The toolchain negotiates the public annotation protocol and records its
   supported core range. It may not import unexported core implementation.
6. Editor releases declare compatible CLI/LSP protocol ranges and fail with an
   actionable health diagnostic when the installed executable is outside that
   range.
7. Pre-1.0 breaking changes require migration notes. The first public preview
   freezes the packages explicitly classified as preview-stable.
8. A repository release is built from a clean exact tag and includes
   checksums, an SBOM where applicable, provenance, and signatures.

## Development and verification

Each product repository owns a fast local `make verify` appropriate to its
artifact. Package-oriented affected checks remain valid within repositories.
Long-running cross-repository, installed-IDE, and real-service matrices live in
the development repository and release workflows; they do not make a runtime
package edit wait on every unrelated product.

The development repository uses Go's native workspace support and ordinary
Git checkouts. It is orchestration, not a dependency manager. A released
consumer is always verified outside that workspace against published module
versions.

## Migration and history preservation

The existing `StevenBuglione/spice` repository is transferred to
`spice-framework/spice` so its full history, issues, and redirects survive.
New repositories are extracted from that history with path-filtered commits
before their source is removed from the core repository. No product is copied
as a historyless initial snapshot.

Extraction follows the dependency order recorded in
[`docs/repository-migration.md`](../docs/repository-migration.md). During the
transition, the monorepo remains the authoritative green source. A repository
becomes authoritative only after its independent module, tests, documentation,
release metadata, and clean-room consumer acceptance pass.

## Cleanup policy

Repository splitting is not permission to discard unexplained work.

- Generated caches, binaries, temporary clones, and verification logs are
  disposable when they are untracked or ignored and reproducible.
- Generated source is removed only through its ownership manifest or after its
  consuming application has been extracted and regenerated.
- A public package is deleted only after import-graph, documentation,
  generated-output, fixture, and Git-history review proves it is obsolete.
- Research documents are retained until their conclusions are incorporated
  into an accepted ADR, public contract, or test; then they may be archived
  with provenance.
- Deprecated public paths receive migration diagnostics or forwarding policy
  where feasible. A hard cut must be documented before the first preview.
- Vendor contents are regenerated, never hand-edited.

## Consequences

The core module becomes smaller, standard-library-first, and easier to adopt.
External integrations no longer expand the dependency and vulnerability
surface of unrelated applications. Editors and starters can release against
their actual platform lifecycles. Reference applications become genuine
external consumers.

The cost is coordinated compatibility work across repositories. That cost is
accepted and controlled through native Go workspaces, explicit version floors,
reusable verification workflows, clean-room tests, and a small number of
cohesive repositories rather than one repository per package.
