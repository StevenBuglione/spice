# Upgrading Spice

## Migrating to the Java-structured annotation model

The pre-0.2 annotation vocabulary aligns configuration ownership with Spring's
familiar concepts while retaining ordinary Go execution:

- replace typed `@Configuration(prefix="...")` declarations with
  `@ConfigurationProperties(prefix="...")`;
- introduce a constructible `@Configuration` type and move package-level
  `@Bean` providers onto its methods;
- annotate event payload types with `@event.Topic` instead of declaring marker
  functions;
- use `@Component` for generic managed collaborators rather than stretching
  `@Service` or a provider function.

Package-level beans and function topics remain accepted outside the
`java-structured` profile during the migration window. Generated application
behavior remains direct Go calls with no reflection or runtime scanning.

Spice is pre-1.0. Public contracts may change between minor releases while the
v1 compatibility policy is being established. Upgrades must nevertheless be
reviewable: module selection, source edits, generated changes, and runtime
behavior are separate explicit steps.

## Before upgrading

1. Start from a clean worktree and preserve the current generated files.
2. Run the current `spice generate --check` and application tests.
3. Read the target release notes and this guide for hard cuts or migrations.
4. Confirm the required Go toolchain. The current line requires Go 1.26.5.

Do not delete `.spice/*.manifest.json` to force an upgrade. The manifests are
the proof of generated ownership and let Spice distinguish a stale file from a
manual edit.

If the application itself moves to a different Go module path, preserve that
generated tree and manifest, update the handwritten module and import paths,
then run one guarded relocation pass before the normal check:

```text
go tool github.com/spice-framework/toolchain/cmd/spice generate --relocate-module-from example.com/previous/module ./...
go tool github.com/spice-framework/toolchain/cmd/spice generate --check ./...
```

Relocation is not a force option. Spice verifies the previous target identity
and every recorded generated-file hash, so a manual edit or unrelated manifest
still fails closed.

## Canonical organization namespace hard cut

The current pre-alpha line moved from the maintainer's personal module
namespace to `github.com/spice-framework/spice`. This is a Go type-identity
change, not an alias: imports, the runtime requirement, annotation descriptor
imports, the annotation `tool` directive, generated source, and vendor metadata
must all use the organization path together.

Before changing an application, prove its existing generated targets are
current. Then update the module graph and annotation imports, run the canonical
CLI to regenerate every target, and rebuild vendor metadata. Do not mechanically
edit generated files or their recorded hashes in an application. The Spice
repository itself used its independently buildable stage-zero compiler for the
one-time core cutover; normal consumers use a released organization CLI.

## Update the Go graph

The runtime, descriptors, SDK, and official annotation tool must resolve to the
same Spice module version. Use Go's module commands from the application
module:

```text
go get github.com/spice-framework/spice@<version>
go get -tool github.com/spice-framework/toolchain/cmd/spice-annotation-core@<version>
go mod tidy
```

Inspect `go.mod` and `go.sum`. If the application vendors dependencies, run
`go mod vendor` and review the vendor change. A local `replace` is appropriate
for framework development, but release and provenance output identifies it as
local source rather than a verified module version.

Spice never installs or upgrades annotation tools during ordinary generation,
verification, or editor analysis. GoLand/LSP installation assistance first
previews the exact temporary-modfile command and module-file diff, then requires
a separate confirmation against unchanged file hashes.

## Preview source and generation changes

Run these in order:

```text
go tool github.com/spice-framework/toolchain/cmd/spice verify ./...
go tool github.com/spice-framework/toolchain/cmd/spice generate --diff ./...
go tool github.com/spice-framework/toolchain/cmd/spice generate ./...
go tool github.com/spice-framework/toolchain/cmd/spice generate --check ./...
go test ./...
```

`--diff` is read-only and bounded. Generation refuses an owned file whose
current hash differs from its manifest, so a framework upgrade cannot silently
erase a manual edit. Review the full target-wide wiring package, mirrored
source units, OpenAPI document, and manifest together. New
package-main targets have no adjacent command bridge; the handwritten
entrypoint imports the generated target package directly. A legacy bridge is
removed only when its old manifest hash still matches.

Manifest schema 5 replaces the schema-4 `zz_spice_gen.go` target monolith with
named concern files and stable per-route units. The first schema-5 generation
removes the old file only when its schema-4 recorded hash still matches. If it
was manually edited, Spice stops with a conflict so that work can be recovered
or intentionally moved before regeneration. Do not delete the manifest or the
old file to bypass this check.

For a service with multiple application targets, pass the same `--target` and
package scope used by its build. Never generate a broad module and assume it
represents every target.

## Verify runtime behavior

Run the application's focused tests and a complete generated process:

```text
go tool github.com/spice-framework/toolchain/cmd/spice run ./... -- -check
go tool github.com/spice-framework/toolchain/cmd/spice test --module <module-import-path> --race --count=1 ./...
```

For production integrations, also exercise the exact database, identity
provider, cache, mail transport, and telemetry versions selected by the
application. Starter compatibility checks validate reviewed module identity;
they do not replace an environment integration test.

## Editor upgrades

The GoLand plugin and CLI/LSP should use the same Spice release. Install the
plugin archive built for the supported GoLand platform, open the Spice health
view, and confirm:

- Spice executable and Go versions;
- module root and vendor/offline mode;
- LSP running state;
- annotation-tool authorization;
- descriptor and implementation navigation.

The plugin only presents annotations. Disabling it must reveal the unchanged
physical `// @...` comments, and ordinary `go test`, package build, and Git
operations must continue to work.

## Current hard cuts

The current development line has deliberately retired:

- `@spice.import`; use explicit file-scoped `@import`;
- protocol `v1alpha1`; annotation tools must implement the current public
  protocol and typed contribution union;
- implicit built-in annotation lookup;
- concrete-to-interface discovery by general assignability;
- single-file GoLand execution for Spice applications.

Diagnostics provide safe edits where a transformation is mechanical. Semantic
changes such as selecting a bean, changing transaction ownership, or exposing a
management endpoint require an explicit developer decision.

## Rollback

Revert the module, checksum, vendor, handwritten, generated, and manifest
changes as one reviewed change, then run the prior CLI's
`generate --check`. Do not combine generated files from two Spice versions.
Database migrations and externally visible messages may be irreversible; use
the application's own rollback/forward-repair process for those effects.

## Toward v1

Before v1.0, Spice will publish a frozen public package and annotation
compatibility policy. After v1, SemVer will govern public Go APIs, annotation
syntax and descriptor/protocol contracts. Generated file layout is an
inspectable implementation detail protected by manifests, so consumers should
call generated public seams rather than parse generated source.
