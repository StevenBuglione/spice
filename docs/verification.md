# Verification workflow

Spice separates fast developer feedback from commit and release acceptance
without changing the quality requirements.

Use `make fast` after an edit. It derives affected packages from Go's package
and test-import graph rather than a maintained allowlist, includes cross-module
reverse dependencies, and fails safe to broader work whenever ownership is
uncertain. This target is an edit-time correctness loop, not the commit gate.
When Git reports no relevant paths, the planner returns before loading any Go
package graph, keeping the no-op check independent of repository size.

Use `make check` while editing. It verifies the exact Go toolchain, formatting,
module tidiness, vet, the allowlisted linter and NilAway policy, and ordinary
compilation of every package and test in the framework module. Independent
consumers run their repository-owned gates against immutable Spice versions.
Run the feature package's focused tests alongside it while editing; `make
verify` remains responsible for executing the complete core suite. Warm runs
reuse Go's build and test cache.

Use `make coverage` when a slice adds meaningful production code. It computes
the exact whole-repository handwritten-product 85% floor without also launching
IDE, security, offline, race, fuzz, or executable suites. Canonical Spice
generated files remain mandatory compilation/execution inputs but are excluded
from the duplicate statement denominator.

Use `make benchmark` when compiler, generation, editor-analysis, or CLI
boundary behavior changes. The repository runs each budgeted benchmark five
times for 500 milliseconds, takes the median of time, bytes, and allocations,
and compares it with the reviewed ceilings in `benchmarks/budgets.json`.
Changing a ceiling requires an adjacent engineering rationale; a noisy
single sample cannot fail or conceal a regression.

The standalone
[`spice-framework/petclinic`](https://github.com/spice-framework/petclinic)
repository owns the pinned offline body-edit comparison documented in
[`spring-speed-parity.md`](spring-speed-parity.md). The core verifier never
clones or resolves its external Spring checkout.

`spice dev` independently fingerprints the valid-Go structure of watched
sources. A change confined to function or method bodies reuses the last
validated immutable generation plan and proceeds directly to `go build`.
Annotation comments, imports, declarations, signatures, fields, types, and
top-level values remain in the fingerprint, so any compiler-relevant edit
still performs complete analysis and guarded generation. The development
event stream reports generation/build duration and whether structural reuse
occurred.

Use `make verify` before every commit. After deterministic formatting, module,
and vendor prerequisites, independent analysis and security stages run with at
most four workers. The shuffled test pass emits the repository coverage
profile, eliminating a redundant all-package compilation while retaining the
same tests and 85% floor. Race, fuzz, offline, and executable stages then run
sequentially to reuse build caches; running them concurrently oversubscribes
the Go compiler and increases wall time. Petclinic independently lints, vets,
security-scans, shuffled/race tests, coverage-checks, and vendor-offline tests
its own module, then verifies and executes all three generated targets through
its pinned Spice tool dependency. Every applicable stage remains mandatory.
Stage start and completion lines include durations so regressions are visible.

Use `make verify-release` for release automation and explicit release
ceremonies. It adds the benchmark budgets to the complete core verification.
The independent GoLand repository owns Plugin Verifier plus installed-IDE
Windows/Linux acceptance against exact core and Petclinic commits. After the
coordinated compatibility tuple passes, the repository-owned `cmd/spice-release` command
creates deterministic cross-platform archives, an SPDX SBOM, signed SHA-256
checksums, and a public verification key as documented in
[`releasing.md`](releasing.md).

No command downloads modules implicitly. A failed or canceled stage never
produces a green result.
