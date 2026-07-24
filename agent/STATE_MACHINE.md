# Spice Autonomous Delivery State Machine

This document is the shared coordination protocol for every scheduled Spice task. GitHub is the only durable state store. A sandbox, chat transcript, or task-local memory is never authoritative.

## Goals

The pipeline must:

- finish existing work before starting new work;
- prevent two writers from changing the same branch concurrently;
- make incomplete work recoverable by a later run;
- ensure every implementation is independently reviewed and executed;
- merge verified work promptly;
- avoid creating backlog faster than it can be delivered;
- use otherwise idle runs for bounded health or preflight work rather than inventing unrelated scope.

## One active delivery lane

Until the project deliberately raises the limit through an ADR, Spice permits at most one active implementation lane.

An active lane is any of:

- an issue prefixed `[agent-working]`;
- an open `agent/issue-*` pull request;
- an open implementation pull request linked to an agent issue;
- a branch identified by the latest machine-readable agent state comment.

Research may continue while a delivery lane is active, but no second implementation lane may be started.

## Work states

The logical states are:

1. `BACKLOG` — bounded issue prefixed `[agent-ready]`.
2. `CLAIMED` — issue prefixed `[agent-working]`; branch may not exist yet.
3. `IMPLEMENTING` — branch or draft PR exists and a writer lease may be active.
4. `READY_FOR_VERIFICATION` — implementation reports all required local commands passed, PR is reviewable, and writer lease is released.
5. `VERIFYING` — verifier has acquired the immutable PR head for review.
6. `CHANGES_REQUESTED` — verifier or CI found work that must be addressed on the same branch.
7. `VERIFIED` — verifier recorded evidence against an exact head SHA; merge should happen in the same run when possible.
8. `MERGED` — PR merged and linked issue closed.
9. `BLOCKED` — an external dependency or human decision prevents safe progress.
10. `RECOVERY_REQUIRED` — work exists but its prior writer lease expired or state became inconsistent.

## Machine-readable state comments

Each task that changes delivery state must add a GitHub issue or PR comment containing this block:

```text
<!-- spice-agent-state:v1
role: implementer-primary|implementer-recovery|verifier|watchdog
status: CLAIMED|IMPLEMENTING|READY_FOR_VERIFICATION|VERIFYING|CHANGES_REQUESTED|VERIFIED|BLOCKED|RECOVERY_REQUIRED|MERGED
issue: <number-or-none>
pr: <number-or-none>
branch: <branch-or-none>
head: <commit-sha-or-none>
started_at: <RFC3339>
lease_until: <RFC3339-or-none>
next_action: <single concrete action>
verification: <not-run|running|passed|failed|blocked>
-->
```

The newest valid state comment on the active PR takes precedence. Before a PR exists, the newest valid state comment on the issue takes precedence. Human GitHub actions and actual repository state always override stale comments.

## Writer lease

Only the primary implementer and recovery implementer may hold a writer lease.

A writer must:

1. Inspect the latest state comment, issue, PR, branch head, reviews, and CI.
2. Refuse to write when another unexpired writer lease exists.
3. Publish an `IMPLEMENTING` state with a lease before changing the branch.
4. Use a lease no longer than 40 minutes.
5. Renew the lease with a new state comment before it expires if the same run is still actively writing.
6. Release the lease by publishing `READY_FOR_VERIFICATION`, `CHANGES_REQUESTED`, `BLOCKED`, or `RECOVERY_REQUIRED` with `lease_until: none`.

A lease is stale when its time expired and no branch push or newer state comment shows progress during the last 15 minutes. A stale lease may be recovered by the recovery implementer or normalized by the watchdog.

A writer must re-fetch the branch immediately before pushing and must not overwrite commits it did not create. Unexpected concurrent commits require stopping and recording `RECOVERY_REQUIRED`.

## Stable review handoff

A PR is ready for final verification only when all are true:

- the latest state is `READY_FOR_VERIFICATION`;
- no writer lease is active;
- the PR is not draft;
- the state comment records the exact head SHA;
- required local commands are reported as passed;
- the PR description contains runnable verification evidence;
- the head has not changed since the ready state was published.

The verifier records `VERIFYING` against that exact head SHA. If the head changes during review, the verification result is invalid and the verifier must stop without merging.

## Selection priority

### Primary implementer

Always select work in this order:

1. Oldest open agent PR with `CHANGES_REQUESTED`.
2. Oldest open agent PR with failed CI.
3. Oldest incomplete draft or `IMPLEMENTING` agent PR without a valid writer lease.
4. Oldest `[agent-working]` issue with a known branch but no PR.
5. Oldest `[agent-working]` issue with no branch.
6. Highest-priority `[agent-ready]` issue, but only when no active lane exists.

Never start a new issue while an earlier lane can be recovered.

### Recovery implementer

Use the same order, but act only when:

- no valid writer lease exists; and
- the active lane is incomplete, failed, changes-requested, or stale; or
- no lane exists because the primary implementer did not claim available work.

Do not modify a `READY_FOR_VERIFICATION` PR. Do not create a second implementation lane.

### Verifier

Use this order:

1. Oldest `READY_FOR_VERIFICATION` PR with stable head.
2. Oldest PR claiming readiness but missing required evidence; mark `CHANGES_REQUESTED` with an exact checklist.
3. Active incomplete PR: perform read-only pre-review and leave only immediately actionable findings; do not merge or rewrite the branch.
4. No PR: run default-branch health verification and audit the next `[agent-ready]` issue for testability. Create a follow-up only for a real defect.

### Watchdog

Use this order:

1. Retry a merge only when a verifier recorded `VERIFIED` for the current head and the previous merge failed transiently.
2. Normalize stale or contradictory state.
3. Mark expired abandoned work `RECOVERY_REQUIRED`.
4. Return an orphaned `[agent-working]` issue to `[agent-ready]` only when no branch, PR, or progress exists for at least two hours.
5. Detect open PRs with requested changes, failed CI, or no progress and publish a precise recovery instruction.
6. Verify default-branch CI and `make verify` when the delivery lane is empty.
7. Request backlog replenishment only when no active lane and no `[agent-ready]` issue exist.

The watchdog does not implement product features and does not independently approve unverified code.

## Meaningful idle behavior

Standing down to protect an active lease is correct behavior, not a failure. A task that cannot take its primary action may do only the safe fallback assigned to its role:

- Researcher: improve current research, RFCs, ADRs, or coverage status without creating excess ready issues.
- Primary implementer: inspect CI or prepare a continuation plan; do not touch another lane.
- Recovery implementer: inspect state and leave a recovery plan; do not write under another lease.
- Verifier: perform read-only pre-review or default-branch health verification.
- Watchdog: normalize state and test pipeline health.

No task creates unrelated work merely to appear productive.

## Backlog flow control

The researcher must keep at most three `[agent-ready]` issues. When three already exist, it may update research or refine existing issues but must not create another ready issue. It must not alter an active implementation issue's acceptance criteria unless a blocking factual error is found; such a change requires a clearly documented comment.

## Merge guarantees

A PR may merge only when:

- the linked issue and acceptance criteria are unambiguous;
- the verifier independently ran `make verify` and issue-specific runtime commands;
- GitHub Actions passed for the exact reviewed head;
- the verifier recorded `VERIFIED` for the exact current head;
- no review thread or blocking risk remains;
- the PR still targets the intended default branch.

The verifier should merge in the same run. The watchdog may retry only a transiently failed merge of an already `VERIFIED` unchanged head.

## Recovery deadlines

- Active writer lease: maximum 40 minutes, renewable only with visible progress.
- Ready PR awaiting verification: should be reviewed at the next verifier run.
- Changes-requested or failed-CI PR: should be picked up by the next available writer run.
- No progress on active lane for 90 minutes: watchdog marks `RECOVERY_REQUIRED`.
- Orphan claim with no branch or PR for two hours: watchdog may return it to `[agent-ready]`.

These are coordination thresholds, not permission to bypass correctness or force a merge.