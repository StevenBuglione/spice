# Releasing Spice core

This repository publishes the standard-library-only
`github.com/spice-framework/spice` library module. Core releases contain source
and provenance only. CLI binaries and toolchain artifacts are released from
[`spice-framework/toolchain`](https://github.com/spice-framework/toolchain).

## Candidate gate

The exact candidate commit must pass `make verify-release` under Go 1.26.5.
That command is an unconditional alias for the complete core `make verify`
contract; it does not reduce the local gate for a release runner. The protected
central workflow additionally requires that a canonical `vX.Y.Z` or SemVer
prerelease tag resolves to the exact checked-out commit, that the commit is an
ancestor of `origin/main`, and that the reviewed public key is the exact
regular file committed at the tagged tree path.

The release caller is intentionally exact and fail-closed:

- it pins
  `spice-framework/.github/.github/workflows/library-release.yml` at immutable
  commit `9ae80e32f64b29697acd9ebe629468850b4ae9f2`;
- repository-level workflow permissions are empty;
- only the release job receives `contents: write` for final publication; and
- only `SPICE_LIBRARY_RELEASE_SIGNING_KEY` is explicitly forwarded. Secret
  inheritance and additional secret mappings are rejected by the repository
  quality gate.

## Source-only artifact contract

The trusted renderer consumes the exact inert Git tree and produces five
library assets for a version such as `v0.1.0`:

- `spice_0.1.0_source.tar.gz` containing the complete committed source tree
  below one versioned archive root;
- `spice_0.1.0_sbom.spdx.json` containing SPDX 2.3 source and module
  provenance;
- `checksums.txt` containing canonical SHA-256 entries for the source archive
  and SBOM;
- `checksums.txt.sig`, a raw Ed25519 signature over the exact checksum file;
  and
- `checksums.txt.pem`, the matching public key for transport convenience.

No binary is built from core and no second dependency resolver is introduced.
Planning, rendering, signing, and independent verification use immutable
trusted development/toolchain revisions. The uncredentialed validation phase
may populate the public Go module cache only to run this repository's exact
gate. Release planning, signing, and artifact verification run with Go module
and checksum network access disabled.

Consumers must authenticate the signature against the reviewed key committed
at [`security/release/ed25519-public.pem`](../security/release/ed25519-public.pem),
not against an unauthenticated key downloaded beside the assets. The repository
quality gate parses the key as a single Ed25519 SubjectPublicKeyInfo PEM and
pins its SHA-256 DER fingerprint:
`a7d12fc21024a11f0472887a37c731697a0aa2c2f6b84ff3afef6d47563422f1`.
The central workflow also refuses a private key that does not match this tagged
public anchor.

## Protected authority

The private key exists only as the repository Actions secret
`SPICE_LIBRARY_RELEASE_SIGNING_KEY`; it is distinct from every other Spice
repository's key. It is not stored in a GitHub environment, source file,
artifact, log, runner cache, or local workspace.

Two protected environments separate authority:

1. `release-signing` exposes the private key only after candidate validation
   and release-plan review.
2. `release-publish` permits publication only after independent artifact
   authentication has succeeded.

Both environments accept only `v*` deployment refs and require the sole
current repository owner as reviewer. Because the organization currently has
one human operator, self-review is enabled and documented rather than replaced
with a fictional second approver. Add an independent required reviewer and
disable self-review before delegating release authority to another maintainer.

Repository tag rules split creation from immutability. Only the named release
owner may bypass the active creation restriction for `refs/tags/v*`. A second
active ruleset prohibits updates and deletion of those tags with no bypass
actor. A mistaken release tag therefore remains auditable and must be followed
by a new version; it is never moved or deleted.

## Release ceremony

1. Confirm the public-key fingerprint, repository secret, protected
   environments, deployment policies, and both tag rulesets.
2. Run `make verify-release` on the exact clean candidate and require hosted CI
   to pass.
3. Create and push an annotated canonical SemVer tag whose target is the
   accepted main commit.
4. Review and approve `release-signing` only after uncredentialed validation
   and the inert plan succeed.
5. Review and approve `release-publish` only after the independent verifier
   authenticates the signed artifact set against the committed key.
6. Download the published assets into a clean directory and independently
   verify the signature, checksums, archive root/tree/commit, SPDX provenance,
   and unchanged remote tag target.

The release workflow being configured does not itself claim that a signed
preview exists. Until an immutable tag completes this ceremony, Spice remains
pre-alpha and has no compatibility-bearing release.
