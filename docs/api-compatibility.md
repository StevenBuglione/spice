# API compatibility and maturity

Spice is pre-alpha and has not published a compatibility-bearing release. This
document separates implementation presence from API support so a package does
not become an accidental public contract merely because its Go import path is
currently reachable.

The machine-readable source of truth is
[`api-compatibility.json`](api-compatibility.json). Repository verification
loads the actual `go list` package graph and fails when a package has no
classification, a classification is stale, or a maturity value is invalid.
The longest matching package prefix owns a package.

## Maturity levels

| Maturity | Contract |
| --- | --- |
| `preview-stable` | Intended to enter the first public preview compatibility surface. Breaking changes before 1.0 require release notes and migration guidance after that preview is tagged. |
| `experimental` | Usable for evaluation, examples, and feedback, but may change or move before preview. It must not be advertised as production-supported. |
| `internal` | Reachable only as a narrow compatibility/identity bridge and not a supported application import. Toolchain implementation is absent from this module. |

No package is stable before the first signed preview. `preview-stable` records
intent and review priority; it does not retroactively create a compatibility
promise for the current untagged repository.

## Current boundary

The intended preview surface is deliberately narrow:

- annotation syntax, descriptors, the public SDK/protocol, and SDK test tools;
- dependency handles and lifecycle;
- configuration, conversion, validation, resources, and bounded expressions;
- core web, management, security, data, mail, interception, and `spicetest`
  contracts.

Async, batch, cache, events, outbound HTTP, internationalization, messaging,
migrations, observability adapters, retry, scheduling, sessions, views, and
every external-service starter remain experimental. Their implementation and
tests are useful evidence, but they do not receive a support claim until their
own repository and release matrix satisfy ADR 0012.

Compiler packages, executable entrypoints, generated application targets, and
reference applications live in separate modules and are not core APIs. The
only repository internal package is the local quality gate. Go module and
`internal` boundaries enforce this separation.

## Change policy

1. Adding a Go package requires adding an exact or inherited classification in
   the same commit.
2. Promoting an experimental package requires public documentation, positive
   and negative tests, a runnable external example, compatibility tests, and a
   recorded API review.
3. Moving a preview-stable contract requires a deprecation or migration path
   appropriate to the current major version.
4. Internal packages may change freely but cannot be required by a released
   application or third-party annotation tool.
5. Capability-map `available` and `integration` labels describe implementation
   disposition only. They do not override this API maturity policy.
