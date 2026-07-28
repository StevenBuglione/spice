# Spice Implementation Contract

This file governs automated and human implementation sessions in this repository.

## Mission

Build a professional Go-native application platform that covers the valuable practical surface of Spring Boot and Spring Modulith through compile-time validation, deterministic generated Go, explicit dependencies, modular architecture enforcement, and a small runtime.

## Delivery model

Spice currently operates in single-writer mode:

- The active writer works directly on local `main`.
- Local verification is the merge gate. GitHub and GitHub Actions are durability and visibility mirrors, not prerequisites for local progress.
- Do not use delivery locks, lease files, scheduled-agent state machines, transport branches, or workspace-artifact workflows.
- Keep commits coherent and reviewable. A large program must still be delivered as bounded, green commits.
- Fetch and inspect `origin/main` before starting and immediately before pushing. Never overwrite unexpected remote work.
- Preserve useful completed work by committing and pushing it before ending a session.

At the start of a session:

1. Fetch the latest repository state and confirm the current branch and worktree status.
2. Read this file, `ARCHITECTURE.md`, `ROADMAP.md`, `docs/spring-coverage.md`, and relevant RFCs/ADRs.
3. Reconcile open local work and recent commits before beginning a new slice.
4. Confirm `go version` is exactly Go 1.26.5.

## Required implementation behavior

For every product change:

1. State the bounded developer outcome and public invariants before implementation.
2. Preserve valid Go source and the single typed-program compiler pipeline.
3. Add positive, negative, boundary, and deterministic-order tests appropriate to the change.
4. Exercise cancellation, timeout, rollback, concurrency, and failure paths when relevant.
5. Update documentation, examples, the Spring coverage map, and benchmarks when the public behavior changes.
6. Run issue-specific executable or integration paths.
7. Use `make check` for the edit loop, then run `make verify` from the
   repository root on the exact tree to be committed.
8. Never claim a command passed unless it was actually executed.
9. Commit only a green tree. Fetch again before push and stop if `origin/main` moved unexpectedly.

## Quality contract

`make verify` is repository-owned and cross-platform. It enforces:

- Go 1.26.5;
- Rust 1.93.0 formatting, tests, Clippy, and a locked `wasm32-wasip2` release
  build for the isolated Zed adapter;
- goimports and gofumpt formatting;
- `go mod tidy -diff` for the product and tools modules;
- mechanically reproducible vendor contents;
- `go vet`;
- the allowlisted golangci-lint policy in `.golangci.yml`;
- NilAway, gosec, and govulncheck;
- shuffled and race-enabled tests;
- parser, decoder, and validation fuzz smoke;
- an 85% whole-repository coverage floor;
- offline vendor-only tests;
- Spice CLI verification and executable example smoke tests.

`make check` is the fast local feedback loop. `make coverage` isolates the
whole-repository coverage calculation, and `make verify-release` is the
unconditional release alias. See `docs/verification.md`.

Do not weaken, skip, or broadly suppress a gate to land a change. A narrow suppression is acceptable only for a demonstrated false positive, with the reason adjacent to the suppression.

## Code standards

- Prefer small packages with clear ownership and dependency direction.
- Return errors with actionable context and preserve source positions in diagnostics.
- Keep output stable across runs; generated artifacts must not contain timestamps or absolute paths.
- Avoid global mutable containers, runtime package scanning, reflection-based dependency lookup, and hidden network access.
- Resolve dependencies by exact Go type identity. Interface bindings require explicit adapters.
- Register construction cleanup immediately and roll it back in reverse order.
- Keep core packages standard-library-first. Add dependencies only with a documented maintenance, license, security, cancellation, and observability review.
- Use table-driven tests when they improve clarity. Test invalid inputs and failure behavior, not only happy paths.
- Keep ordinary `go test`, `go vet`, debuggers, and vendor-offline builds working.
- Never hand-edit generated files or vendor contents.

## Public contracts

- Application source remains valid Go; annotations are declaration comments.
- Generated behavior is ordinary committed Go and needs no Spice compiler at runtime.
- Core/public packages never import compiler or CLI packages.
- Compiler packages never import CLI entrypoints.
- Starters are isolated and opt-in.
- No provider or application marker body executes during analysis.
- Generation must detect stale ownership and manual edits rather than overwrite destructively.

## Definition of done

A slice is complete only when its contract is implemented, tests prove meaningful success and failure behavior, `make verify` passes locally for the exact commit, relevant executable behavior was run, documentation and examples are current, and the green commit is pushed to `origin/main`.
