# Spice Pipeline Audit — 2026-07-24

## Observed failure pattern

The original staggered scheduled-task pipeline improved continuity, but it still produced excessive churn:

- comment-based leases were not atomic and produced duplicate implementation PRs;
- issue #8 grew to 32 commits and 91 changed files, including a generated vendor tree;
- the verifier repeatedly discovered one additional contract edge case after each handoff;
- the researcher continued merging documentation while the implementation lane remained open;
- temporary workflow and transport PRs were repeatedly created because the sandbox could not clone GitHub directly;
- scheduled runs sometimes overlapped or failed, while other tasks could not see a definitive live scheduler status.

## Root causes

1. **Coordination by comments:** two tasks could both read an unlocked state before either comment became visible.
2. **Oversized implementation slice:** package loading, type identity, source provenance, cgo behavior, diagnostics, cancellation and vendoring were combined.
3. **Late contract discovery:** property-level invariants were discovered during final verification rather than frozen before coding.
4. **Unbounded verifier mandate:** every plausible edge case could become another merge blocker.
5. **Research outran delivery:** hourly research created useful documents but increased main-branch churn while the active PR aged.
6. **Ad hoc workspace transport:** temporary workflows became part of implementation diffs.

## Pipeline v2 corrections

- Atomic compare-and-swap writer and reviewer leases in `agent-state:delivery.json`.
- Issue #32 as a single human-readable delivery dashboard.
- 45-minute renewable leases with 20-minute heartbeats and strict stale-lock recovery.
- Contract freeze, finite test matrix and explicit scope estimate before implementation.
- Default PR sizing guidance: one coherent capability, about eight non-generated files and 800 reviewed non-generated lines.
- Two-cycle review consolidation and split/final-stabilization policy after repeated churn.
- Bounded blocker classes for final verification; non-critical hardening becomes follow-up work.
- Research reduced and write-suppressed while an active lane or sufficient backlog exists.
- Permanent exact-head PR workspace artifact; temporary transport workflows/PRs are prohibited.
- GitHub Actions concurrency cancels superseded CI and artifact runs.

## Current issue #8 stabilization

Issue #8 is grandfathered as a large slice. Its contract is now frozen around:

- the existing loader acceptance criteria;
- the merged collision-free, versioned, length-prefixed stable-ID grammar;
- global uniqueness collision-matrix tests;
- removal of temporary workflow material.

After those items pass, new non-critical edge cases must be filed as follow-ups rather than extending the PR again.
