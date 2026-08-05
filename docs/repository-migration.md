# Spice ecosystem migration ledger

This ledger turns [ADR 0012](../adrs/0012-multi-repository-product-boundaries.md)
into bounded, reversible delivery stages. A checked box requires the stated
evidence; repository creation alone never completes a stage.

## Baseline

- Canonical source commit: `9a83a298c4e37a780b2f596f099ec137158fc298`.
- Current canonical remote: `github.com/StevenBuglione/spice`.
- Target organization: `github.com/spice-framework`.
- The target organization is active, initially contains no repositories, and
  the maintainer has organization administration access.
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

## Measured coupling to remove

The current Go import graph identifies these extraction blockers:

- `compiler/starter` imports the aggregate root `starter` catalog. Descriptor
  and manifest validation must consume public generic metadata rather than a
  compiled registry of every integration.
- `internal/spicegen/commerce` imports the Commerce example. The entire owned
  Commerce target moves with the application before `internal` can belong
  exclusively to the toolchain.
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
- [ ] Move the root-owned Commerce generated target and manifest into the
  Commerce module.
- [ ] Remove compiler dependency on the aggregate starter catalog.
- [ ] Add clean-room application scaffolding and dependency-add commands with
  previewable module changes.

Exit evidence: Petclinic and Commerce generate, build, run, debug, and test
using only their package-main path plus target selection when genuinely
ambiguous.

## Stage 2: Establish organization infrastructure and canonical paths

- [ ] Create `spice-framework/.github` with the organization profile, security
  contacts, contribution policy, and reusable Go/Gradle workflows.
- [ ] Create `spice-framework/development` with idempotent bootstrap tooling,
  native workspace generation, compatibility metadata, and cross-repository
  verification.
- [ ] Rewrite module, annotation import, documentation, and generated
  provenance paths to `github.com/spice-framework`.
- [ ] Transfer the original repository to `spice-framework/spice` and verify
  Git redirects, default branch, rules, Actions, issues, and local remotes.
- [ ] Publish a documented temporary migration tag only if clean-room module
  resolution requires one.

Exit evidence: a clean machine resolves the canonical core path without a
personal-account replacement.

## Stage 3: Extract independent consumers first

- [ ] Extract `goland` with history, package the plugin, run Plugin Verifier,
  and execute the installed Windows/Linux UI matrix against a released CLI.
- [ ] Extract `zed` with history and verify LSP behavior within Zed's supported
  presentation ceiling.
- [ ] Extract `petclinic` and `commerce` with history; remove unpublished
  replacements from their release acceptance.
- [ ] Make reference applications test the minimum and current compatible core
  and toolchain versions.

Exit evidence: both reference applications and both editors are external
consumers of canonical artifacts.

## Stage 4: Extract external-service starters

For each starter repository:

- [ ] filter and preserve relevant source history;
- [ ] add an independent Go module, license, support matrix, ownership, and
  dependency review;
- [ ] add fast unit verification and Docker-backed integration verification;
- [ ] prove cancellation, timeout, retry, cleanup, and observability behavior;
- [ ] verify the minimum and current compatible core versions;
- [ ] publish checksums, SBOM/provenance, and a signed preview tag;
- [ ] remove the durable source from the core repository only after the remote
  and clean-room consumer are green.

Extraction order follows dependency complexity:

1. `starter-smtp`;
2. `starter-postgres` and `starter-mysql`;
3. `starter-redis`;
4. `starter-observability`;
5. `starter-security`;
6. `starter-websocket` and `starter-grpc`;
7. `starter-kafka`.

Exit evidence: importing core alone selects none of the starter client
libraries and every advertised production starter has real-system results.

## Stage 5: Extract and harden the toolchain

- [ ] Move compiler implementation behind supported internal boundaries.
- [ ] Decompose provider, application, service, generation, LSP, filesystem,
  and quality-gate hotspots by domain responsibility.
- [ ] Extract `toolchain` with relevant history and retain the recoverable
  ordinary-Go stage-zero bootstrap.
- [ ] Keep official descriptor handlers typed and navigable while the tool
  package path points at the canonical toolchain module.
- [ ] Preserve deterministic generation and source maps across repository
  boundaries.
- [ ] Complete honest toolchain dogfooding without claiming that parser or
  renderer infrastructure is runtime-managed application code.
- [ ] Add cold CLI, first analysis, generation, structural edit, LSP latency,
  dev restart, startup, memory, and allocation budgets.

Exit evidence: the core module has no compiler dependency, the toolchain builds
from published core contracts, and damaged generated toolchain output is
recoverable by stage zero.

## Stage 6: Preview release

- [ ] Add risk-weighted coverage floors for critical packages in each repo.
- [ ] Run the coordinated Windows, Linux, macOS, amd64, and arm64 matrix.
- [ ] Run real PostgreSQL, MySQL, Redis, Kafka, SMTP, OIDC, gRPC, and WebSocket
  acceptance for every starter presented as supported.
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
| `.zed/settings.json` | User-facing supported editor configuration | Move with the Zed repository or replace with documented workspace setup |
| `.spice/*.manifest.json` | Generated ownership metadata | Move with the generated target; never use for plugin or repository selection |
| generated Commerce target | Application-owned generated source currently under root `internal` | Move with Commerce and regenerate before toolchain extraction |
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
