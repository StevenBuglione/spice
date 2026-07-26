# Application Bootstrap

Spice's preferred application declaration is the ordinary Go process
entrypoint. It contains annotations, command arguments, and exit conversion—no
framework assembly:

```go
package main

import "os"

// @Application
// @management.Enable(expose=["health", "liveness", "readiness", "info", "metrics"])
// @observability.Logging
func main() {
	os.Exit(spiceMain(os.Args[1:]))
}
```

`spiceMain` is generated into the same package. It returns a stable exit code
and never calls `os.Exit`.

## Compile-time discovery

For generation, Spice loads one selected local Go package scope through the
standard package driver and its existing typed compiler pipeline. Within that
scope it discovers:

- package-documentation `@Module` roots and ownership;
- `@Bean` providers and their exact-type dependencies;
- typed configuration declarations;
- controllers, routes, authorization, and transaction boundaries;
- lifecycle hooks, jobs, asynchronous methods, caches, and events;
- explicitly selected starter entrypoints and application features.

Every generated import and call is ordinary inspectable Go. Discovery does not
use reflection, runtime package scanning, `init`, a service locator, a global
registry, provider execution, or dependency presence.

A normal single-application module can run:

```text
spice generate
spice generate --check
spice build
```

In a monorepo, bound the compile-time package scope and select the command
unambiguously:

```text
spice generate --target Commerce ./examples/commerce/...
```

`--target` accepts the derived target name, command import path, or stable
marker symbol ID. Package patterns are analysis scope, not module imports and
not runtime activation.

## Run

`spice run` is the first-class development execution path:

```text
spice run --target Commerce ./examples/commerce/... -- -check
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
<command-directory>/zz_spice_gen.go
<command-directory>/openapi.json   # when controllers exist
.spice/<target>.manifest.json
```

The manifest records `application-package` layout and exact SHA-256 ownership.
Generation preserves unchanged files, refuses manual edits and unowned
collisions, and supports read-only check and bounded diff modes.

Generated source is excluded only from regeneration analysis with the reserved
`spice_generate` build tag. When the generated bridge is missing or stale, the
loader permits exactly the unresolved `spiceMain` call at its source position
inside the annotated `func main`; every other load error remains fatal. This
allows safe first generation and recovery without weakening ordinary builds.

## Process and reusable ownership

`spiceMain` owns conventional `SPICE_` environment loading and
`SIGINT`/`SIGTERM` because it is the process boundary. It creates a fresh
bounded shutdown context and returns zero for success, one for runtime failure,
or two for invalid command usage.

The generated `NewApplication`, `NewApplicationWithOptions`, `Start`, `Stop`,
`Run`, and `RunCommand` seams remain available to same-package tests and
embedded policies. They accept caller-owned contexts, sources, observers,
middleware, writers, loggers, and shutdown policy. Reusable application APIs
never capture process signals.

## Legacy marker compatibility

During the pre-1.0 period, a package-level marker may still enumerate exact
provider roots as parameters:

```go
// @Application
func Commerce(*platform.Server, *orders.Service) {}
```

Legacy markers retain `internal/spicegen/<target>`. When migrating the same
target to annotated `main.go`, guarded generation can move schema-1 legacy
ownership only if every old generated file is unchanged. Manual edits fail
closed.
