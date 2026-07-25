# Governance

Spice is currently maintained by Steven Buglione. Architectural and public-contract decisions are recorded as RFCs and ADRs in this repository.

## Decision principles

The project favors:

- Evidence over framework fashion.
- Go-native outcomes over literal Spring translations.
- Broad practical capability coverage over class-name compatibility.
- Compile-time feedback and inspectable generated code over runtime magic.
- Secure, observable defaults with documented escape hatches.
- Small runtime and optional integrations over a dependency-heavy core.

## Delivery

The maintainer works in single-writer mode directly on local `main`. Every commit must pass the complete local `make verify` gate before it is committed and pushed. Work is split into coherent green commits even when it belongs to a larger completion program.

GitHub is the durable mirror for commits, issues, discussions, releases, and post-push CI. GitHub Actions provides additional platform evidence but does not replace or delay the mandatory local gate. The project does not use autonomous writer/reviewer leases, an `agent-state` branch, scheduled delivery roles, or workspace transport artifacts.

Before pushing, the writer fetches `origin/main` and refuses to overwrite unexpected commits. Security-sensitive or compatibility-changing decisions require an RFC or ADR and explicit negative tests.

## Releases

Spice follows semantic versioning. It remains pre-1.0 while public contracts are still evolving. A v1.0 release requires the published compatibility policy, resolved M0-M4 coverage classifications, supported migration guidance, complete release artifacts, and a clean security review.
