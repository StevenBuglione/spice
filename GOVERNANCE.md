# Governance

Spice is currently maintained by Steven Buglione and published through the
[`spice-framework`](https://github.com/spice-framework) GitHub organization.
Architectural and public-contract decisions are recorded as RFCs and ADRs in
the repository that owns the affected contract. Ecosystem-wide decisions live
in the core `spice` repository.

## Decision principles

The project favors:

- Evidence over framework fashion.
- Go-native outcomes over literal Spring translations.
- Broad practical capability coverage over class-name compatibility.
- Compile-time feedback and inspectable generated code over runtime magic.
- Secure, observable defaults with documented escape hatches.
- Small runtime and optional integrations over a dependency-heavy core.

## Repository ownership

Spice uses the bounded product repositories defined by
[ADR 0012](adrs/0012-multi-repository-product-boundaries.md). Repositories
exist for independent compatibility and verification lifecycles, not for
individual packages. The core runtime, toolchain, editors, external-service
starters, and reference applications have explicit owners and one-way
dependency rules.

The `development` repository coordinates source checkouts through native Go
workspaces and reusable verification. It is not a library dependency manager,
and released applications remain ordinary independent Go modules.

## Delivery

The maintainer works in single-writer mode directly on local `main` in the
repository that owns the change. Every commit must pass that repository's
complete local `make verify` gate before it is committed and pushed. Work is
split into coherent green commits even when it belongs to a coordinated
ecosystem program.

GitHub is the durable mirror for commits, issues, discussions, releases, and post-push CI. GitHub Actions provides additional platform evidence but does not replace or delay the mandatory local gate. The project does not use autonomous writer/reviewer leases, an `agent-state` branch, scheduled delivery roles, or workspace transport artifacts.

Before pushing, the writer fetches `origin/main` and refuses to overwrite
unexpected commits. Cross-repository changes land dependency-first and remain
compatible with the previously published version until consumers are green.
Security-sensitive or compatibility-changing decisions require an RFC or ADR,
explicit negative tests, and migration guidance.

## Releases

Every product repository follows semantic versioning and publishes on its own
compatibility lifecycle. Version numbers are not forced into lockstep. Go
modules use standard module tags; editor plugins use their native artifact
versions and declare supported CLI/LSP protocol ranges.

Spice remains pre-1.0 while public contracts are still evolving. A v1.0 core
release requires the published compatibility policy, resolved capability
classifications, supported migration guidance, complete release artifacts,
clean security review, and external clean-room adoption evidence.
