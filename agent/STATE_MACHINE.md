# Spice Autonomous Delivery State Machine

This document governs every scheduled Spice task. GitHub is the only durable state store.

Read `agent/LOCK_PROTOCOL.md` first. `delivery.json` on branch `agent-state` is the authoritative machine state. Issue #32 is its human-readable mirror. PR comments are historical evidence, not lock authority.

## Goals

The pipeline must:

- finish existing work before starting new work;
- prevent concurrent writers and overlapping final reviews;
- make interrupted work recoverable;
- independently execute and review every implementation;
- merge verified work promptly;
- avoid oversized slices and late unbounded redesign;
- keep research from outrunning delivery;
- use idle runs for bounded health work or a clean no-op.

## One implementation lane

Spice permits one active implementation lane until an ADR raises the limit.

An active lane is represented in `delivery.json` and normally has an issue, branch and PR. Actual open PRs and branches override stale state. Research is not another implementation lane, but routine research must not create continuous `main` churn while the lane is active.

## States

1. `BACKLOG` — bounded `[agent-ready]` issue.
2. `CONTRACT_FROZEN` — finite public invariants, test matrix, commands and exclusions recorded.
3. `CLAIMED` — issue is `[agent-working]`; branch may not exist.
4. `IMPLEMENTING` — writer lock held or incomplete implementation exists.
5. `READY_FOR_VERIFICATION` — exact head passed all implementer proof and writer lock is released.
6. `VERIFYING` — review lock held for the exact immutable head.
7. `CHANGES_REQUESTED` — a bounded merge blocker must be fixed on the same lane.
8. `SPLIT_REQUIRED` — slice is too large or repeated review churn proves it is not safely bounded.
9. `VERIFIED` — independent proof passed for exact head; merge immediately.
10. `MERGED` — PR merged and issue/state closed.
11. `BLOCKED` — external dependency or human decision prevents progress.
12. `RECOVERY_REQUIRED` — expired or inconsistent work needs a writer.

Every transition atomically updates `delivery.json`, increments `generation`, and mirrors issue #32. PR comments may record detailed evidence but do not replace the state file.

## Atomic locks

### Writer

Only primary and recovery implementers may acquire `writer_lock`.

- Acquire through compare-and-swap using the fetched `delivery.json` blob SHA.
- Refuse while writer or review lock is active.
- Initial lease: at most 45 minutes.
- Heartbeat at least every 20 minutes, before a push, and before a long verification phase.
- A later scheduled run must use a new token and cannot inherit another run's lock.
- Release using the same token.
- Re-fetch before branch push; unexpected commits require `RECOVERY_REQUIRED`.

### Reviewer

Only the verifier may acquire `review_lock`.

- Lane must be `READY_FOR_VERIFICATION`.
- Writer lock must be released.
- PR must be non-draft and current head must equal the ready head.
- Lock records the exact reviewed head.
- Any head change invalidates review.
- Release using the same token.

### Stale locks

The watchdog may clear a lock only after lease expiry plus 15 minutes and only when no recent heartbeat, state generation, branch push, PR update or CI progress exists. It atomically changes the lane to `RECOVERY_REQUIRED`.

## Contract freeze

Before first implementation, the issue or PR must contain `CONTRACT_FROZEN` with:

- `contract_revision`;
- exact public invariants;
- positive and negative regression matrix;
- runnable commands;
- explicit exclusions;
- expected package/file scope.

A factual correction may revise the contract only when it proves a safety, correctness, identity, data-integrity or build blocker. Increment the revision, return the PR to draft, update the matrix, and then resume implementation.

Research and verifier roles cannot casually append acceptance criteria during implementation.

## Selection priority

### Primary implementer

1. `CHANGES_REQUESTED` lane.
2. Failed CI lane.
3. `RECOVERY_REQUIRED` lane.
4. Incomplete `IMPLEMENTING` lane without active lock.
5. Claimed issue with branch/PR gaps.
6. New highest-priority `[agent-ready]` issue only when no lane exists.

Before a new issue, freeze its contract. Never start a second lane.

### Recovery implementer

Act only when:

- no writer or review lock is active; and
- lane is `RECOVERY_REQUIRED`, failed CI, `CHANGES_REQUESTED` with no primary progress for at least 75 minutes, or incomplete without progress for at least 75 minutes.

Do not claim fresh backlog merely because a scheduled run fired. Do not modify `READY_FOR_VERIFICATION`, `VERIFYING`, or `VERIFIED` work.

### Verifier

1. Stable `READY_FOR_VERIFICATION` lane.
2. Readiness claim missing evidence: one consolidated checklist.
3. Active incomplete lane: at most one preflight contract/testability comment per contract revision.
4. No lane: default-branch health and next-issue contract audit.

The verifier blocks only for the blocker classes in `AGENTS.md`. New non-blocking hardening becomes follow-up work.

### Researcher

- If a lane exists or at least two `[agent-ready]` issues exist, default to read-only research and no repository write.
- A durable research correction during an active lane is allowed only when it resolves an explicit current blocker.
- Do not create a routine research PR every run.
- At most two ready issues normally; three only when explicitly justified.

### Watchdog

1. Retry a transient merge only for current exact-head `VERIFIED` state.
2. Reconcile actual state, atomic file and issue #32 mirror.
3. Clear stale locks under the strict stale-lock rules.
4. Mark stalled lanes `RECOVERY_REQUIRED`.
5. After two `CHANGES_REQUESTED` cycles, evaluate issue size:
   - publish one final bounded stabilization matrix; or
   - set `SPLIT_REQUIRED` and require decomposition.
6. Repair duplicate lanes and linkage without overwriting work.
7. Check default-branch health when idle.
8. Request backlog replenishment only when no lane and fewer than two ready issues exist.

## Verification-cycle budget

`delivery.json.lane.verification_cycles` increments on every `CHANGES_REQUESTED` decision.

- Cycle 1: normal correction.
- Cycle 2: verifier must consolidate all known blockers into one frozen matrix.
- Cycle 3+: watchdog must set `SPLIT_REQUIRED` unless a final stabilization exception is explicitly recorded with a finite checklist.

A final stabilization exception forbids new acceptance expansion. Further novel non-critical findings become follow-up issues.

## PR scope flow control

Before claiming work, estimate reviewed non-generated scope. Split by default when the issue exceeds roughly eight non-generated files, 800 reviewed non-generated lines, or one coherent package capability. Generated vendor trees may be isolated mechanically, but must not hide an oversized handwritten change.

A PR that grows beyond its frozen scope returns to draft. Do not allow a long-lived branch to accumulate unrelated research, temporary workflows or transport utilities.

## Permanent workspace artifact

Use the repository's persistent PR workspace artifact workflow when direct GitHub networking is unavailable. Never create temporary workflow files or transport PRs. Confirm artifact SHA before executing it.

## Merge guarantees

Merge only when:

- frozen contract is satisfied;
- verifier independently ran `make verify` and required runtime commands;
- CI passed for exact reviewed head;
- review lock targets unchanged head;
- no blocker remains;
- verifier records `VERIFIED` and merges in the same run.

The watchdog may retry only a transiently failed merge of the same verified head.

## Recovery thresholds

- Lock lease: 45 minutes, renewable with heartbeat.
- Stale-lock grace: 15 minutes.
- Primary progress timeout before recovery eligibility: 75 minutes.
- Ready PR: next verifier run.
- No visible lane progress: 90 minutes triggers watchdog reconciliation.
- Orphan claim without branch/PR: two hours.

Standing down because another atomic lock is active is correct. A no-op is better than duplicate work.
