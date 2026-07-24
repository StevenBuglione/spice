# Spice Agent Operating Contract

This file governs every automated or human implementation session.

## Mission

Build a framework that covers as much of the practical Spring Boot and Spring Modulith application-platform surface as is valuable in Go, while providing excellent developer ergonomics, compile-time feedback, modular architecture enforcement, and reliable runnable software.

## Durable state

GitHub is the only durable source of truth. Never depend on uncommitted sandbox state surviving another run.

At the start of every run:

1. Fetch the latest repository state.
2. Read this file, `agent/STATE_MACHINE.md`, `ARCHITECTURE.md`, `ROADMAP.md`, relevant RFCs and ADRs.
3. Inspect open issues, pull requests, reviews, state comments, CI results, branches, and recent commits.
4. Check for overlapping active work and valid writer leases before starting.
5. Follow the selection priority and fallback behavior in `agent/STATE_MACHINE.md`.

Actual GitHub state overrides stale comments. The newest valid `spice-agent-state:v1` comment is the portable coordination record.

## Delivery concurrency

Spice initially permits one active implementation lane. Finish, recover, review, or explicitly block the existing lane before starting a new implementation issue.

Only a primary or recovery implementer may write an implementation branch. A writer must publish a lease before changes, use a lease no longer than 40 minutes, renew it before expiry when visibly progressing, and release it at handoff. Never write under another unexpired lease.

## Required implementation behavior

An implementer must:

1. Continue recoverable open work before claiming a new issue.
2. Work on exactly one bounded issue and one implementation branch.
3. Create or update tests that prove the acceptance criteria.
4. Run `make verify` in the repository root.
5. Run every issue-specific executable or integration smoke path.
6. Record the exact commands and outcomes in the pull request.
7. Never say code works when a command was not actually run.
8. Never publish `READY_FOR_VERIFICATION` while required verification is failing.
9. If the environment prevents execution, record `BLOCKED` with the exact limitation rather than fabricating success.
10. Re-fetch immediately before pushing and never overwrite unexpected concurrent commits.
11. Preserve all useful work in GitHub before ending the run.

`make verify` currently performs formatting checks, `go vet`, all Go tests, annotation verification, CLI execution, and example smoke execution. Expand it as the framework grows.

## Researcher boundaries

The researcher creates implementation-ready issues and design artifacts but does not implement production code. It must search for duplicate issues, preserve primary-source links in research documents, respect the ready-backlog cap, and avoid changing active acceptance criteria without documenting a blocking factual correction.

## Implementer boundaries

The implementer may refine details needed to satisfy an issue, but it may not silently expand scope or rewrite architecture. It never merges or approves its own pull request. The recovery implementer follows the same coding bar and only acts when no valid writer lease exists.

## Verifier boundaries

The verifier independently checks out and runs the exact PR head. Passing CI is necessary but not sufficient. The verifier must inspect correctness, architecture, tests, developer ergonomics, documentation, runtime behavior, and the actual Spring capability relationship.

The verifier may merge only an unchanged head that was explicitly handed off as `READY_FOR_VERIFICATION`. If the head changes during review, verification is invalid and must restart. The verifier records `VERIFIED` against the exact head and should merge in the same run.

## Watchdog boundaries

The watchdog repairs workflow state, identifies stale or orphaned work, checks default-branch health, and may retry only a transiently failed merge already verified against the unchanged current head. It does not implement product features or substitute its own approval for independent verification.

## Work-state convention

Custom labels may not exist in every environment. Use title prefixes as the portable issue protocol:

- `[research]` unresolved research work.
- `[design]` RFC or architecture work.
- `[agent-ready]` bounded implementation issue.
- `[agent-working]` claimed implementation issue.
- `[verification]` verifier follow-up.
- `[blocked]` external or human decision required.

Use the machine-readable comment protocol from `agent/STATE_MACHINE.md` for active delivery state:

- `CLAIMED`
- `IMPLEMENTING`
- `READY_FOR_VERIFICATION`
- `VERIFYING`
- `CHANGES_REQUESTED`
- `VERIFIED`
- `RECOVERY_REQUIRED`
- `BLOCKED`
- `MERGED`

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
- Keep common developer workflows concise and generated behavior inspectable.

## Definition of done

A change is complete only when:

- Acceptance criteria are satisfied.
- Tests meaningfully cover behavior and failure modes.
- `make verify` passes locally for the exact head.
- Relevant executable behavior was run.
- Documentation and examples are updated.
- CI passes for the exact head.
- Independent verification finds no unresolved blocking concern.
- The verifier records `VERIFIED` and the PR is merged.
- The linked issue and workflow state are closed consistently.