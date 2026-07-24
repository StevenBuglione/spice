# Spice Atomic Lease Protocol

GitHub scheduled tasks do not expose a dependable live `is_running` signal to one another. Spice therefore coordinates writers and reviewers through an atomic state file rather than timing assumptions or append-only comments.

## Authority

The authoritative machine state is:

```text
branch: agent-state
path:   delivery.json
```

Issue #32 mirrors that state for people. PR and issue state comments remain an audit trail, but they are not locks.

## Why this is atomic

A task must fetch `delivery.json`, retain the returned blob SHA, and replace the file using that exact SHA. GitHub returns `409 Conflict` when another task changed the file first. A task that receives a conflict lost the race: it must refetch and stand down or retry state resolution. It must never overwrite state using a stale SHA.

## Lock fields

`delivery.json` contains independent `writer_lock` and `review_lock` records. Every acquisition uses a unique run token, for example:

```text
implementer-primary-20260724T152000Z-7f3c2a
```

Each lock records:

- `active`
- `owner`
- `token`
- `acquired_at`
- `heartbeat_at`
- `lease_until`
- reviewed `head` for a review lock

Every state-file update increments `generation`.

## Writer acquisition

A primary or recovery implementer must:

1. Reconcile actual issues, PRs, heads, CI and comments with `delivery.json`.
2. Refuse acquisition while an unexpired writer lock or review lock exists.
3. Generate a unique token.
4. Set the lane and writer lock using an update with the fetched blob SHA.
5. Treat `409 Conflict` as a lost lock race and refetch.
6. Mirror the successful state to the single control comment in issue #32.
7. Only then modify the implementation branch.

The initial lease is at most 45 minutes. Heartbeat before any push, before a long verification phase, and at least every 20 minutes while actively writing. A heartbeat extends the lease by at most 45 minutes and must use the same token.

A later scheduled run by the same role is still a different run and must not reuse another run's token.

## Review acquisition

The verifier may acquire `review_lock` only when:

- the lane is `READY_FOR_VERIFICATION`;
- no writer lock is active;
- the PR is non-draft;
- the ready-state head equals the current PR head.

The verifier atomically sets `review_lock.active`, its unique token, lease timestamps and exact head. A writer cannot acquire while the review lock is active. Any head change invalidates the review.

## Release

The lock holder must release its own lock in the same run whenever possible. Release requires the current token; a task must not clear a lock owned by another token.

Writer release transitions to one of:

- `READY_FOR_VERIFICATION`
- `CHANGES_REQUESTED`
- `BLOCKED`
- `RECOVERY_REQUIRED`

Reviewer release transitions to one of:

- `CHANGES_REQUESTED`
- `VERIFIED`
- `RECOVERY_REQUIRED`

A successful verifier records `VERIFIED` and merges the exact unchanged head in the same run.

## Stale recovery

A lock is not stale merely because another scheduled task started.

The watchdog may clear a lock only when all are true:

1. `lease_until` expired at least 15 minutes ago.
2. No newer state generation or heartbeat exists.
3. No branch push, PR update, workflow start or other visible progress occurred in the last 15 minutes.
4. The watchdog atomically replaces the state using the current blob SHA.

It then sets the lane to `RECOVERY_REQUIRED`. It never clears an unexpired lock.

## Human visibility

After each successful state transition, update the single `<!-- spice-delivery-control:v2 -->` comment in issue #32. Display:

- lane status, issue, PR, branch and head;
- writer/reviewer owner and expiry, or `released`;
- verification-cycle count;
- next action and update time.

Failure to mirror does not invalidate a successfully acquired atomic lock, but the next watchdog should repair the mirror.

## Prohibited coordination

Do not use these as lock authority:

- scheduled clock offsets;
- task chat memory;
- append-only PR comments;
- issue title prefixes;
- branch names;
- CI running status alone.

They remain useful evidence, but only the compare-and-swap state file prevents two tasks from both believing they own the lane.
