# Spice Roadmap

## Current developer-proof priority

The developer-proof milestone now composes one complete workflow: a minimal
annotated `main.go`, compile-time local-module discovery, stable
generate/verify/build/run behavior, a last-known-good `spice dev` supervisor,
one shared overlay-aware compiler service, editor-neutral LSP support, a
polished primary GoLand integration, a supported secondary Zed integration,
generated exact-scope security, transaction-owned persistent data, and one
secure mail integration exercised by the modular commerce application. The
remaining work is release hardening and deliberately selected starter breadth,
not replacement of this foundation.

The Spring coverage map is now contract-closed: every capability is classified
as executable `available`, bounded `integration`, or deliberately
`not-planned`. Repository verification rejects unresolved states. This does not
claim Java class-for-class parity; it freezes the practical Go-native boundary
and keeps ecosystem breadth visible as integration work.

Release hardening now includes a repository-owned offline artifact builder:
clean exact tags produce deterministic `-trimpath` CLI archives for
Windows/Linux/macOS on amd64/arm64, a vendor-derived SPDX 2.3 SBOM, SHA-256
checksums, and an Ed25519 signature. Tag automation repeats the full release
gate before publishing; Go's module/vendor graph remains the only dependency
and build input.
Release acceptance also enforces repository-owned median time, memory, and
allocation budgets for immutable model construction, deterministic rendering,
warm editor analysis, and CLI parsing. The getting-started tutorial,
Spring-to-Spice migration map, and pre-1.0 upgrade procedure make the generated
and operational contracts independently usable without repository history.

The preferred package-main discovery and explicit generated-package entrypoint
are now exercised by Commerce and Petclinic; no generated declaration is
written beside handwritten source. `spice run` applies guarded generation and builds a unique
trimpath candidate, and relays the selected package-main process on Windows and
Unix. Coverage-map labels are not substitutes for executable acceptance. The
shared `spice.diagnostics/v1` contract now backs text and machine-readable
verification while retaining physical/source-mapped positions for editor use.
The stable-core executable matrix is recorded in
`docs/stable-core-acceptance.md`; the commerce reference now closes the
security/data/mail vertical integration. The
reusable dev supervisor and portable recursive watcher now provide
deterministic debounce, unique candidate builds, last-known-good recovery, and
bounded Windows/Unix process-group restart through `spice dev`. A native watch
accelerator remains follow-up; content polling is the correctness fallback.
The isolated overlay compiler service now returns the shared diagnostics,
resolved annotations, exact provider/application/module/configuration models,
safe fixes, and pure generation readiness with cancellation, stale-sequence
rejection, and bounded content-keyed caching. Generate, build, run, and dev
now consume that service, including imported auto-configuration decisions and
Go module/replacement provenance. `spice lsp` now publishes the same versioned
diagnostics and provides compiler-derived annotation, module, configuration,
hover, safe-edit, and semantic-token metadata over bounded stdio JSON-RPC.
The repository-owned GoLand plugin is now the primary editor target. Its native
folding pipeline reclaims the exact physical width of canonical `// `
prefixes, its native annotator supplies configurable structured color, and
multi-range PSI references resolve import paths, symbols, aliases, namespaces,
and invocations to real indexed Go descriptor functions. Native Quick
Documentation combines descriptor GoDoc with module/replacement provenance,
tool authorization, handler/protocol data, and implementation source; native
definition search maps descriptors to their real handlers. An offline-safe
completion fallback and actionable Spice health window remain available while
the LSP restarts. A packaged-plugin Starter/Driver suite launches pinned
GoLand with the freshly built repository LSP, proves declaration-safe typing,
save/undo/redo/reformat/reopen preservation, zero-width coordinates, real
modifier-hover/click navigation, visible documentation, and health state, and
captures reviewed light/Darcula editor renders plus documentation and health
artifacts. The shared LSP supplies alias-aware
explicit-import completion, rich descriptor documentation and signature help,
real descriptor definitions, and real handler implementation links without an
editor registry. The Zed extension remains supported beside `gopls` on Windows
and Linux for native completion, diagnostics, modifier-click navigation, quick
fixes, and semantic-token presentation, within Zed's public no-concealment
boundary. The bounded test mail transport now supplies
immutable decoded delivery snapshots, deterministic failure injection,
explicit capacity behavior, and payload-free observations. The secure SMTP
slice adds verified STARTTLS/implicit TLS, bounded cancellation and retry,
ambiguous-delivery protection, and payload-free observations. Commerce now
uses generated exact-scope authorization, serializable transaction ownership,
an explicit repository interface binding, module-owned lifecycle migration, a
transaction-aware zero-dependency development database, a PostgreSQL
restart-persistence integration path, reviewed secure PostgreSQL and MySQL
pool starters, an explicit `mail.Sender` binding, and a
post-commit inspectable receipt workflow. That workflow is now repeatable on
Windows and Linux. Reviewed Kafka producer/consumer-group, gRPC server/client,
and WebSocket server/client integrations have resumed M4. The OpenTelemetry
starter now also projects compiler-owned publisher/subscriber module event
interactions into payload-free spans and bounded delivery metrics. MongoDB,
OpenSearch, RabbitMQ, and GraphQL remain bounded follow-up slices.

Petclinic now serves through a generated lifecycle-owned HTTP listener with
bounded timeouts and graceful drain. Its shared responsive layout, embedded
assets, immutable English/German/Spanish message catalogs, typed
`Accept-Language` binding, localized HTML error pages, and canonical owner
browser workflow have passed real installed-browser visual and interaction
verification. The packaged GoLand 2026.2 suite now opens that real Petclinic
module, uses an exact freshly built Spice executable, proves physical-comment
preservation, zero-width concealment, light/dark token colors, navigation and
documentation, and executes multi-package Run/Debug without a single-file
`gocommand-*` path or false `spiceMain` error. A cross-platform CLI integration
test now runs the same isolated Petclinic module through the invalid-edit,
last-known-good, fix, guarded-regeneration, and graceful-restart sequence.
Petclinic deliberately retains upstream's public user routes while its
compile-time management policy restricts every generated actuator endpoint to
direct IPv4/IPv6 loopback peers and ignores forgeable forwarding headers.
Release hardening remains the next reference-application gate.

The Go-native extension migration has begun with the hard-cut `@import`
directive and explicit named, aliased, and namespace annotation bindings, the
public statically decoded descriptor SDK,
fail-closed file symbol tables, and offline auxiliary descriptor loading in the
single typed program. Exact target-module `tool` authorization, the public
bounded JSON-RPC SDK/server, offline module and replacement provenance,
persistent process hosting, cross-platform process-tree cancellation, and
read-only list/doctor commands are now present. The complete typed contribution
union is validated on both sides of the protocol, every official annotation
has a one-file descriptor and external core-tool handler, commerce uses
explicit imports, and verify/generate/build/run/dev/LSP share the tool-aware
compiler service. Annotation semantics now fail closed: every invocation must
resolve through an explicit file-scoped import to a statically decoded
descriptor and a validated typed tool contribution. No product path consults a
built-in registry or infers behavior from an annotation name.
Descriptor-backed LSP/GoLand completion, documentation, definition, and
implementation navigation are now available. The retired `@spice.import`
spelling produces an exact shared diagnostic and versioned replacement and is
neither resolved nor concealed. A separate fixture module now proves
public-SDK-only descriptors and
handlers, named aliases, namespace discovery, tool-owned diagnostics, provider
generation, real source navigation, and offline build/run. Pre-import
completion now discovers descriptors across the target module graph,
workspace modules, replacements, vendor source, and populated module cache
without network access or implicit bindings. Undeclared tools now use a
two-step LSP quick fix: `go get -tool` runs against a temporary modfile, the
exact module-file diff is shown without mutation, and a separate confirmed
action applies only the content-derived plan while both original hashes
match. Tool negotiation also declares and validates every public descriptor
package. Installation commands remain cancellable while the LSP reads new
requests. GoLand Run invokes `spice run`; Debug first runs a registered Spice
generation task and then retains the native complete-package Go/Delve path.
The pinned GoLand suite executes Petclinic through both command paths on the
Windows/Linux verification matrix. The executable commerce slice now
  proves authentication decisions, generated authorization, persistence, and
  test receipt delivery on the same generated application graph.

The constructor graph now treats `@Service`, `@Controller`, and `@Repository`
as constructible beans and records deterministic constructor discovery in the
immutable IR. `@Implements` resolves named and instantiated generic Go
interfaces from any package in the loaded module graph, validates exact
pointer/value method sets, generates source-owned Go assertions, and exposes
only those explicit bindings to dependency selection.
Commerce proves concrete payment construction with interface injection and
inspectable direct-call generated Go. The graph now also provides explicit
bean names and aliases, repeatable qualifiers, primary/fallback selection,
ordered slice/map injection, typed optional/lazy/provider handles, and
singleton/prototype/request/session cleanup ownership without reflection or a
runtime locator. Commerce proves qualifier/primary selection against a
separate fallback implementation. The shared compiler now owns the complete
named runtime-interface catalog used for `@Implements` authoring; LSP
completion inserts namespace imports without consulting an IDE index, and the
renderer owns mirrored source units. Each contributing source file maps to one
generated file below `internal/spicegen/<target>/sources`; direct provider
construction, configuration binding, and interface assertions execute there,
while target-wide coordination is explicitly decomposed into contracts,
configuration, providers, bounded assembly, features, HTTP, stable per-route
units, lifecycle, and command files. The schema-5 ownership manifest records
concern roles, primary source, related sources, and exact generated ranges.
Guarded migration removes only hash-matching schema-4 monoliths and schema-3
adjacent shards.
The packaged GoLand workflow proves native generic-aware Implement Methods
generation, pointer receiver selection, namespace-aware navigation, and valid
physical source. The independent third-party SDK/tool fixture and complete
commerce workflow are executable. Package-main generation separates
target-wide wiring from one nested source unit per contributing handwritten
file, and `spice generated` provides deterministic two-way source/range lookup.
Generated applications expose typed singleton `Components` plus exact-type
`BeanOverrides`; `bean.Replace` and `bean.ReplaceFactory` preserve direct
dependency wiring, rollback, module cleanup, and shutdown without a mutable
test container. `spicetest.Context` and `annotation/sdk/sdktest` provide
lifecycle and third-party handler test harnesses without reflection or compiler
imports.

Library defaults now use an explicit blank import of a conventional
`autoconfigure` package whose static descriptor is resolved from the
application's ordinary Go module graph. Application beans replace fallbacks
before construction, unavailable required dependencies back off, and `spice
beans --explain` exposes selection and Go module/replacement provenance. The
handwritten renderer and CLI are partitioned by the same semantic boundaries
as their generated artifacts and commands.

The dogfooding gate now builds the compiler offline without generated
application dependencies, generates and executes an isolated app from zero
output, proves byte-identical regeneration, rejects corruption, and proves
recovery. The next decision is the first bounded Spice-owned application slice
to migrate; compiler bootstrap packages remain ordinary Go.

## M0 — Product and compiler proof

- Annotation syntax and parser.
- Source scanning and declaration association.
- Typed annotation definition model.
- Source-positioned diagnostics.
- Deterministic generation contract.
- Competitive and Spring capability inventory.
- Developer interviews and ergonomics benchmarks.

## M1 — Application and module foundation

- Constructor provider graph.
- Lifecycle and shutdown.
- Typed bootstrap-feature metadata and generated command composition.
- Application module discovery.
- API/internal boundary rules.
- Allowed dependency declarations.
- Module cycle detection.
- Module documentation generation.
- Module-focused test support.

## M2 — Web application MVP

- Controller and route annotations.
- Parameter and body binding.
- Validation.
- Standard error model.
- OpenAPI generation.
- HTTP middleware and filters.
- Health, liveness, readiness, and graceful shutdown.
- Reference modular-monolith application.

## M3 — Enterprise foundations

- Externalized configuration and profiles.
- Structured logging and OpenTelemetry.
- Security policies, OAuth2/OIDC resource server support.
- SQL, reviewed PostgreSQL/MySQL pools, generated typed HTTP transaction
  boundaries, migrations, and repository support.
- Cache abstraction.
- Annotation-driven fixed-delay scheduling and asynchronous execution.
- Restartable ordered batch execution with process-local and lease-aware
  PostgreSQL persistence.
- Focused module execution plus generated HTTP and transaction-rollback data
  test slices.
- Typed application events and durable publication.
- Immutable transport-neutral external messages and explicit single-settlement
  delivery contracts; client-specific broker starters and generated listeners
  remain M4 work.

## M4 — Broad Spring Boot coverage

- Kafka, RabbitMQ, and additional messaging starters.
- Redis, MongoDB, Elasticsearch/OpenSearch, and selected data starters.
- gRPC, GraphQL, WebSocket, and outbound HTTP clients.
- Immutable mail composition, test delivery, and a secure SMTP starter.
- Last-known-good development supervision and cross-platform restart.
- Starter SDK and third-party annotation SDK.
- Shared overlay analysis, language server, primary GoLand integration, and
  supported Zed integration.

## Release gates

A milestone does not complete because APIs exist. It completes when:

- Reference applications run.
- Integration tests exercise real behavior.
- Error messages are actionable.
- Generated code is inspectable.
- Documentation supports a new developer without tribal knowledge.
- Benchmarks show acceptable startup, build, and runtime overhead.
