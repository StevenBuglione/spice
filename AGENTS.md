# Spice Agent Operating Contract

This file governs every automated or human implementation session.

## Mission

Build a framework that covers as much of the practical Spring Boot and Spring Modulith application-platform surface as is valuable in Go, while providing excellent developer ergonomics, compile-time feedback, modular architecture enforcement, and reliable runnable software.

## Durable state

GitHub is the only durable source of truth. Never depend on uncommitted sandbox state surviving another run.

At the start of every run:

1. Fetch the latest repository state.
2. Read this file, `ARCHITECTURE.md`, `ROADMAP.md`, relevant RFCs and ADRs.
3. Inspect open issues, pull requests, CI results, and recent commits.
4. Check for overlapping active work before starting.

## Required implementation behavior

An implementer must:

1. Work on exactly one bounded issue.
2. Create or update tests that prove the acceptance criteria.
3. Run `make verify` in the repository root.
4. Run every issue-specific executable or integration smoke path.
5. Record the exact commands and outcomes in the pull request.
6. Never say code works when a command was not actually run.
7. Never open a ready-for-review pull request while required verification is failing.
8. If the environment prevents execution, report the limitation and leave the work blocked rather than fabricating success.

`make verify` currently performs formatting checks, `go vet`, all Go tests, annotation verification, CLI execution, and example smoke execution. Expand it as the framework grows.

## Researcher boundaries

The researcher creates implementation-ready issues and design artifacts but does not implement production code. It must search for duplicate issues and preserve source links in research documents.

## Implementer boundaries

The implementer may refine details needed to satisfy an issue, but it may not silently expand scope or rewrite architecture. It never merges its own pull request.

## Verifier boundaries

The verifier independently checks out and runs the code. Passing CI is necessary but not sufficient. The verifier must inspect correctness, architecture, tests, developer ergonomics, documentation, and the actual runnable path.

## Work-state convention

Custom labels may not exist in every environment. Use title prefixes as the portable state protocol:

- `[research]` unresolved research work.
- `[design]` RFC or architecture work.
- `[agent-ready]` bounded implementation issue.
- `[agent-working]` claimed implementation issue.
- `[verification]` verifier follow-up.
- `[blocked]` external or human decision required.

## Code standards

- Prefer small packages with clear ownership.
- Return errors with actionable context.
- Keep deterministic output stable across runs.
- Preserve source positions for compiler diagnostics.
- Avoid global mutable registries when generation can produce explicit code.
- Add dependencies only with a documented reason.
- Use table-driven tests where they improve clarity.
- Test invalid inputs and failure behavior, not only happy paths.
- Maintain compatibility with ordinary `go test`, `go vet`, `gofmt`, and debuggers.

## Definition of done

A change is complete only when:

- Acceptance criteria are satisfied.
- Tests meaningfully cover behavior and failure modes.
- `make verify` passes.
- Relevant executable behavior was run.
- Documentation and examples are updated.
- CI passes.
- Independent verification finds no unresolved blocking concern.
