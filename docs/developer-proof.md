# Petclinic developer proof

This walkthrough proves the complete Spice development loop without changing
the physical Go language. The source always retains `// @...`; GoLand folds
only the comment prefix for presentation.

## Prerequisites

- Go 1.26.5;
- the packaged Spice plugin installed in the pinned GoLand 2026.2 build;
- [`spice-framework/petclinic`](https://github.com/spice-framework/petclinic)
  cloned and opened at its module root;
- no external database.

The default Petclinic target uses instance-owned in-memory repositories and
performs no network I/O outside its loopback HTTP listener.

## Edit and restart loop

Start the real multi-package Petclinic target through the application module's
authorized Spice tool dependency:

```powershell
$env:SPICE_PETCLINIC_ADDRESS = "127.0.0.1:8080"
go tool github.com/spice-framework/spice/cmd/spice dev --target Petclinic .
```

```sh
SPICE_PETCLINIC_ADDRESS=127.0.0.1:8080 \
  go tool github.com/spice-framework/spice/cmd/spice dev --target Petclinic .
```

In `main.go`, add invalid `// @Unknown` immediately after `// @Application`.

1. GoLand immediately shows the shared source-positioned Spice diagnostic.
2. The physical document remains valid Go and still contains `// `.
3. `spice dev` rejects the candidate build and keeps the last-known-good
   process rather than replacing it.
4. Restore `// @Application` and save.
5. The diagnostic clears, guarded generation succeeds, and `spice dev`
   gracefully replaces the process.

Ctrl/Cmd-hover underlines the import path, imported symbol, annotation, typed
`@Implements` interface, constructor, and handler references. Ctrl/Cmd-click
opens their real Go declarations. Quick Documentation shows descriptor GoDoc,
arguments, module/version/replacement provenance, authorized tool, protocol,
and implementation link. The installed-plugin acceptance suite verifies those
interactions, zero-width concealment, light/dark colors, physical-source
preservation, and health presentation. It also invokes the real application
gutter action, captures a complete-package `spice run`, starts native Go/Delve
Debug, and stops at a breakpoint in the physical `main.go`. The suite rejects
temporary `gocommand-*` execution, naked annotations, and omitted generated
entrypoints.

## Exercise the vertical application

With the restarted process listening on `127.0.0.1:8080`, open `/owners/find`,
create or edit an owner, add a pet and visit, then inspect `/vets` and the
generated `/actuator/*` endpoints. The in-memory workflow proves generated
configuration, interface DI, validation, HTML/JSON routing, localization,
lifecycle, management, and persistence boundaries. PostgreSQL and MySQL
profiles select different repository implementations at compile time and are
covered by real-database workflow tests.

The independently versioned
[`spice-framework/commerce`](https://github.com/spice-framework/commerce)
integration remains the polished mail proof. Its receipt
response reports a stable message ID, `transport: "test"`,
`accepted: true`, and the attachment filename. The decoded test-transport
acceptance test additionally verifies the exact envelope, subject, text body,
and attachment bytes. The generated application package in the independent
Commerce module at `internal/spicegen/commerce` visibly
separates contracts, configuration,
providers, bounded assembly, features, HTTP coordination, one stable file per
route, lifecycle, and command behavior. Mirrored files under
`internal/spicegen/commerce/sources/<package>` contain
source-owned direct
constructors, configuration binders, and explicit interface assignments.
The schema-5 manifest provides exact source/generated locations and concern
roles, and the
`spice generated` command exposes those locations to humans and IDE clients.
There is no adjacent bridge, reflection, or runtime container.

Run the focused executable proofs directly:

```text
go test -run TestPetclinicDevelopmentWorkflowKeepsLastKnownGoodAndRestarts ./acceptance/devloop
cd <commerce-checkout>
go test -run TestCommerceDeveloperProof .
go test -run TestNotifierDeliversInspectableTestReceipt ./notifications
```

## Automated acceptance map

| Workflow evidence | Repository gate |
| --- | --- |
| Invalid overlay diagnostic, versioned clear, stale rejection | `internal/lsp` server tests |
| Real Petclinic invalid edit, last-known-good retention, generated restart | `TestPetclinicDevelopmentWorkflowKeepsLastKnownGoodAndRestarts` |
| Debounce, cancellation, timeout, and process replacement boundaries | `internal/devloop` engine tests |
| Physical `// ` preservation, concealment width, themes, hover/click, docs | packaged GoLand Starter/Driver suite |
| Complete-package Run/generate/build command construction | GoLand run-configuration integration tests |
| Actual installed gutter Run and native Go/Delve breakpoint | Packaged installed-GoLand Petclinic acceptance suite |
| Generated authorization, transaction, persistence, test mail, management | `TestCommerceDeveloperProof` |
| Exact decoded MIME and attachment | notifications tests |
| PostgreSQL close/reopen durability | tagged storage integration test |
| Offline third-party annotation SDK/tool | executable fixture smoke |

Petclinic's `make verify` runs its black-box development workflow in normal,
race, and vendor-offline modes alongside its 85% business coverage floor,
generated freshness, and all three executable targets. Core's `make verify`
independently runs the framework compiler/runtime gates under Go 1.26.5. The
independently versioned
[`spice-framework/goland`](https://github.com/spice-framework/goland)
repository separately requires packaged-plugin verification and installed-IDE
interaction tests on the exact compatible core and Petclinic commits.
