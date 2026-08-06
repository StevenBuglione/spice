# Security Policy

Spice is pre-alpha and has no stable release line yet. The configured release
path does not imply that a signed preview has been published.

Please avoid publicly disclosing exploitable vulnerabilities. Contact the repository owner privately through GitHub where possible and include reproduction steps, affected versions, and impact.

Security-sensitive framework areas include:

- Annotation and generated-code injection.
- Configuration secret exposure.
- HTTP request binding and validation.
- Authentication and authorization defaults.
- Module boundary bypasses.
- Dependency and starter supply-chain behavior.

No security feature is considered complete without negative tests and documented secure defaults.

## Release authenticity

Core source releases are signed with a repository-distinct Ed25519 key. The
reviewed public trust anchor is committed at
`security/release/ed25519-public.pem`; its SHA-256 SubjectPublicKeyInfo DER
fingerprint is
`a7d12fc21024a11f0472887a37c731697a0aa2c2f6b84ff3afef6d47563422f1`
and is enforced by the repository quality gate. Authenticate downloaded
signatures against that committed key, not only against a public key shipped in
the same release.

The matching private key is available only to the protected signing phase as
the repository Actions secret `SPICE_LIBRARY_RELEASE_SIGNING_KEY`. Candidate
validation is uncredentialed, signing and publishing require separate
environment approvals, and `v*` tags cannot be updated or deleted. Report any
suspected key disclosure, tag-rule bypass, workflow-pin drift, artifact
substitution, or checksum/signature mismatch through the private channel above.
