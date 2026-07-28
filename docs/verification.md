# Verification workflow

Spice separates fast developer feedback from commit and release acceptance
without changing the quality requirements.

Use `make check` while editing. It verifies the exact Go toolchain, formatting,
module tidiness, vet, the allowlisted linter and NilAway policy, and ordinary
compilation of every package and test. Run the feature package's focused tests
alongside it while editing; `make verify` remains responsible for executing
the complete repository suite. Warm runs reuse Go's build and test cache.

Use `make coverage` when a slice adds meaningful production code. It computes
the exact whole-repository 85% floor without also launching IDE, security,
offline, race, fuzz, or executable suites.

Use `make verify` before every commit. After deterministic formatting,
module, and vendor prerequisites, independent analysis, security, editor,
and Zed stages run with at most four workers. Installed GoLand verification
runs when editor, compiler/LSP, annotation SDK, Go module/vendor, or commerce
UI-fixture inputs changed. Broad Go compilation stages then run sequentially
to reuse build caches; measurements showed that running race, coverage, fuzz,
and offline builds together oversubscribes the Go compiler and increases wall
time. Every applicable stage remains mandatory. Stage start and completion
lines include durations so regressions are visible.

Use `make verify-release` for release automation and explicit release
ceremonies. It always includes installed GoLand verification regardless of
changed paths, so commit-time dependency scoping cannot weaken release
acceptance.

No command downloads modules implicitly. A failed or canceled stage never
produces a green result.
