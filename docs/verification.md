# Verification workflow

Spice separates fast developer feedback from commit and release acceptance
without changing the quality requirements.

Use `make check` while editing. It verifies the exact Go toolchain, formatting,
module tidiness, vet, the allowlisted linter and NilAway policy, and ordinary
compilation of every package and test in both the framework module and the
independent Petclinic consumer module. Run the feature package's focused tests
alongside it while editing; `make verify` remains responsible for executing
the complete repository suite. Warm runs reuse Go's build and test cache.

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

Use `make verify` before every commit. After deterministic formatting,
module, and vendor prerequisites, independent analysis, security, editor,
and Zed stages run with at most four workers. Installed GoLand verification
runs when editor, compiler/LSP, annotation SDK, Go module/vendor, or commerce
UI-fixture inputs changed. Broad Go compilation stages then run sequentially
to reuse build caches; measurements showed that running race, coverage, fuzz,
and offline builds together oversubscribes the Go compiler and increases wall
time. The Petclinic consumer module is independently linted, vetted,
security-scanned, shuffled/race tested, coverage exercised, and tested from its
own vendor tree. Its generated target is also verified and executed using a
repository-built Spice binary. Every applicable stage remains mandatory. Stage
start and completion lines include durations so regressions are visible.

Use `make verify-release` for release automation and explicit release
ceremonies. It always includes installed GoLand verification regardless of
changed paths, so commit-time dependency scoping cannot weaken release
acceptance. It also enforces the benchmark budgets. After it passes, the
repository-owned `cmd/spice-release` command
creates deterministic cross-platform archives, an SPDX SBOM, signed SHA-256
checksums, and a public verification key as documented in
[`releasing.md`](releasing.md).

No command downloads modules implicitly. A failed or canceled stage never
produces a green result.
