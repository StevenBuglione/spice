# Application Bootstrap

Spice's preferred application declaration is the ordinary Go process
entrypoint. It contains annotations, command arguments, and exit conversion—no
framework assembly:

```go
package main

import (
	"os"

	_ "example.com/shop/orders"
	_ "example.com/shop/payments"
	_ "example.com/shop/platform"
	spiceapp "example.com/shop/internal/spicegen/shop"
)

// @import { Application } from "github.com/StevenBuglione/spice/annotation/core"
// @import { Enable } from "github.com/StevenBuglione/spice/annotation/management"
// @import { Logging } from "github.com/StevenBuglione/spice/annotation/observability"

// @Application
// @Enable(expose=["health", "liveness", "readiness", "info", "metrics"], access="loopback")
// @Logging
func main() {
	os.Exit(spiceapp.Main(os.Args[1:]))
}
```

The generated target package is an explicit ordinary Go dependency. Its
`Main` returns a stable exit code and never calls `os.Exit`. This makes the
process boundary, generated implementation, Go-to-definition behavior, and
debugger transition visible without writing generated declarations beside
handwritten source.

## Compile-time discovery

For generation, Spice loads one selected command package through the standard
package driver and its existing typed compiler pipeline. Direct blank imports
of packages in the same Go module explicitly compose the application. Those
already type-checked dependencies are promoted into the same immutable program;
Spice does not perform a second package load. Named imports are ordinary code
dependencies, external blank imports retain ordinary Go side-effect semantics,
and `*/autoconfigure` imports use the separate explicit library-default
contract. Within the composed scope Spice discovers:

- package-documentation `@Module` roots and ownership;
- `@Bean` providers and their exact-type dependencies;
- typed configuration declarations;
- controllers, routes, authorization, and transaction boundaries;
- lifecycle hooks, jobs, asynchronous methods, caches, and events;
- explicitly imported library auto-configuration defaults and application
  features.

Every generated import and call is ordinary inspectable Go. Discovery does not
use reflection, runtime package scanning, `init`, a service locator, a global
registry, provider execution, or dependency presence.

A normal single-application module can run:

```text
spice generate
spice generate --check
spice build
```

In a monorepo, pass only the command package and select the command
unambiguously:

```text
cd examples/commerce
spice generate --target Commerce .
```

`--target` accepts the derived target name, command import path, or stable
marker symbol ID. Package patterns are analysis scope, not module imports and
not runtime activation.

## Run

`spice run` is the first-class development execution path:

```text
cd examples/commerce
spice run --target Commerce . -- -check
```

Arguments before `--` select the application and compile-time package scope;
arguments after it belong to the generated application command. Spice applies
guarded generation, builds only the selected package-main import path with
`-trimpath` into a unique temporary artifact, and starts that exact candidate.
Application standard input, output, error output, and nonzero exit codes are
preserved.

The child runs in an isolated process group. Interrupt and termination are
relayed on Windows and Unix so the generated command can drain HTTP and execute
its bounded lifecycle shutdown. A second interrupt or an unresponsive process
after the relay deadline is terminated. The temporary artifact is removed
after exit. Legacy parameter-root markers remain generatable and buildable but
are deliberately not runnable because they do not identify a package-main
process.

## Generated layout and ownership

The preferred target owns:

```text
internal/spicegen/<target>/spice_contracts_gen.go
internal/spicegen/<target>/spice_configuration_gen.go
internal/spicegen/<target>/spice_providers_gen.go
internal/spicegen/<target>/spice_assembly_gen.go
internal/spicegen/<target>/spice_features_gen.go       # when needed
internal/spicegen/<target>/spice_http_gen.go           # when HTTP is enabled
internal/spicegen/<target>/spice_http_route_<symbol>_<id>_gen.go
internal/spicegen/<target>/spice_lifecycle_gen.go
internal/spicegen/<target>/spice_command_gen.go
internal/spicegen/<target>/sources/<source-directory>/<source>_spice_gen.go
internal/spicegen/<target>/artifacts/openapi.json   # when controllers exist
.spice/<target>.manifest.json
```

There is deliberately no catch-all generated file. Contracts, configuration,
provider-graph construction, phase assembly, optional features, HTTP
coordination, lifecycle methods, and process commands each have one named
boundary. Every HTTP route has a readable, stable symbol-and-hash-derived file,
so a breakpoint or route edit does not require navigating an application-sized
renderer output.
The small assembly unit invokes these bounded phases in validated order. The
repository quality gate caps target-level generated units in the reference
applications at 400 lines so future features must add a semantic shard instead
of rebuilding the monolith.

The handwritten command imports the target package directly. Every contributing
handwritten file—including the application marker—owns one mirrored source
unit; providers, configuration binders, application metadata, and
conventional blank-identifier `@Implements` assertions derived from that file
live together there. Source
units use nested generated packages rather than appearing beside handwritten
Go, and target provider wiring calls their typed exported adapters.

The schema-5 manifest records each file's concern role—including a distinct
`target-http-route` role—primary source, related source declarations, exact
generated ranges, and SHA-256 ownership. Every regular file below a generated
target must appear in that manifest; the repository gates reject handwritten
tests, helper files, stale targets, and other unowned artifacts. Application
acceptance tests live outside `internal/spicegen` and import the generated
package as an ordinary black-box dependency. Generation
preserves unchanged files, refuses manual edits and unowned collisions, and
supports read-only check and bounded diff modes. Migration removes legacy
schema-4 monoliths and adjacent schema-3 shards only when their recorded hash
still matches.
Generated files have standard Go source positions and direct calls into
handwritten functions, so stepping from wiring into user code uses the normal
Go debugger. `spice generated --source path.go --line n` and the reverse
`--generated` form query the manifest without compiling or changing files.
Generated dependency variables, source adapter imports, and route functions
use stable semantic names. Short deterministic suffixes disambiguate exported
cross-file helpers; opaque ordinal names such as `provider17` are not the
ordinary wiring contract.

Generated source is excluded only from regeneration analysis with the reserved
`spice_generate` build tag. During analysis, Spice verifies that annotated
`func main` imports the exact generated target package and calls its `Main`
function. If that package is not available under the analysis tag, the loader
adds a pure in-memory stub package at that exact import path. It does not write
a bridge, suppress an undefined identifier, or accept any other load error.
This permits safe first generation while ordinary Go commands remain strict.

## Process and reusable ownership

The generated target package's `Main` owns conventional `SPICE_` environment loading and
`SIGINT`/`SIGTERM` because it is the process boundary. It creates a fresh
bounded shutdown context and returns zero for success, one for runtime failure,
or two for invalid command usage.

The generated `NewApplication`, `NewApplicationWithOptions`, `Start`, `Stop`,
`Run`, `Components`, `RunCommand`, and `Main` seams are exported directly by
the generated target package for tests and embedded policies. `Components` is
a generated typed snapshot of singleton beans, not a reflection container or
string lookup. `BeanOverrides` is a generated compile-time-typed test and
embedding seam for public singleton beans. `bean.Replace` supplies an exact
value; `bean.ReplaceFactory` supplies a value plus lifecycle cleanup. Generated
construction uses the replacement at the original provider position, so
dependencies, rollback, module cleanup ownership, and shutdown ordering remain
unchanged. Disabled zero values preserve production behavior; there is no
mutable application context or string-addressed bean replacement.
Generated `BeanOverrideLayer` and `ComposeBeanOverrides` additionally compose
named library, embedding, and test layers before construction. Layers are
validated in order and later enabled exact-type fields deliberately replace
earlier fields; the resulting `BeanOverrides` value still enters the same
construction path.

The reusable APIs accept caller-owned contexts, configuration sources,
overrides, observers, middleware, writers, loggers, and shutdown policy. They
never capture process signals.

## Legacy marker compatibility

During the pre-1.0 period, a package-level marker may still enumerate exact
provider roots as parameters:

```go
// @Application
func Commerce(*platform.Server, *orders.Service) {}
```

Legacy parameter-root markers also retain `internal/spicegen/<target>`. During
ownership migration, guarded generation removes old same-package bridges only
when their manifest hash still matches. Manual edits fail closed.
