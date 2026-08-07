# Spice ecosystem migration ledger

This ledger turns [ADR 0012](../adrs/0012-multi-repository-product-boundaries.md)
into bounded, reversible delivery stages. A checked box requires the stated
evidence; repository creation alone never completes a stage.

## Baseline

- Canonical source commit: `9a83a298c4e37a780b2f596f099ec137158fc298`.
- Baseline source remote: the maintainer's personal `StevenBuglione` repository.
- Canonical module and target remote: `github.com/spice-framework/spice`.
- Current source remote: `github.com/spice-framework/spice`; the baseline URL
  redirects to the same `main` history.
- Target organization: `github.com/spice-framework`.
- The target organization is active and the maintainer has organization
  administration access.
- Go 1.26.5 is the mandatory migration toolchain.
- The existing untracked `.tmp`, ignored `bin`, and ignored `out` trees are
  local reproducible artifacts, not migration inputs.

## Non-negotiable migration invariants

1. Application source remains valid Go and generated output remains ordinary
   inspectable Go.
2. Published consumers never need the compiler or editor at runtime.
3. The core `spice` module does not select external-service client libraries.
4. Annotation tool authorization remains an ordinary target-module `tool`
   directive.
5. Every extracted repository retains relevant Git history and Apache-2.0
   licensing.
6. No repository becomes authoritative until its local gate and a clean-room
   consumer pass.
7. No source is removed from the current repository until the extracted remote
   commit is durable and independently verified.
8. Releases use ordinary Go module tags or native editor artifacts; no custom
   library dependency resolver is introduced.
9. Security defaults fail closed, and real external integrations are not
   labeled production-ready without real-system evidence.
10. Migration commits remain bounded and green on local `main`.

## Measured coupling and completed boundaries

The migration baseline identified these extraction blockers and their current
disposition:

- `compiler/starter` now consumes the portable `annotation/sdk/starter`
  metadata contract instead of the aggregate runtime `starter` package. The
  compatibility aliases at the old path remain only while integrations are
  extracted and can be removed after their first independent releases.
- The independent `spice-framework/commerce` repository owns its generated
  target, manifest, acceptance tests, module graph, vendor tree, and complete
  application verification; core no longer duplicates that source or gate.
- The independent `spice-framework/petclinic` repository owns its three
  generated targets, manifests, application source, Spring parity benchmark,
  black-box developer-loop proof, database acceptance, module graph, vendor
  tree, and complete verification; core no longer duplicates that source or
  gate.
- The independent `spice-framework/zed` repository owns the Rust/WASM adapter,
  compatibility fixture, locked dependency graph, and exact release gate; core
  no longer duplicates that source, user workspace configuration, or gate.
- application package scope is repeated in CLI arguments. Composition must be
  declared through ordinary Go imports at the application entrypoint so
  extracted applications remain self-describing.
- application acceptance tests import generated targets from dedicated
  black-box packages outside `internal/spicegen/<target>`. The shared quality
  gate rejects every file or non-empty target absent from its ownership
  manifest.

## Stage 0: Correct product truth and security

- [x] Change management endpoint access from public to loopback by default;
  add explicit public-access and forwarding-header negative tests.
- [x] Upgrade `github.com/klauspost/compress` to at least `v1.18.7` and
  `go.opentelemetry.io/otel` to at least `v1.44.0`, then rerun vulnerability
  and compatibility checks.
- [x] Fix the GoLand affected-range calculation for direct pushes.
- [x] Add installed GoLand Run and real Delve breakpoint acceptance on Windows
  and Linux.
- [x] Reclassify capability documentation by maturity and remove claims not
  backed by mandatory evidence.
- [x] Classify exported packages as preview-stable, experimental, or internal.

Exit evidence: the complete current-repository verifier is green and the
security scan contains no known vulnerable selected module versions.

## Stage 1: Make applications independently movable

- [x] Declare application composition through ordinary blank Go imports and
  remove repeated package-pattern arguments from normal Petclinic and Commerce
  workflows.
- [x] Move handwritten tests outside generated ownership roots and enforce that
  every file below a generated target is manifest-owned.
- [x] Render conventional generated interface assertions while preserving
  exact pointer/value/generic validation.
- [x] Move the root-owned Commerce generated target and manifest into the
  Commerce module.
- [x] Move Petclinic's application, generated targets, Spring parity harness,
  and black-box developer-loop proof into its independent repository.
- [x] Remove compiler dependency on the aggregate starter catalog.
- [x] Add clean-room application scaffolding and dependency-add commands with
  previewable module changes.

Exit evidence: Petclinic and Commerce generate, build, run, debug, and test
using only their package-main path plus target selection when genuinely
ambiguous. The mandatory smoke gate also creates a new external application,
proves dependency preview is read-only, applies the exact guarded plan,
generates from zero output, vendors offline, compiles, tests, builds, and runs
the result.

## Stage 2: Establish organization infrastructure and canonical paths

- [x] Create `spice-framework/.github` with the organization profile, security
  contacts, contribution policy, and reusable Go/Gradle workflows.
- [x] Create `spice-framework/development` with idempotent bootstrap tooling,
  native workspace generation, compatibility metadata, and cross-repository
  verification.
- [x] Rewrite module, annotation import, documentation, and generated
  provenance paths to `github.com/spice-framework`.
- [x] Transfer the original repository to `spice-framework/spice` and verify
  Git redirects, default branch, rules, Actions, issues, and local remotes.
- [x] Record that no temporary migration tag was required after clean-room
  canonical pseudo-version resolution passed.

Exit evidence: a clean machine resolves the canonical core path without a
personal-account replacement.

Infrastructure evidence: `spice-framework/.github` publishes inherited
governance and immutable-action reusable workflows. The standard-library-only
`spice-framework/development` command validates its schema-1 catalog, safely
bootstraps exact remotes, guards generated `go.work` and editor workspaces, and
runs repository-owned gates concurrently with `GOWORK=off`. Its local gate
passes race tests, security scans, trimpath builds, and 85.6% coverage. A clean
organization workspace cloned all active and migrating repositories and proved
current guarded workspace output plus independent Development and core
verification; core linter exclusions are stable whether diagnostics are
module-relative or containing-workspace-relative.

Canonical-path evidence: every product, example, annotation descriptor, tool
directive, editor fixture, generated source map, OpenAPI artifact, ownership
manifest, and vendor tree now uses `github.com/spice-framework/spice`. The
repository verifier scans the complete owned tree and rejects the retired
personal-account module namespace. All six generated targets were proven
current before the transition, migrated through their recorded SHA-256
ownership, and rendered again by the independent stage-zero compiler.

Transfer progress: the complete repository and issue history now resides at
`spice-framework/spice`; both canonical and historical Git URLs resolve commit
`9ab6bf3`, the local `origin` uses the organization SSH URL, and `main` rejects
force pushes and deletion without requiring pull requests. Projects, the wiki,
and merge commits are disabled; private vulnerability reporting, dependency
alerts, and automated security fixes are enabled. Post-transfer Actions and
clean-room canonical module resolution are both green.

## Stage 3: Extract independent consumers first

- [x] Extract `goland` with history, package the plugin, run Plugin Verifier,
  and execute the installed Windows/Linux UI matrix against a released CLI.
- [x] Extract `zed` with history and verify LSP behavior within Zed's supported
  presentation ceiling.
- [x] Extract `petclinic` with history and prove its clean-room, offline, SQL,
  and cross-platform acceptance gates.
- [x] Extract `commerce` with history and remove unpublished replacements from
  its release acceptance.
- [ ] Make reference applications test the minimum and current compatible core
  and toolchain versions.

Exit evidence: both reference applications and both editors are external
consumers of canonical artifacts.

Commerce evidence: `spice-framework/commerce` commit `a5346c3` pins immutable
canonical core plus standalone SMTP and PostgreSQL modules with no local
replacement. Hosted run `31035836394` is green on Windows, Linux, and macOS;
its PostgreSQL 18.4 job proves durable transaction, migration, close, and
reopen behavior. Core retains links to this evidence but no longer rebuilds or
vendors the application.

Petclinic evidence: `spice-framework/petclinic` commit `925df3b` pins immutable
canonical core and standalone PostgreSQL source with no local replacement.
Hosted run `31035834661` is green on Windows, Linux, and macOS, including real
PostgreSQL 18.4 and MySQL 8.4.11 jobs. Its repository-owned gate owns the Spring
feedback harness and exercises invalid annotation failure, last-known-good
retention, graceful complete-package restart, generated-source integrity, and
former naked-annotation regressions as a black-box `go tool` workflow. Core
retains links to this evidence but no longer rebuilds or vendors the
application.

## Stage 4: Extract external-service starters

For each remaining starter repository:

- [ ] filter and preserve relevant source history;
- [ ] add an independent Go module, license, support matrix, ownership, and
  dependency review;
- [ ] add fast unit verification and Docker-backed integration verification;
- [ ] prove cancellation, timeout, retry, cleanup, and observability behavior;
- [ ] verify the minimum and current compatible core versions;
- [ ] publish checksums, SBOM/provenance, and a signed preview tag;
- [ ] remove the durable source from the core repository only after the remote
  and clean-room consumer are green.

`starter-smtp` is the first completed source extraction:

- [x] preserve the package history in `spice-framework/starter-smtp`;
- [x] publish an independent Go module, Apache-2.0 license, support matrix,
  ownership contract, and dependency review;
- [x] pass the fast and complete quality contracts plus authenticated,
  required-STARTTLS Mailpit delivery;
- [x] retain cancellation, timeout, conservative retry, cleanup, security, and
  payload-free observation tests;
- [x] verify both the declared minimum and current compatible core versions;
- [ ] publish checksums, SBOM/provenance, and a signed preview tag;
- [x] migrate Commerce to the immutable module and remove the durable core copy
  only after Windows, Linux, macOS, PostgreSQL, and dependency-graph gates pass.

`starter-postgres` is the second completed source extraction:

- [x] preserve the package history in `spice-framework/starter-postgres`;
- [x] publish an independent Go module, Apache-2.0 license, support matrix,
  ownership contract, and pgx dependency review;
- [x] pass the complete quality contract with 85.9% product coverage, offline
  vendoring, and zero reachable vulnerabilities;
- [x] prove transactions, repositories, advisory-locked migrations,
  cancellation, batch restart/leases, durable outbox behavior, and SQL test
  slices against the immutable PostgreSQL 18.4 container digest;
- [x] verify Windows, Linux, macOS, and the real PostgreSQL job in hosted run
  `31034798376` at commit `6310c4b`;
- [x] verify both the declared minimum and current compatible core versions;
- [ ] publish checksums, SBOM/provenance, and a signed preview tag;
- [x] migrate Commerce and Petclinic to the immutable module and remove pgx and
  the durable PostgreSQL implementation from core only after both consumer
  matrices passed.

`starter-mysql` is independently published and consumed by Petclinic:

- [x] preserve its filtered package and dependency-review history;
- [x] publish an independent module, Apache-2.0 license, support policy,
  reproducible vendor graph, and exact core compatibility metadata;
- [x] verify secure configuration, cancellation, cleanup, recovery, and pool
  ownership against the immutable MySQL 8.4.11 container digest;
- [x] pass Windows, Linux, macOS, offline, quality, and real-MySQL hosted jobs;
- [x] migrate Petclinic to the immutable standalone module before removing the
  core implementation and driver graph.

`starter-redis` is independently published at commit `260a181`:

- [x] preserve its filtered client and typed-cache history;
- [x] publish an independent module, Apache-2.0 license, support policy,
  reproducible vendor graph, and minimum/current core compatibility metadata;
- [x] verify authenticated opt-in plaintext configuration, independent pools,
  typed JSON, expiry, cancellation, deletion, and idempotent cleanup against
  the immutable Redis 8.4.0 container digest;
- [x] pass Windows, Linux, macOS, offline, compatibility, quality, and real
  Redis hosted jobs with 94.5% coverage and zero reachable vulnerabilities;
- [x] remove the duplicate core implementation and go-redis graph only after
  the standalone repository was durable and green.

The observability, security-client, RPC/WebSocket, and Kafka extraction wave is
also accepted. Each module preserves filtered history, carries an Apache-2.0
license, support policy, canonical dependency review, reproducible vendor
graph, and strict minimum/current Spice compatibility manifest. The linked
main-branch run is the durable hosted acceptance record; the pseudo-version is
the exact Go module revision accepted for consumer migration.

| Module | Accepted commit and pseudo-version | Hosted evidence | Decisive acceptance |
| --- | --- | --- | --- |
| [`starter-otel`](https://github.com/spice-framework/starter-otel) | `d3a928b22d7b84a216199cf4480038bd2b2c2e71`; `v0.0.0-20260805193847-d3a928b22d7b` | [run `31040425162`](https://github.com/spice-framework/starter-otel/actions/runs/31040425162) | HTTP and module-event spans/metrics, payload-safe attributes, idempotent completion, minimum/current core, offline vendor, and Windows/Linux/macOS |
| [`starter-oauth2client`](https://github.com/spice-framework/starter-oauth2client) | `b7518b4ed9d8ec5ac9548df8f2d6a5c7ff9f06ff`; `v0.0.0-20260805194856-b7518b4ed9d8` | [run `31041215328`](https://github.com/spice-framework/starter-oauth2client/actions/runs/31041215328) | Local TLS token/resource flow, redirect refusal, response limits, cancellation, credential-safe failures, minimum/current core, offline vendor, and Windows/Linux/macOS |
| [`starter-oidc`](https://github.com/spice-framework/starter-oidc) | `d3bbf42c26a1be15ce5efb3ac0cd503f68c85f9e`; `v0.0.0-20260805195025-d3bbf42c26a1` | [run `31041320255`](https://github.com/spice-framework/starter-oidc/actions/runs/31041320255) | Local TLS discovery/JWKS, exact issuer/audience/expiry/signature checks, bounded transport, cancellation, token-safe failures, minimum/current core, offline vendor, and Windows/Linux/macOS |
| [`starter-websocket`](https://github.com/spice-framework/starter-websocket) | `2990064511b4bada03fd61f33c560d62b29544a0`; `v0.0.0-20260805200426-2990064511b4` | [run `31042434587`](https://github.com/spice-framework/starter-websocket/actions/runs/31042434587) | Real local TLS client/server sessions, authentication/origin enforcement, limits, cancellation, graceful/forced close, payload-safe observations, minimum/current core, offline vendor, and Windows/Linux/macOS |
| [`starter-grpc`](https://github.com/spice-framework/starter-grpc) | `b476d3301285ff5265cb4f8039b7a305a5b469fe`; `v0.0.0-20260805200534-b476d3301285` | [run `31042528507`](https://github.com/spice-framework/starter-grpc/actions/runs/31042528507) | Real local TLS/mTLS RPC and health, interceptors, cancellation, concurrency, message limits, graceful/forced cleanup, payload-safe diagnostics, minimum/current core, offline vendor, and Windows/Linux/macOS |
| [`starter-kafka`](https://github.com/spice-framework/starter-kafka) | `2ea33867d1a16e0d0b97ae560e6c59c13f24345a`; `v0.0.0-20260805200634-2ea33867d1a1` | [run `31042594133`](https://github.com/spice-framework/starter-kafka/actions/runs/31042594133) | Authenticated Redpanda delivery, manual commit, restart/no-redelivery, cancellation, cleanup, minimum/current core, offline vendor, Windows/Linux/macOS, and a 43.1-second final local gate with 90.5% coverage; live TLS broker acceptance remains target-owned |

These records complete source ownership and verification, not a signed preview
release. Checksums, SBOM/provenance, and signed preview tags remain Stage 6
release work.

Extraction order follows dependency complexity:

1. `starter-smtp`;
2. `starter-postgres` and `starter-mysql`;
3. `starter-redis`;
4. `starter-otel` (accepted);
5. `starter-oauth2client` and `starter-oidc` (accepted);
6. `starter-websocket` and `starter-grpc` (accepted);
7. `starter-kafka` (accepted).

Exit evidence: importing core alone selects none of the starter client
libraries and every advertised production starter has real-system results.

## Stage 5: Extract and harden the toolchain

- [x] Remove compiler, CLI, LSP, bootstrap, generated-toolchain, fixture,
  benchmark-baseline, and release-construction ownership from the core cutover
  tree.
- [x] Reduce core to its 51 public annotation/runtime/test-support packages, a
  standard-library-only module graph, and one repository-only quality gate.
- [ ] Publish and accept the extracted `toolchain` repository with relevant
  history, independently green verification, and the recoverable ordinary-Go
  stage-zero bootstrap before committing the core cutover.
- [x] Keep official descriptor handlers typed and navigable while
  `annotation/coretool.Path` points at
  `github.com/spice-framework/toolchain/cmd/spice-annotation-core`.
- [ ] Preserve deterministic generation and source maps across repository
  boundaries.
- [ ] Complete honest toolchain dogfooding without claiming that parser or
  renderer infrastructure is runtime-managed application code.
- [ ] Add cold CLI, first analysis, generation, structural edit, LSP latency,
  dev restart, startup, memory, and allocation budgets.

Exit evidence: the committed core module has no compiler dependency, the
public/green toolchain builds from the exact accepted core contracts, and
damaged generated toolchain output is recoverable by stage zero. The prepared
core deletion is not committed or pushed until that external evidence is
recorded.

## Stage 6: Preview release

- [ ] Add risk-weighted coverage floors for critical packages in each repo.
- [ ] Run the coordinated Windows, Linux, macOS, amd64, and arm64 matrix.
- [ ] Run real PostgreSQL, MySQL, Redis, Kafka, SMTP, OIDC, gRPC, and WebSocket
  acceptance for every starter presented as supported. Kafka's authenticated
  Redpanda protocol path is green; a live TLS broker configuration remains a
  target-owned acceptance prerequisite rather than inferred evidence.
- [ ] Publish the compatibility catalog and migration guide.
- [ ] Publish signed preview versions of core, toolchain, editors, and supported
  starters.
- [ ] Build two clean-room applications outside the development workspace.
- [ ] Record cold and warm feedback timings and compare only equivalent Spring
  workflows.

Exit evidence: a new user can install a versioned CLI, create an application,
open it in GoLand, generate, debug, persist data, and deliver test mail without
cloning the Spice development workspace.

## Cleanup ledger

| Candidate | Current classification | Required action |
| --- | --- | --- |
| `.tmp/` | Untracked reference clones and verification output | Remove locally after any still-needed visual evidence is copied to an owned test artifact |
| `bin/`, `out/` | Ignored reproducible binaries, IDE distributions, caches, and logs | Remove locally between verification runs; never migrate |
| `agent/`, `scripts/` | No tracked source | Do not create repositories or migration work for empty directories |
| `research/` | Tracked design evidence | Retain until each document is incorporated or explicitly archived |
| `.zed/settings.json` | User-facing project-local editor configuration | Retired from core; the standalone Zed and core integration guides document an explicit per-project example |
| `.spice/*.manifest.json` | Generated ownership metadata | Move with the generated target; never use for plugin or repository selection |
| core `cmd/`, `compiler/`, `testdata/`, and toolchain-only `internal/` | Toolchain implementation | Remove atomically only after the standalone toolchain revision is public and green |
| generated Commerce target | Owned by `spice-framework/commerce` below `internal/spicegen` | Retain and verify only in the standalone application repository |
| generated Petclinic targets | Owned by `spice-framework/petclinic` below `internal/spicegen` | Retain and verify only in the standalone application repository |
| generated-tree handwritten tests | Useful tests in the wrong ownership boundary | Relocate; do not delete |

## Audit remediation ownership

| Finding | Owning stage/repository |
| --- | --- |
| No consumable release or scaffold | Stages 1 and 6; `toolchain` and `development` |
| Starter dependency contamination | Stage 4; starter repositories |
| CLI-owned application composition | Stage 1; `toolchain`, Petclinic, and Commerce |
| Incomplete GoLand Run/Debug proof | Stages 0 and 3; `goland` |
| GoLand CI affected-range defect | Stage 0; current repo then `goland` |
| Missing mandatory real-service gates | Stages 4 and 6; starter repos and `development` |
| Public management default | Stage 0; `spice` |
| Excessive capability breadth | All stages; freeze additions until preview |
| Large implementation hotspots | Stage 5; `toolchain` |
| Excessive public compiler API | Stages 0 and 5; `toolchain` |
| Partial dogfooding claims | Stages 0 and 5; documentation and `toolchain` |
| Handwritten files in generated roots | Stage 1; applications and generator checks |
| Narrow aggregate coverage margin | Stages 5 and 6; every repository |
| Slow feedback loop | Per-repository gates plus Stages 5 and 6 |
| Documentation overstatement | Stages 0 and 6; `spice` and `development` |
| SDK extension ceiling and trust | Stage 0 documentation; `spice` SDK |
| Incomplete dependency-direction linting | Stage 0 then per-repository verification |
| Known vulnerable selected versions | Stage 0, then automated repository updates |
