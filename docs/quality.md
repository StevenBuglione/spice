# Engineering quality

Spice treats verification as part of each repository's public contract. This
core module requires Go 1.26.5 and a Go-owned verifier that behaves the same
from PowerShell, Linux, and macOS.

## Core commands

```text
make fmt       # apply goimports and gofumpt
make check     # boundaries, docs, formatting, modules/vendor, and vet
make coverage  # shuffled race-enabled public tests and the 85% floor
make lint      # allowlisted golangci-lint plus NilAway
make security  # gosec plus govulncheck
make test      # same single race/coverage pass as make coverage
make fuzz      # bounded SDK/config/expression/web fuzz smoke
make offline   # -mod=vendor public tests with network resolution disabled
make verify    # complete core gate
```

Focused `go test` commands remain the fastest edit loop. The complete gate
does not run separate broad ordinary, race, and coverage suites: one shuffled
race-enabled invocation proves all three outcomes and computes aggregate
coverage across exactly 50 public packages.

## Pinned tools

Quality tools live in the isolated `tools` module so they cannot enter the
standard-library-only public module:

| Tool | Pin | Purpose |
| --- | ---: | --- |
| golangci-lint | v2.12.2 | allowlisted static-analysis policy |
| gofumpt | v0.10.0 | canonical formatting |
| goimports / x-tools | v0.48.0 | deterministic import organization |
| gosec | v2.28.0 | source security analysis |
| govulncheck | v1.1.4 | reachable vulnerability analysis |
| NilAway | `f4f8ac24c032dec36186896ecca26c1f232ef777` | nil-flow analysis |

The root `go.mod` intentionally has no third-party requirement or tool
directive. Applications authorize the separately versioned Spice CLI and
annotation tool in their own module.

## Linter policy

`.golangci.yml` starts from `default: none` and allowlists correctness,
error handling, nil flow, contexts, security, documentation, maintainability,
architecture, and suppression-discipline rules. Broadly noisy style policies
such as `lll`, `varnamelen`, `mnd`, `funlen`, `paralleltest`,
`wrapcheck`, and `err113` are deliberately excluded.

Public packages may not import core internals, compiler/CLI paths, or optional
starter implementations. Forbid rules reject debug printing, fatal logging,
and process exit outside the verifier entrypoint. Tests have only narrow,
documented exceptions.

## Ownership of specialized quality gates

Compiler and generator latency/allocation budgets live in
[`spice-framework/toolchain`](https://github.com/spice-framework/toolchain).
Installed-IDE screenshots and interaction tests live in
[`spice-framework/goland`](https://github.com/spice-framework/goland).
Real external-system matrices live with each starter. Spring feedback
comparison and end-to-end generated application acceptance live in
[`spice-framework/petclinic`](https://github.com/spice-framework/petclinic).

The development repository coordinates those independently green artifacts by
exact revision. Core does not clone them or duplicate their implementation
tests.
