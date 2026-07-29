# Getting started

This guide builds and runs the in-memory Petclinic application using ordinary
Go source and inspectable generated Go. It requires Go 1.26.5 and GNU Make.
No database, container, or network download is needed after the repository
dependencies are present.

## Build the CLI

From the repository root:

```powershell
go build -trimpath -o .\bin\spice.exe .\cmd\spice
.\bin\spice.exe version
```

```sh
go build -trimpath -o ./bin/spice ./cmd/spice
./bin/spice version
```

The executable is a development build. Release archives are produced only
from clean exact tags by the process in [releasing.md](releasing.md).

## Inspect the application

Petclinic is an independent consumer module. Its `go.mod` selects Spice,
authorizes the annotation tool with a standard Go `tool` directive, and uses a
local `replace` only inside this source checkout.

The process entrypoint in `examples/petclinic/main.go` is valid Go:

```go
// @import { Application } from "github.com/StevenBuglione/spice/annotation/core"
// @import { Enable } from "github.com/StevenBuglione/spice/annotation/management"

// @Application
// @Enable(expose=["health", "readiness", "info"], access="loopback")
func main() {
	os.Exit(spiceapp.Main(os.Args[1:]))
}
```

`@import` is file-scoped and explicit. The imported descriptor is a real Go
function that the compiler decodes without executing. The target module's
`go.mod` is the only authority that permits its handler tool to run.
`spiceapp` is the ordinary Go import alias for
`github.com/StevenBuglione/spice-petclinic/internal/spicegen/petclinic`.

## Verify and generate

From `examples/petclinic`, use the package scope that belongs to the in-memory
application:

```text
../../bin/spice verify . ./memory ./model ./owner ./presentation ./system ./vet
../../bin/spice generate --check --target Petclinic . ./memory ./model ./owner ./presentation ./system ./vet
```

`generate --check` is read-only. To create or update owned artifacts, run the
same command without `--check`. The output has three clear roles:

```text
internal/spicegen/petclinic/spice_assembly_gen.go
    bounded construction-phase sequencing
internal/spicegen/petclinic/spice_{contracts,configuration,providers}_gen.go
    typed contracts, configuration metadata, and dependency construction
internal/spicegen/petclinic/spice_http_gen.go
    HTTP and management coordination
internal/spicegen/petclinic/spice_http_route_<symbol>_<id>_gen.go
    one stable, source-linked route adapter
internal/spicegen/petclinic/spice_{features,lifecycle,command}_gen.go
    optional features, reusable lifecycle, and process entrypoint behavior
internal/spicegen/petclinic/sources/<package>/<source>_spice_gen.go
    one source-owned unit with direct construction, binding, and assertions
internal/spicegen/petclinic/artifacts/openapi.json
    generated non-Go contracts
.spice/petclinic.manifest.json
    hashes, roles, source origins, and generated ranges
```

Generated files are committed, formatted Go. The ownership manifest prevents
Spice from overwriting manual edits or unrelated files. Run
`spice generated --source main.go` to locate the source-owned application unit,
or pass that generated path with `--generated` for the reverse mapping.

## Run and debug

Start the application:

```powershell
$env:SPICE_PETCLINIC_ADDRESS = "127.0.0.1:8080"
..\..\bin\spice.exe run --target Petclinic . ./memory ./model ./owner ./presentation ./system ./vet
```

```sh
SPICE_PETCLINIC_ADDRESS=127.0.0.1:8080 \
  ../../bin/spice run --target Petclinic . ./memory ./model ./owner ./presentation ./system ./vet
```

Open `http://127.0.0.1:8080/`. Search owners, edit an owner, add a pet and a
visit, and inspect `/vets`. Management routes under `/actuator/` accept only a
direct loopback peer.

`spice run` generates safely and then builds the complete package with
`-trimpath`; it never compiles a temporary source fragment. A Go debugger sees
normal generated calls and steps directly into handwritten constructors,
controllers, and repositories. The GoLand plugin uses this complete-package
path for Run and generates before native Go/Delve Debug.

## Use the development loop

Replace `run` with `dev`. Spice watches relevant files, debounces changes, and
gracefully replaces the process only after analysis, generation, and build
succeed:

```text
../../bin/spice dev --target Petclinic . ./memory ./model ./owner ./presentation ./system ./vet
```

Add `// @Unknown` below `// @Application` and save. The compiler reports an
exact diagnostic while the last-known-good process remains live. Remove the
invalid annotation and save; Spice regenerates and restarts the package.

## Test a generated application

Use ordinary Go tests for domain code. For generated wiring, create a typed
context with `spicetest.NewContext`; for routes, use `spicetest.NewHTTP`; for
database behavior that must roll back, use `spicetest.NewSQL`. These helpers
accept typed generated applications and never use reflection, provider
replacement, or a runtime container.

Focused module tests retain ordinary Go controls:

```text
spice test --module example.com/shop/orders --race --count=1 ./...
```

See [testing.md](testing.md) for complete examples and cleanup behavior.

## Next steps

- [application.md](application.md) explains discovery and generated ownership.
- [annotations.md](annotations.md) lists the supported annotation contracts.
- [goland.md](goland.md) installs and verifies the primary editor experience.
- [spring-to-spice.md](spring-to-spice.md) maps familiar Spring concepts.
- [developer-proof.md](developer-proof.md) runs the decisive editor/dev proof.
