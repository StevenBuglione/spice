# Security Policy

Spice is pre-alpha and has no stable release line yet.

Please avoid publicly disclosing exploitable vulnerabilities. Contact the repository owner privately through GitHub where possible and include reproduction steps, affected versions, and impact.

Security-sensitive framework areas include:

- Annotation and generated-code injection.
- Configuration secret exposure.
- HTTP request binding and validation.
- Authentication and authorization defaults.
- Module boundary bypasses.
- Dependency and starter supply-chain behavior.

No security feature is considered complete without negative tests and documented secure defaults.
