# GoLand integration

GoLand is Spice's primary editor integration. The repository-owned plugin in
`editors/goland` combines native IntelliJ Platform presentation with the
editor-neutral `spice lsp` compiler service:

- typing `@` on an otherwise blank package declaration or parameter line
  inserts physical `// @` source in the same undoable editor command; typing
  at the start of an existing declaration or parameter inserts a dedicated
  annotation line above it instead of commenting out the Go declaration;
  folding and annotation completion refresh immediately, so source is valid
  Go from the first annotation character rather than only after completion;
- exact `// ` prefixes on canonical annotation comments are permanently
  folded to an empty placeholder, so `// @Application` is displayed as
  `@Application` with no blank columns;
- the concealed prefix, sigils, import symbols/aliases, namespaces, annotation
  names, argument names, Go type references, literals, keywords, punctuation,
  unresolved symbols, and deprecated symbols have separate theme-aware native
  color keys under
  **Settings | Editor | Color Scheme | Spice**;
- exact multi-range PSI references on annotation invocations, import paths,
  imported symbols, aliases, namespaces, and already-authored Go interface
  operands provide modifier-hover underlining, `Ctrl`/`Cmd`-click, and
  **Go to Declaration or Usages** navigation; interface operands use the
  shared compiler while the LSP is healthy and namespace-import-aware local PSI
  resolution while it restarts;
- `@Implements` completion comes only from the shared Spice compiler's loaded
  Go type universe. Accepting a named interface can atomically insert a
  namespace `@import` for any package in the Go module graph. The plugin never
  scans GoLand's type index to invent DI candidates or writes generated
  assertions into handwritten source;
- `Alt+Enter` on an authored `@Implements` offers native GoLand method
  generation, including generic/embedded methods and the compiler-selected
  pointer/value receiver. This action authors inspectable Go methods; only the
  Spice compiler validates the binding, generates its compile assertion, and
  makes it selectable for injection;
- annotation lines carry a native gutter marker whose tooltip names the exact
  descriptor package and symbol, module version/replacement, authorized tool,
  handler, and protocol, and whose navigation target is that real Go
  declaration;
- annotated `main` packages receive a preferred Spice Application
  configuration: Run invokes `spice run` for the selected target and complete
  package pattern, while Debug first executes a registered `spice generate`
  task and then delegates to GoLand's native complete-package Go/Delve path;
  neither path can use the temporary single-file runner, drop generated
  bootstrap files, or interpret folded presentation as source;
- an old or explicitly selected Go single-file configuration fails before
  execution when its source contains a Spice application or a raw annotation;
  the diagnostic directs the developer to the complete-package Spice
  configuration instead of allowing a `gocommand-*` fragment;
- named, aliased, and namespace-qualified references resolve to the real
  one-file Go SDK descriptor selected by the file's explicit
  `@import`;
- the public GoLand LSP API launches `spice lsp` for versioned diagnostics,
  import and annotation completion, rich descriptor hover, parameter
  information, safe code actions, module/configuration metadata, definition
  links, and handler implementation links.
- an indexed completion fallback continues offering already imported
  annotations while the LSP restarts, and a read-only **Spice** health window
  reports the exact executable, Spice/Go versions, module root, LSP state,
  vendor/read-only mode, authorized `go.mod` tools, and the last bounded
  failure without downloading or modifying dependencies.
- the hard-cut `@spice.import` diagnostic remains visibly un-folded and offers
  one previewable, undoable replacement of only the retired token with
  `@import`; the plugin never treats both spellings as valid.

The source file remains ordinary Go. Copying, saving, `gofmt`, Git, the Go
compiler, and generated code all retain the physical `// ` characters.
Concealment and color are editor presentation only. Ordinary Go language
features remain owned by GoLand's Go plugin.

The typed conversion is intentionally narrow. It applies only at a
horizontal-whitespace-only package position or inside a function parameter
list. Typing `@` inside a function body, string, comment, or after existing Go
tokens remains ordinary Go editing.

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
`%LOCALAPPDATA%\Programs\GoLand` installation. On a headless Linux host,
`make verify` launches the GoLand gate through `xvfb-run -a`; a visible display
is used unchanged when `DISPLAY` is already set.

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
`editors/goland/build/distributions/spice-goland-0.2.0.zip`. In GoLand, open
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

`make goland` does more than instantiate folding APIs. On Windows and Linux it
builds the current Spice CLI, packages the plugin, launches that archive in a
clean pinned GoLand profile through JetBrains Starter/Driver. The Windows
harness identifies the exact project-titled window and requires foreground
ownership before physical input. It first attaches Win32 input threads and,
when the foreground lock still rejects programmatic activation, clicks the
window title bar once and rechecks the exact foreground handle. It then:

1. opens a real Go module through the registered plugin extensions;
2. runs native highlighting and asserts the exact text-attribute key and range
   for sigils, namespaces, annotation names, parameters, literals, and
   punctuation;
3. runs GoLand's registered folding pipeline and asserts that every exact
   annotation and annotation-import prefix region is empty-placeholder,
   collapsed, and non-expandable;
4. proves the editor document, committed PSI, saved virtual file, and copied
   selection all retain the physical `// ` prefixes after concealment;
5. proves `@Application` produces a persisted Spice Application configuration
   with no single-file paths, an exact `spice run` command, one enabled
   generate-before-debug task, and higher context priority than every ordinary
   Go application configuration;
6. types `@` at a declaration boundary, saves it, verifies physical `// @`,
   exercises undo, redo, reformat, close, and reopen, and proves the original
   file is restored byte-for-byte;
7. measures the installed editor coordinates and proves folded annotations
   start at the same horizontal coordinate as their declaration;
8. switches the installed IDE between light and dark themes and captures the
   real editor component from the packaged plugin;
9. compares those captures and the faster realized-component renders with
   committed fixed-theme goldens, enforcing bounded mean and changed-region
   tolerances while retaining exact token/range assertions;
10. rejects blank or degenerate images and protocol/source corruption;
11. verifies that empty LSP diagnostic publications contain
    `diagnostics: []`, so GoLand clears stale errors instead of rejecting a
    null payload;
12. clicks the installed `Run Application` gutter marker, selects the preferred
   Spice configuration from GoLand's native menu, captures the exact
   command-package `spice run` invocation and live application-start output,
   then stops the process through its IDE process handler;
13. moves the real mouse with the platform modifier held, proves the exact
    annotation range visibly underlines, and Ctrl-clicks through to the real
    descriptor source;
14. opens installed Quick Documentation and verifies visible GoDoc, descriptor,
    targets, module/replacement provenance, tool authorization, handler,
    protocol, and implementation source sections;
15. opens the installed Spice health window, verifies every operational field,
    and records the rendered health surface;
16. rejects both raw and commented Spice applications presented to GoLand's
    temporary single-file runner before any `gocommand-*` source is created;
17. completes `@Implements` from the compiler-owned interface catalog, verifies
    the exact namespace import, invokes native missing-method generation with a
    pointer receiver and inspectable body, runs Spice generation, and verifies
    the source-owned generated assertion while handwritten annotations remain
    physical comments;
18. packages the plugin, validates its structure/configuration, and runs the
   JetBrains binary/API verifier.
19. sets a real XDebugger breakpoint on the physical `os.Exit` line, launches
   native Go/Delve Debug after guarded Spice generation, requires an enabled
   Resume action and suspended `main.main` frame at `main.go`, captures the
   Debug tool window, and stops the debug process through the IDE.

The generated visual reports are:

```text
editors/goland/build/reports/visual/spice-annotations-light.png
editors/goland/build/reports/visual/spice-annotations-dark.png
editors/goland/build/reports/visual/spice-installed-light.png
editors/goland/build/reports/visual/spice-installed-dark.png
editors/goland/build/reports/visual/spice-installed-documentation.png
editors/goland/build/reports/visual/spice-installed-health.png
editors/goland/build/reports/visual/spice-installed-gutter.txt
editors/goland/build/reports/visual/spice-installed-run.txt
editors/goland/build/reports/visual/spice-installed-debug.txt
editors/goland/build/reports/visual/spice-installed-debug-breakpoint.png
```

The reviewed baselines live in
`editors/goland/src/test/resources/goldens` for realized-component tests and
`editors/goland/src/integrationTest/resources/goldens` for the packaged-plugin
run. Block normalization and explicit tolerances absorb bounded platform-font
and antialiasing drift without silently accepting a missing fold, theme-wide
color regression, blank render, or major layout shift. The build reports
remain the exact current render for human inspection on Windows and Linux.

The Petclinic execution test is part of ordinary `make goland` and therefore of
the `make verify` matrix. Windows and Linux also run the installed-plugin UI
suite; macOS currently runs compile, unit, execution, packaging, and Plugin
Verifier coverage while its stable UI runner remains pending. The UI suite
is launched in its own non-parallel Gradle invocation after unit, packaging,
structure, and Plugin Verifier checks finish. This keeps one installed IDE
owner and prevents the verifier from contending with the test profile/cache.
The UI suite
sets `SPICE_EXECUTABLE` to the exact freshly built repository binary in
addition to prepending its directory to `PATH`, so Windows executable lookup
cannot silently select a stale globally installed language server. The suite
opens the real Petclinic module and its committed nested application source
unit before checking the application source. It uses platform-native process invocation
and temporary output paths; no shell-specific command or fixture binary is
hidden in the plugin.

Generation analysis uses the `spice_generate` build tag, which deliberately
excludes committed generated files. The shared loader verifies the annotated
entrypoint's exact generated-package import and `Main` call, then supplies that
package through the in-memory analysis overlay. It never writes a stub or
suppresses an undefined symbol. Both the LSP and plugin reject every unrelated
Go error. Run and Debug always execute the complete package, so no
`gocommand-*` file fragment or bridge-specific inspection suppression exists.

The schema-5 ownership manifest is the source/generated navigation contract.
`spice generated --source ...` and `--generated ...` expose the same mapping in
stable text or JSON for IDE and debugger integrations.

The folding parser is intentionally narrow: it recognizes declaration comments
whose complete text begins with canonical `// @`, including explicit
`@import` declarations. It never conceals ordinary comments or a
coincidental `@` later in prose. The Spice compiler/LSP remains the semantic
authority for targets, arguments, module rules, descriptor metadata, and
diagnostics. It also owns the named runtime-interface catalog, exact type
identity, method sets, pointer/value compatibility, provider candidates, and
all DI selection. The plugin's lexer is presentation-only. Its Go PSI use is
limited to navigation and safe authoring for a symbol already returned or
written by the developer; it cannot make a type injectable.

## Descriptor navigation and documentation

An annotation reference is never resolved from a hidden built-in table.
`@import` establishes the local symbol table:

```go
// @import { Controller, Get as GET } from "github.com/StevenBuglione/spice/annotation/web"
// @import * as security from "github.com/StevenBuglione/spice/annotation/security"

// @Controller
// @GET(path="/orders/{id}")
// @security.Authorize(anyRoles=["admin"])
```

GoLand's native PSI reference resolves `Controller`, `GET`, and
`security.Authorize` to their exact descriptor functions, so modifier-hover,
`Ctrl`/`Cmd`-click, and `Ctrl`/`Cmd+B` open ordinary indexed Go source.
`spice lsp` supplies the same real descriptor location to every editor and
provides **Go to Implementation** for the descriptor's declared handler source
symbol. GoLand's native definition search also maps the indexed descriptor
function to that handler, so implementation navigation survives an LSP
restart. Local application modules, vendor source, and local `replace` targets
are resolved read-only from the application's `go.mod`.

Quick Documentation and hover combine the descriptor GoDoc with its typed
arguments, defaults and allowed values, targets, examples, compatibility
range, resolved module version or local replacement, authorized tool path,
handler identity, protocol, and implementation symbol. Signature help uses the
same argument metadata. Selecting an annotation completion either uses its
existing local alias or inserts the required explicit named/namespace import;
the inserted edit is visible source, never an implicit compiler binding.
Pre-import suggestions come from the shared compiler service's bounded offline
scan of the target Go module graph, workspace, replacements, vendor tree, and
module cache. Their detail identifies package, version or replacement, tool,
and target-module authorization before the edit is selected. GoLand neither
maintains a private annotation registry nor searches the network.

When that tool is not yet authorized, GoLand exposes the shared LSP's two-step
quick fix. The first action previews the exact standard `go get -tool` command
and complete `go.mod`/`go.sum` diff without changing the project. The second,
separately selected **Apply previewed** action is the confirmation and succeeds
only if the original module-file hashes still match. No plugin-private
dependency resolver, background download, or direct unpreviewed edit exists.

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
gradlew.bat -PgolandPath=C:\path\to\GoLand integrationTest
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
- Run intentionally uses Spice's guarded generate/build/run pipeline. Debug
  intentionally retains GoLand's native Go debugger after the explicit
  generation prerequisite, so breakpoints, Delve behavior, and package
  inspection remain ordinary GoLand features.
- Definition navigation requires resolvable descriptor Go source. Missing
  module-cache or vendor source remains an actionable offline compiler
  diagnostic; the plugin does not download it.
- LSP start failures are reported by GoLand and do not disable native
  concealment, coloring, imported-symbol completion, documentation, or PSI
  navigation to already indexed descriptors.
