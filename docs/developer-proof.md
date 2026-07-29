# Petclinic developer proof

This walkthrough proves the complete Spice development loop without changing
the physical Go language. The source always retains `// @...`; GoLand folds
only the comment prefix for presentation.

## Prerequisites

- Go 1.26.5;
- the packaged Spice plugin installed in the pinned GoLand 2026.2 build;
- the repository opened at its module root;
- no external database.

The default Petclinic target uses instance-owned in-memory repositories and
performs no network I/O outside its loopback HTTP listener.

## Edit and restart loop

Build the current CLI, then start the real multi-package Petclinic target:

```powershell
go build -trimpath -o ./bin/spice.exe ./cmd/spice
Set-Location examples/petclinic
$env:SPICE_PETCLINIC_ADDRESS = "127.0.0.1:8080"
../../bin/spice.exe dev --target Petclinic . ./memory ./model ./owner ./presentation ./system ./vet
```

```sh
go build -trimpath -o ./bin/spice ./cmd/spice
cd examples/petclinic
SPICE_PETCLINIC_ADDRESS=127.0.0.1:8080 \
  ../../bin/spice dev --target Petclinic . ./memory ./model ./owner ./presentation ./system ./vet
```

In `examples/petclinic/main.go`, add invalid `// @Unknown` immediately after
`// @Application`.

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
preservation, and complete-package Run/Debug.

## Exercise the vertical application

With the restarted process listening on `127.0.0.1:8080`, open `/owners/find`,
create or edit an owner, add a pet and visit, then inspect `/vets` and the
generated `/actuator/*` endpoints. The in-memory workflow proves generated
configuration, interface DI, validation, HTML/JSON routing, localization,
lifecycle, management, and persistence boundaries. PostgreSQL and MySQL
profiles select different repository implementations at compile time and are
covered by real-database workflow tests.

The separate commerce integration remains the polished mail proof. Its receipt
response reports a stable message ID, `transport: "test"`,
`accepted: true`, and the attachment filename. The decoded test-transport
acceptance test additionally verifies the exact envelope, subject, text body,
and attachment bytes. The generated application package in
`internal/spicegen/commerce` visibly separates contracts, configuration,
providers, bounded assembly, features, HTTP coordination, one stable file per
route, lifecycle, and command behavior. Mirrored files under
`internal/spicegen/commerce/sources/<package>` contain source-owned direct
constructors, configuration binders, and explicit interface assignments.
The schema-5 manifest provides exact source/generated locations and concern
roles, and the
`spice generated` command exposes those locations to humans and IDE clients.
There is no adjacent bridge, reflection, or runtime container.

Run the focused executable proofs directly:

```text
go test -run TestPetclinicDevelopmentWorkflowKeepsLastKnownGoodAndRestarts ./internal/cli
go test -run TestCommerceDeveloperProof ./examples/commerce
go test -run TestNotifierDeliversInspectableTestReceipt ./examples/commerce/notifications
```

## Automated acceptance map

| Workflow evidence | Repository gate |
| --- | --- |
| Invalid overlay diagnostic, versioned clear, stale rejection | `internal/lsp` server tests |
| Real Petclinic invalid edit, last-known-good retention, generated restart | `TestPetclinicDevelopmentWorkflowKeepsLastKnownGoodAndRestarts` |
| Debounce, cancellation, timeout, and process replacement boundaries | `internal/devloop` engine tests |
| Physical `// ` preservation, concealment width, themes, hover/click, docs | packaged GoLand Starter/Driver suite |
| Complete-package Run/Debug with generated files | GoLand run-configuration integration tests |
| Generated authorization, transaction, persistence, test mail, management | `TestCommerceDeveloperProof` |
| Exact decoded MIME and attachment | notifications tests |
| PostgreSQL close/reopen durability | tagged storage integration test |
| Offline third-party annotation SDK/tool | executable fixture smoke |

`make verify` runs the complete mandatory set under Go 1.26.5, including
formatting, vet, lint/NilAway, gosec, govulncheck, shuffled/race tests, fuzz
smoke, the 85% handwritten-product repository coverage floor (generated files
remain compile/execution tested), vendor-offline tests, packaged plugin
verification, installed-GoLand interaction tests, generated freshness, and
executable smoke paths.
