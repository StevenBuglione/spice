# Focused module testing

`spice test` runs ordinary Go tests for one verified application-module graph:

```bash
spice test \
  --module=example.com/shop/orders \
  --race \
  --count=1 \
  ./...
```

The module identity is the full import path from its package-level `@Module`.
Package patterns are used for compiler discovery and default to `./...`; they
should include every module root that can participate in the graph.

Before starting `go test`, Spice resolves and validates annotations, builds the
module model, rejects architecture violations, and focuses it on the requested
module. The executed package list contains exactly:

1. transitively observed dependency modules, in dependency-first order;
2. the focused module;
3. every package owned by those modules, sorted within each module.

Dependents, unrelated modules, declared-but-unused dependencies, and unassigned
packages are excluded. Spice passes full package import paths directly to
`go test -trimpath`; it does not create a test container, scan packages at
runtime, generate files, or change application wiring.

Supported test controls are:

| Option | Effect |
|---|---|
| `--race` | Enable the Go race detector. |
| `--count=N` | Run each test and benchmark `N` times; `N` must be positive. |
| `--run=REGEXP` | Run tests matching the standard Go test expression. |
| `--timeout=DURATION` | Apply a positive Go test timeout. |

Invalid command usage exits with status 2. Package loading, annotation,
architecture, focus, and `go test` failures exit with status 1. A fully passing
focused graph exits with status 0.

This command is the Modulith-style package test slice. Generated application
contexts and specialized web/data test harnesses remain separate roadmap work.
