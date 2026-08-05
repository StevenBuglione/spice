# Verification workflow

The core repository uses one cross-platform Go verifier. It is intentionally
smaller than the toolchain verifier because this module owns public libraries,
not compiler, CLI, LSP, generated application, editor, or release-construction
implementation.

## Feedback loop

Use focused package tests while editing, then:

```text
make check     # version, boundaries, docs, formatting, tidy/vendor, and vet
make lint      # allowlisted golangci-lint plus NilAway
make security  # gosec plus govulncheck
make fuzz      # short parser/decoder/validation fuzz smoke
make test      # one shuffled race-enabled public-package pass plus coverage
make offline   # public packages with -mod=vendor and all network resolution off
make verify    # complete core commit gate
```

`make test` deliberately combines race testing and coverage in one invocation
across the exact 50 public packages. It enforces at least 85% aggregate
public-source statement coverage without adding the repository-only quality
gate to the denominator.

The bounded fuzz phase executes 100 inputs for the SDK protocol and starter
manifest, configuration JSON decoder, expression parser, and web JSON decoder.
It protects high-risk parser and validation surfaces without duplicating a
broad test pass.

## Module and offline policy

The root module is standard-library-only. Verification runs root and tools
`go mod tidy -diff`, requires the root module graph to contain only
`github.com/spice-framework/spice`, refuses a committed `vendor` directory,
and reproduces the expected empty vendor result. The isolated `tools` module
pins quality binaries without entering the public runtime graph.

The offline phase sets `GOPROXY=off`, `GOSUMDB=off`, `GOWORK=off`, and
`GOTOOLCHAIN=local`, then tests every public package with `-mod=vendor`.
Because core has no third-party dependencies, this is a literal vendor-only
product test even though no vendor directory is necessary.

## Repository boundary checks

The verifier rejects compiler, CLI, LSP, bootstrap, generated-toolchain,
release-builder, benchmark-baseline, and fixture ownership in core. It also
checks the canonical module namespace, the external official annotation-tool
path, public import direction, strict API maturity metadata, and complete Spring
coverage dispositions.

The separately versioned repositories own their specialized gates:

- `toolchain`: compiler, generator, CLI, LSP, dev loop, generated output,
  bootstrap, dogfooding, performance budgets, and release construction;
- `goland` and `zed`: packaged-editor and interaction acceptance;
- `starter-*`: dependency review, real-service, offline, and compatibility
  matrices;
- `petclinic` and `commerce`: complete generated application workflows;
- `development`: exact-revision cross-repository compatibility.

Run `make verify` on the exact tree before a core commit. Release decisions
add the coordinated ecosystem evidence; they do not weaken or bypass this gate.
