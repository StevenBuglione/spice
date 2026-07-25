# Engineering quality

Spice treats local verification as part of the product contract. The repository requires Go 1.26.5 and exposes the same Go-owned verifier through GNU Make on Windows, Linux, and macOS.

## Commands

```text
make fmt       # apply goimports and gofumpt
make lint      # golangci-lint plus NilAway
make security  # gosec plus govulncheck
make test      # shuffled and race-enabled tests
make fuzz      # bounded parser and validation fuzz smoke
make offline   # product tests with GOPROXY=off and vendor only
make smoke     # Spice CLI and executable example checks
make verify    # every required gate
```

`make verify` also checks both modules with `go mod tidy -diff`, regenerates vendor contents into a temporary directory and compares them byte-for-byte, enforces 85% whole-repository statement coverage, and verifies the exact Go toolchain version.

## Pinned tools

Development tools live in the isolated `tools` module so they cannot enter the Spice runtime graph:

| Tool | Pin | Purpose |
|---|---:|---|
| golangci-lint | v2.12.2 | allowlisted static-analysis policy |
| gofumpt | v0.10.0 | canonical formatting |
| goimports / x-tools | v0.48.0 | deterministic import organization |
| gosec | v2.28.0 | source security analysis |
| govulncheck | v1.1.4 | reachable Go vulnerability analysis |
| NilAway | `f4f8ac24c032dec36186896ecca26c1f232ef777` | nil-flow analysis |

The product module separately pins `golang.org/x/tools v0.48.0` because `compiler/load` owns the `go/packages` boundary.

## Linter policy

`.golangci.yml` starts from `default: none` and explicitly enables useful rules for:

- compiler and type correctness;
- unchecked errors and nil flow;
- contexts and HTTP resource handling;
- Unicode and directive safety;
- deterministic, maintainable code;
- documentation and test helpers;
- architecture dependency direction;
- suppression discipline.

Rules that impose high noise or arbitrary local style costs are deliberately not enabled globally: `lll`, `varnamelen`, `mnd`, `funlen`, `paralleltest`, `wrapcheck`, and `err113`. Complexity is bounded in production code while tests are exempt from the global complexity threshold.

`depguard` prevents public/runtime packages from depending on compiler or CLI implementation packages and prevents compiler packages from depending on command entrypoints. `forbidigo` rejects debug printing, fatal logging, and process termination except for explicit command entrypoints and test processes.

Suppressions must name one specific rule and explain the demonstrated false positive next to the code. Broad file, package, or linter exclusions are not acceptable substitutes for fixing findings.

## Security and offline behavior

The standalone gosec scan excludes generated files and does not scan tests as product code. Govulncheck must report no reachable vulnerabilities for the exact product graph. The compiler and runtime never enable network access themselves; offline product tests run with `GOPROXY=off` and `GOFLAGS=-mod=vendor`.

Tools may require their pinned modules to be downloaded on a fresh development machine. Once cached, product compilation and tests remain independent of the tools module.
