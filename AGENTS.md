# Spice Agent Operating Contract

This file governs every automated or human implementation session.

## Mission

Build a framework that covers as much of the practical Spring Boot and Spring Modulith application-platform surface as is valuable in Go, while providing excellent developer ergonomics, compile-time feedback, modular architecture enforcement, and reliable runnable software.

## Durable state

GitHub is the only durable source of truth. Never depend on uncommitted sandbox state surviving another run.

At the start of every run:

1. Fetch the latest repository state.
2. Read this file, `agent/STATE_MACHINE.md`, `agent/LOCK_PROTOCOL.md`, `ARCHITECTURE.md`, `ROADMAP.md`, and relevant RFCs/ADRs.
3. Inspect open issues, pull requests, reviews, state comments, CI, branches, recent commits, and `delivery.json` on branch `agent-state`.
4. Reconcile actual GitHub state before taking action.
5. Follow the role priority and fallback behavior in `agent/STATE_MACHINE.md`.

Actual issue, PR, branch, commit and CI state overrides stale metadata. The authoritative lock and lane record is `agent-state:delivery.json`. State comments are an audit trail and issue #32 is the human-readable mirror.

## Atomic delivery coordination

Scheduled tasks cannot rely on clock offsets or a live scheduler-status API to detect another running task. Writer and reviewer ownership therefore use the compare-and-swap protocol in `agent/LOCK_PROTOCOL.md`.

Only a primary or recovery implementer may hold `writer_lock`. Only the verifier may hold `review_lock`. A task must acquire its lock atomically before changing a branch or beginning final review. A `409 Conflict` means another task won the race; refetch and stand down. Never clear another run's unexpired lock or reuse its token.

## Delivery concurrency

Spice permits one active implementation lane until an ADR deliberately raises the limit. Finish, recover, review, merge, split, or explicitly block the current lane before starting another implementation issue.

Research branches are not implementation lanes, but routine research must not continuously change `main` while a delivery lane is active. See the researcher boundaries below.

## Contract freeze and issue sizing

An implementation issue must be bounded before code begins.

A normal issue should target:

- one coherent compiler/runtime capability;
- one primary package or a very small package set;
- no more than roughly eight non-generated files;
- no more than roughly 800 reviewed non-generated changed lines;
- one implementer run under normal conditions.

Generated dependency trees and generated artifacts do not count toward line limits, but they must be isolated and mechanically reproducible. When a proposed change materially exceeds these bounds, split it before implementation or record an explicit `large-slice-approved` rationale in the issue.

Before first implementation, publish a `CONTRACT_FROZEN` checklist containing:

- exact public invariants;
- positive and negative test matrix;
- runnable commands;
- explicit out-of-scope cases;
- contract revision number.

After freeze, acceptance criteria may change only for a reproducible blocking factual correction affecting safety, data integrity, identity uniqueness, documented public behavior, or build correctness. Any correction increments `contract_revision`, returns the PR to draft, and updates the frozen test matrix before more code is written.

## Required implementation behavior

An implementer must:

1. Continue recoverable open work before claiming a new issue.
2. Work on exactly one bounded issue and the canonical implementation branch/PR.
3. Acquire the atomic writer lock before modifying code.
4. Add tests that prove the frozen contract and verifier-requested regressions.
5. Run `make verify` in the repository root.
6. Run every issue-specific executable or integration path.
7. Record exact commands and actual outcomes in the PR.
8. Never say code works when a command was not run.
9. Never publish `READY_FOR_VERIFICATION` while required verification is failing.
10. If execution is impossible, record `BLOCKED` with the exact limitation.
11. Heartbeat the writer lock during long work and release it at handoff.
12. Re-fetch immediately before pushing and never overwrite unexpected commits.
13. Preserve useful work in GitHub before ending the run.
14. Never add temporary transport/export workflows to an implementation branch.

`make verify` performs formatting checks, `go vet`, all Go tests, annotation verification, CLI execution, and example smoke execution. Expand it as the framework grows.

## Permanent workspace transport

When direct sandbox GitHub networking is unavailable, use the repository's permanent read-only workspace artifact workflow. Do not create temporary workflow files or transport pull requests. Artifacts are transport only; the verifier must still confirm the artifact's head SHA and execute the code independently.

## Researcher boundaries

The researcher does not implement product code and does not manufacture work.

- Keep at most two `[agent-ready]` issues by default; three is an emergency maximum.
- When an implementation lane exists or two ready issues already exist, perform read-only research unless a current verifier blocker explicitly requires a durable contract correction.
- Do not open routine research PRs merely because the task ran.
- Prefer updating a research tracking issue or returning a no-op report while the delivery lane is active.
- Do not alter an active frozen contract except under the factual-correction rule above.
- Preserve primary-source links, dates, limitations and license implications.

## Implementer boundaries

The implementer may refine internal details needed to satisfy the frozen contract, but may not silently expand scope or redesign architecture. It never merges or approves its own PR. The recovery implementer follows the same coding bar and acts only under the recovery conditions in the state machine.

## Verifier boundaries

The verifier independently executes the exact PR head. Passing CI is necessary but insufficient.

A finding blocks merge only when it is one of:

1. a frozen acceptance criterion or public invariant is unmet;
2. a reproducible correctness, security, data-loss or build defect exists;
3. a documented identity/determinism guarantee is violated;
4. the change introduces a regression;
5. required test or runtime evidence is absent or not meaningful.

New hardening ideas outside the frozen contract become follow-up issues unless they prove one of those blocker classes. Do not turn final verification into an unbounded redesign phase.

After two `CHANGES_REQUESTED` cycles, the watchdog must assess whether the issue is oversized. Further cycles require either a focused final stabilization checklist or `SPLIT_REQUIRED`; the verifier may not keep expanding the contract one edge case at a time.

The verifier records `VERIFIED` against the exact unchanged head and merges in the same run whenever possible.

## Watchdog boundaries

The watchdog reconciles `delivery.json`, issue #32, actual GitHub state, stale locks, abandoned work and merge failures. It may mark `SPLIT_REQUIRED` or `RECOVERY_REQUIRED`, but does not implement product code or substitute for verification.

## Work-state convention

Title prefixes remain a portable human protocol:

- `[research]`
- `[design]`
- `[agent-ready]`
- `[agent-working]`
- `[verification]`
- `[blocked]`
- `[pipeline]`

Machine lane states are:

- `BACKLOG`
- `CONTRACT_FROZEN`
- `CLAIMED`
- `IMPLEMENTING`
- `READY_FOR_VERIFICATION`
- `VERIFYING`
- `CHANGES_REQUESTED`
- `SPLIT_REQUIRED`
- `VERIFIED`
- `RECOVERY_REQUIRED`
- `BLOCKED`
- `MERGED`

## Code standards

- Prefer small packages with clear ownership.
- Return errors with actionable context.
- Keep deterministic output stable across runs.
- Preserve source positions for diagnostics.
- Avoid global mutable registries when generation can produce explicit code.
- Add dependencies only with documented rationale.
- Use table-driven tests where they improve clarity.
- Test invalid inputs and failure behavior, not only happy paths.
- Maintain compatibility with ordinary `go test`, `go vet`, `gofmt`, and debuggers.
- Keep common workflows concise and generated behavior inspectable.

## Definition of done

A change is complete only when:

- the frozen contract is satisfied;
- tests meaningfully cover behavior and failure modes;
- `make verify` passes locally for the exact head;
- relevant executable behavior was run;
- documentation and examples are current;
- CI passes for the exact head;
- independent verification finds no blocker under the bounded blocker rules;
- the verifier records `VERIFIED` and merges;
- the linked issue, atomic state file and control issue are closed consistently.
