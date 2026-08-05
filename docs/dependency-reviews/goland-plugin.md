# GoLand plugin dependency review

## Scope

The GoLand adapter is isolated under `editors/goland`. None of its Java,
Gradle, test, or IntelliJ Platform dependencies enter Spice's Go compiler,
runtime, generated applications, vendor tree, or deployed services.

## Direct tooling and test dependencies

| Dependency | Pin | Scope | License and maintenance |
| --- | --- | --- | --- |
| IntelliJ Platform Gradle Plugin | 2.18.1 | Build/package | JetBrains-maintained, Apache-2.0 |
| GoLand platform | 2026.2.0.1 / `262.8665.336` | Compile/test/verifier | JetBrains product SDK; not redistributed by the Spice archive |
| Go plugin | bundled `262.8665.336` | Compile/runtime contract | JetBrains-bundled GoLand/IDEA dependency |
| Gradle wrapper | 9.6.1 | Build orchestration | Apache-2.0; distribution and wrapper JAR SHA-256 pinned |
| IntelliJ Plugin Verifier | 1.409 | Binary/API verification | JetBrains-maintained, Apache-2.0 |
| JUnit | 4.13.2 | Test only | Eclipse Public License 1.0 |

`gradle.lockfile` fixes the resolved transitive graph for the exact downloadable
GoLand distribution used by CI. `gradle-installed-goland.lockfile` separately
fixes the graph for installed-IDE verification because the IntelliJ Platform
plugin exposes that same build through a different `localIde` coordinate. Both
lock states remain strict. The produced plugin archive contains Spice
classes/resources and the repository Apache-2.0 license; it does not bundle
GoLand, the Go plugin, Gradle, the verifier, JUnit, or a second compiler
implementation.

## Security, cancellation, and data

The runtime adapter performs no network requests, credential reads, telemetry,
package scanning, or dependency downloads. It starts only the explicitly
configured or inherited `spice` executable with the single `lsp` argument and
UTF-8 stdio. `GeneralCommandLine` avoids shell interpretation. GoLand's LSP
client owns process shutdown and request cancellation; Spice's bounded server
owns protocol limits and compiler cancellation.

The bundled annotation reference is read through a fixed classpath resource
with a 2 MiB bound. The virtual declaration page is read-only and project
scoped. Native syntax coloring and folding examine only existing `PsiComment`
text and maintain no global mutable application model.

## Compatibility and observability

The repository runs Java compiler lint with warnings as errors, real GoLand
fixture tests, light/Darcula visual rendering, archive/configuration validation,
and Plugin Verifier against `GO-262.8665.336`. Runtime LSP launch failures use
GoLand's normal language-server reporting. The adapter adds no independent
logging or telemetry channel and never records annotation values, source, or
secrets.

Platform updates require an explicit target/lock change, repeat visual
inspection on Windows and Linux, and a clean verifier result before release.
