# GoLand integration

GoLand is Spice's primary editor integration. The repository-owned plugin in
`editors/goland` combines native IntelliJ Platform presentation with the
editor-neutral `spice lsp` compiler service:

- exact `// ` prefixes on canonical annotation comments are permanently
  folded to an empty placeholder, so `// @Application` is displayed as
  `@Application` with no blank columns;
- annotation names, argument names, strings, numbers, keyword values, and
  punctuation have separate native color keys under
  **Settings | Editor | Color Scheme | Spice**;
- highlighted PSI references provide modifier-hover underlining,
  `Ctrl`/`Cmd`-click, and **Go to Declaration or Usages** navigation;
- annotation lines carry a native gutter marker whose tooltip names the exact
  descriptor package and symbol and whose navigation target is that real Go
  declaration;
- annotated `main` packages receive a preferred whole-package Run/Debug
  configuration, preventing GoLand's temporary single-file runner from
  dropping generated bootstrap files or interpreting folded presentation as
  source;
- named, aliased, and namespace-qualified references resolve to the real
  one-file Go SDK descriptor selected by the file's explicit
  `@spice.import`;
- the public GoLand LSP API launches `spice lsp` for versioned diagnostics,
  import and annotation completion, rich descriptor hover, parameter
  information, safe code actions, module/configuration metadata, definition
  links, and handler implementation links.

The source file remains ordinary Go. Copying, saving, `gofmt`, Git, the Go
compiler, and generated code all retain the physical `// ` characters.
Concealment and color are editor presentation only. Ordinary Go language
features remain owned by GoLand's Go plugin.

## Compatibility

The current compatibility target is GoLand 2026.2.0.1, platform build
`262.8665.336`, on Java 25. IntelliJ IDEA Ultimate 2026.2 with the bundled or
installed `org.jetbrains.plugins.go` plugin uses the same integration. The
plugin declares build `262` as its minimum and is checked against the exact
target by JetBrains Plugin Verifier 1.409.

Windows and Linux are first-class. macOS uses the same Gradle and runtime
contracts. `SPICE_GOLAND_HOME` may identify a local IDE for repository
verification; otherwise the pinned build downloads the exact GoLand platform.
The Windows verifier also recognizes the standard
`%LOCALAPPDATA%\Programs\GoLand` installation.

## Install a development build

Install the Spice command first:

```text
go install ./cmd/spice
spice version
```

Build and verify the plugin from the repository root:

```text
make goland
```

The installable archive is
`editors/goland/build/distributions/spice-goland-0.1.0.zip`. In GoLand, open
**Settings | Plugins**, choose **Install Plugin from Disk**, select that
archive, and restart if the IDE requests it.

The plugin resolves the language server executable in this order:

1. the `spice.executable` JVM property;
2. the `SPICE_EXECUTABLE` environment variable;
3. `spice` on the environment inherited by GoLand.

For a desktop-launched IDE that does not inherit the terminal `PATH`, set
`SPICE_EXECUTABLE` to the absolute `spice` or `spice.exe` path before starting
GoLand. No shell, network lookup, download, or hidden global client is used at
runtime.

## Visual acceptance

`make goland` does more than instantiate folding APIs. Its real GoLand fixture:

1. opens a Go PSI file through the registered plugin extensions;
2. runs native highlighting and asserts the exact text-attribute key and range
   for sigils, namespaces, annotation names, parameters, literals, and
   punctuation;
3. runs GoLand's registered folding pipeline and asserts that every exact
   annotation and annotation-import prefix region is empty-placeholder,
   collapsed, and non-expandable;
4. proves the editor document, committed PSI, saved virtual file, and copied
   selection all retain the physical `// ` prefixes after concealment;
5. proves `@Application` produces package/directory execution with no
   single-file paths and retains the physical annotation comments;
6. renders realized editor components under the light and Darcula schemes;
7. compares normalized 8-by-8 color blocks with committed fixed-theme goldens,
   enforcing bounded mean and changed-region tolerances while retaining exact
   token/range assertions;
8. rejects blank or degenerate images;
9. packages the plugin, validates its structure/configuration, and runs the
   JetBrains binary/API verifier.

The generated visual reports are:

```text
editors/goland/build/reports/visual/spice-annotations-light.png
editors/goland/build/reports/visual/spice-annotations-dark.png
```

The reviewed baselines live in
`editors/goland/src/test/resources/goldens`. Block normalization and explicit
tolerances absorb bounded platform-font and antialiasing drift without
silently accepting a missing fold, theme-wide color regression, blank render,
or major layout shift. The build reports remain the exact current render for
human inspection on Windows and Linux.

The folding parser is intentionally narrow: it recognizes declaration comments
whose complete text begins with canonical `// @`, including explicit
`@spice.import` declarations. It never conceals ordinary comments or a
coincidental `@` later in prose. The Spice compiler/LSP remains the semantic
authority for targets, arguments, module rules, descriptor metadata, and
diagnostics; the plugin's small lexer is presentation-only.

## Descriptor navigation and documentation

An annotation reference is never resolved from a hidden built-in table.
`@spice.import` establishes the local symbol table:

```go
// @spice.import { Controller, Get as GET } from "github.com/StevenBuglione/spice/annotation/web"
// @spice.import * as security from "github.com/StevenBuglione/spice/annotation/security"

// @Controller
// @GET(path="/orders/{id}")
// @security.Authorize(anyRoles=["admin"])
```

GoLand's native PSI reference resolves `Controller`, `GET`, and
`security.Authorize` to their exact descriptor functions, so modifier-hover,
`Ctrl`/`Cmd`-click, and `Ctrl`/`Cmd+B` open ordinary indexed Go source.
`spice lsp` supplies the same real descriptor location to every editor and
provides **Go to Implementation** for the descriptor's declared handler source
symbol.

Quick Documentation and hover combine the descriptor GoDoc with its typed
arguments, defaults and allowed values, targets, examples, compatibility
range, resolved module version or local replacement, authorized tool path,
handler identity, protocol, and implementation symbol. Signature help uses the
same argument metadata. Selecting an annotation completion either uses its
existing local alias or inserts the required explicit named/namespace import;
the inserted edit is visible source, never an implicit compiler binding.

The independent
[`testdata/annotationfixture`](../testdata/annotationfixture) module is the
third-party acceptance source for descriptor and implementation navigation.
Its paired application uses both an aliased named import and a namespace
import, so the editor workflow does not prove only first-party paths.

If an editor buffer contains raw `@Application`-style lines, `spice lsp`
reports each line at its exact source range with a version-checked action that
inserts `// `. It does not surface the temporary `gocommand-*` loader dump or
silently analyze invalid Go.

## Developer commands

From `editors/goland`:

```text
gradlew.bat -PgolandPath=C:\path\to\GoLand test
gradlew.bat -PgolandPath=C:\path\to\GoLand buildPlugin
gradlew.bat -PgolandPath=C:\path\to\GoLand verifyPlugin
```

Use `./gradlew` on Linux and macOS. The wrapper pins Gradle 9.6.1 and both the
distribution and wrapper JAR checksums. `gradle.lockfile` fixes the complete
GoLand `262.8665.336`, test-framework, JUnit, and verifier graph.

## Boundaries

- Turning off code folding or disabling the Spice plugin reveals physical Go
  source; Spice never rewrites annotations into invalid raw `@` lines.
- Prefix folds are non-expandable by design so the presentation does not
  regress after editor refresh or file reopen.
- Definition navigation requires resolvable descriptor Go source. Missing
  module-cache or vendor source remains an actionable offline compiler
  diagnostic; the plugin does not download it.
- LSP start failures are reported by GoLand and do not disable native
  concealment, coloring, or PSI navigation to already indexed descriptors.
