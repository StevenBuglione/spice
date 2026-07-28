# Commerce developer proof

This walkthrough proves the complete Spice development loop without changing
the physical Go language. The source always retains `// @...`; GoLand folds
only the comment prefix for presentation.

## Prerequisites

- Go 1.26.5;
- the packaged Spice plugin installed in the pinned GoLand 2026.2 build;
- the repository opened at its module root;
- no external database or mail server.

The default `memory://commerce` database and `test` mail transport are
instance-owned, bounded, and perform no network I/O outside the loopback HTTP
slice.

## Edit and restart loop

Start the reference application from the repository root. Use a local token
with at least 16 bytes:

```powershell
$env:SPICE_COMMERCE_DEVELOPER_TOKEN = "commerce-local-token"
go run ./cmd/spice dev --target Commerce ./examples/commerce/...
```

```sh
SPICE_COMMERCE_DEVELOPER_TOKEN=commerce-local-token \
  go run ./cmd/spice dev --target Commerce ./examples/commerce/...
```

In `examples/commerce/main.go`, temporarily change `// @Application` to the
invalid `// @Application(unknown=true)`.

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

With the restarted process listening on `127.0.0.1:8081`:

```text
curl -H "Authorization: Bearer commerce-local-token" \
  -H "Content-Type: application/json" \
  -d "{\"quantity\":2}" \
  http://127.0.0.1:8081/orders

curl -H "Authorization: Bearer commerce-local-token" \
  http://127.0.0.1:8081/orders/order-000001

curl -X POST \
  -H "Authorization: Bearer commerce-local-token" \
  http://127.0.0.1:8081/orders/order-000001/receipt
```

The receipt response reports a stable message ID, `transport: "test"`,
`accepted: true`, and the attachment filename. The decoded test-transport
acceptance test additionally verifies the exact envelope, subject, text body,
and attachment bytes. The generated application code in
`internal/spicegen/commerce/zz_spice_gen.go` visibly contains direct
constructors, explicit interface assignments, authorization policies,
transaction ownership, repository calls, and route adapters. Narrow
`*_commerce_spice_gen.go` files beside implementations contain compiler-owned
interface assertions, and the command bridge contains no wiring. There is no
reflection or runtime container.

Run the single-process executable proof directly:

```text
go test -run TestCommerceDeveloperProof ./examples/commerce
go test -run TestNotifierDeliversInspectableTestReceipt ./examples/commerce/notifications
```

## Automated acceptance map

| Workflow evidence | Repository gate |
| --- | --- |
| Invalid overlay diagnostic, versioned clear, stale rejection | `internal/lsp` server tests |
| Debounce, failed-candidate retention, graceful replacement | `internal/devloop` engine tests |
| Physical `// ` preservation, concealment width, themes, hover/click, docs | packaged GoLand Starter/Driver suite |
| Complete-package Run/Debug with generated files | GoLand run-configuration integration tests |
| Generated authorization, transaction, persistence, test mail, management | `TestCommerceDeveloperProof` |
| Exact decoded MIME and attachment | notifications tests |
| PostgreSQL close/reopen durability | tagged storage integration test |
| Offline third-party annotation SDK/tool | executable fixture smoke |

`make verify` runs the complete mandatory set under Go 1.26.5, including
formatting, vet, lint/NilAway, gosec, govulncheck, shuffled/race tests, fuzz
smoke, the 85% repository coverage floor, vendor-offline tests, packaged plugin
verification, installed-GoLand interaction tests, generated freshness, and
executable smoke paths.
