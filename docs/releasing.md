# Releasing Spice

Spice releases are built by the repository rather than a second build system.
The release command uses the exact Go module, vendor directory, toolchain, and
source tree already accepted by `make verify-release`.

## Artifact contract

For a version such as `v0.9.0`, `cmd/spice-release` creates:

- `spice_0.9.0_{darwin,linux}_{amd64,arm64}.tar.gz`;
- `spice_0.9.0_windows_{amd64,arm64}.zip`;
- `spice_0.9.0_sbom.spdx.json`, an SPDX 2.3 dependency document derived from
  the committed vendor graph;
- `checksums.txt`, covering every archive and the SBOM;
- `checksums.txt.sig`, a raw Ed25519 signature over the exact checksum file;
- `checksums.txt.pem`, the corresponding public key.

Every CLI is built with `CGO_ENABLED=0`, `-mod=vendor`, `-trimpath`, and
`-buildvcs=false`. The release version is injected into the otherwise ordinary
CLI string variable at link time. Archive paths, modes, ordering, gzip/ZIP
headers, and SPDX creation time use the source commit epoch. No absolute
workspace path, current time, or network lookup enters an artifact.

The command writes through a new staging directory and refuses an existing
output path. A normal release requires a clean checkout whose `HEAD` has the
exact requested tag and requires an Ed25519 private key. `-rehearsal` is the
only explicit escape hatch; it permits an unsigned, untagged local build.

## Signing key

Generate an offline Ed25519 PKCS#8 key:

```text
openssl genpkey -algorithm ED25519 -out spice-release-key.pem
```

Store the private key outside the repository. GitHub release automation reads
the private key from the protected `SPICE_RELEASE_SIGNING_KEY` secret. A
base64-encoded 32-byte Ed25519 seed or 64-byte private key is also accepted.
The private key is never copied into an archive, SBOM, log, or published
artifact.

Verify the signed checksum file with OpenSSL:

```text
openssl pkeyutl -verify -pubin -inkey checksums.txt.pem -rawin -in checksums.txt -sigfile checksums.txt.sig
```

Then verify artifact hashes with `sha256sum -c checksums.txt` on Linux/macOS,
or compare each first-column SHA-256 value with `Get-FileHash -Algorithm
SHA256` in PowerShell.

## Ceremony

1. Confirm the version remains pre-1.0 unless the compatibility gate is
   deliberately frozen.
2. Run `make verify-release` on the exact clean commit.
3. Create and push an annotated `vX.Y.Z` tag.
4. The tag workflow repeats release verification, derives
   `SOURCE_DATE_EPOCH` from the tag commit, builds the six CLI archives,
   signs their checksums, and publishes a GitHub release only after all steps
   succeed.
5. Download the published assets, verify the Ed25519 signature and SHA-256
   hashes, then execute `spice version` for the relevant platform.

The workflow uses GitHub only as the post-tag distribution mirror. Artifact
construction and verification remain reproducible local commands.

For a local rehearsal:

```text
go run ./cmd/spice-release -rehearsal -version v0.9.0-rc.1 -output dist-rehearsal
```
