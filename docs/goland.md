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
- built-in references open an in-memory, read-only annotation declaration page
  at the exact definition row rather than a generic website;
- the public GoLand LSP API launches `spice lsp` for versioned diagnostics,
  completion, hover, safe code actions, module/configuration metadata,
  semantic information, and third-party definition links.

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
   for each annotation token kind;
3. runs GoLand's registered folding pipeline and asserts that every exact
   prefix region is empty-placeholder, collapsed, and non-expandable;
4. proves the editor document, committed PSI, saved virtual file, and copied
   selection all retain the physical `// ` prefixes after concealment;
5. renders realized editor components under the light and Darcula schemes;
6. rejects blank or degenerate images;
7. packages the plugin, validates its structure/configuration, and runs the
   JetBrains binary/API verifier.

The generated visual reports are:

```text
editors/goland/build/reports/visual/spice-annotations-light.png
editors/goland/build/reports/visual/spice-annotations-dark.png
```

They are build evidence rather than committed screenshots, avoiding
platform-font and antialiasing drift while preserving a repeatable visual
inspection path on Windows and Linux.

The folding parser is intentionally narrow: it recognizes declaration comments
whose complete text begins with canonical `// @` and a valid qualified
annotation name. It never conceals ordinary comments or a coincidental `@`
later in prose. The Spice compiler/LSP remains the semantic authority for
targets, arguments, module rules, and diagnostics; the plugin's small lexer is
presentation-only.

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
- Built-in declaration pages are bundled from `docs/annotations.md`.
  Third-party definition ownership stays with `spice lsp`.
- LSP start failures are reported by GoLand and do not disable native
  concealment, coloring, or built-in navigation.
