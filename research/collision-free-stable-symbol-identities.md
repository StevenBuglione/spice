# Collision-Free Stable Symbol Identities

Date: 2026-07-24

## Question

How should Spice serialize package and declaration identities so every valid Go program receives deterministic, collision-free, readable compiler keys across annotation resolution, dependency injection, generated code, module graphs, diagnostics, and architecture documents?

## Why this is the highest-value current question

Issue #8 originally specified flat dot-concatenated identities:

```text
package:   <package-path>
type:      <package-path>.<type-name>
function:  <package-path>.<function-name>
method:    <package-path>.<receiver-origin-name>.<method-name>
variable:  <package-path>.<variable-name>
constant:  <package-path>.<constant-name>
```

That format is not injective. Dots are valid inside Go module and package paths. A valid module can therefore contain both:

```text
package example.com/collision/p with a type T
package example.com/collision/p.T
```

Both records flatten to:

```text
example.com/collision/p.T
```

The same collision exists for functions, variables, constants, and methods when packages such as `p.F`, `p.V`, `p.C`, or `p.T.M` coexist with declarations in package `p`.

This is not a presentation defect. Future compiler phases will index annotations, providers, graph nodes, routes, module dependencies, generated bindings, and documentation by stable identity. A collision would silently overwrite valid metadata or make output dependent on insertion order.

## Primary-source findings

### 1. Go package paths cannot be treated as dot-free names

The Go module reference defines a package path as a module path joined with the package subdirectory. Module path elements may contain ASCII letters, digits, and `-`, `.`, `_`, and `~`; dots are intentionally common, including versioned `gopkg.in` paths.

Sources:

- https://go.dev/ref/mod#module-path
- https://go.dev/ref/mod#module-paths-and-versions

The Go specification defines an import path as a string literal whose interpretation is implementation-dependent. The standard compiler permits punctuation beyond identifier syntax and does not establish a delimiter that application frameworks may safely assume will never occur in every package path returned by every package driver.

Source:

- https://go.dev/ref/spec#Import_declarations

Consequence: concatenating a package path and declaration components with any unescaped delimiter is not a durable identity contract.

### 2. Go identifiers are structured names, not globally qualified strings

The Go specification permits Unicode letters and digits in identifiers. Package-level identifiers are unique only inside their package block; methods are unique only for a receiver base type and method name. `init` and `_` do not introduce addressable bindings.

Sources:

- https://go.dev/ref/spec#Identifiers
- https://go.dev/ref/spec#Declarations_and_scope
- https://go.dev/ref/spec#Method_declarations

The `go/types` API preserves this structure through `types.Object.Pkg`, `Name`, object kind, and receiver signatures. `types.Object.Id` is not a global durable symbol identity: it returns an exported name without package qualification and qualifies only unexported names.

Source:

- https://pkg.go.dev/go/types#Object
- https://pkg.go.dev/go/types#Id

Consequence: Spice should serialize an explicit tuple rather than attempting to reuse a display-oriented Go string.

### 3. `x/tools/go/types/objectpath` is useful evidence but not the Spice contract

`objectpath` exists specifically because in-memory `types.Object` pointers do not survive across processes. It represents an object relative to a package and can later resolve the path against another equivalent package.

Source:

- https://pkg.go.dev/golang.org/x/tools/go/types/objectpath

It is not a complete replacement for Spice stable IDs:

- the path excludes the package itself;
- the format is explicitly opaque;
- it does not guarantee paths for unexported package-level non-types, while Spice must model unexported providers and annotated declarations inside application packages;
- one object may have multiple paths, with `For` choosing one consistently but arbitrarily;
- it models a wider API object graph than the bounded top-level Spice declaration catalog.

`objectpath` is BSD-3-Clause as part of `golang.org/x/tools`, which Spice already pins for package loading. Reusing it would add no new license family, but its semantic limitations still make it unsuitable as the primary serialized ID.

### 4. Identity and display should be separate contracts

Spring and Go both retain structured identity internally. Spring bean definitions distinguish bean names, types, factory methods, and resource descriptions instead of deriving all identity from one punctuation-joined string. Go retains package, object, type, receiver, and source position separately.

Relevant Spring sources:

- https://docs.spring.io/spring-framework/reference/core/beans/definition.html
- https://docs.spring.io/spring-framework/reference/core/beans/dependencies/factory-collaborators.html

Spice should similarly keep:

- a canonical machine identity used for equality, maps, serialization, and generated ownership;
- structured fields used by compiler logic;
- a concise display label used in diagnostics and diagrams.

A display label such as `example.com/app/orders.Service.Create` may remain convenient, but it must never be a key.

## Decision

Adopt a versioned structured symbol key and a canonical length-prefixed UTF-8 serialization.

Conceptual key:

```go
type SymbolKey struct {
    Version     uint8
    Kind        Kind
    PackagePath string
    Receiver    string
    Name        string
}
```

Required semantic fields:

| Kind | PackagePath | Receiver | Name |
|---|---|---|---|
| package | required | empty | empty |
| type | required | empty | required |
| function | required | empty | required |
| method | required | normalized origin name | required |
| variable | required | empty | required |
| constant | required | empty | required |

The method receiver remains the defining named receiver origin already required by issue #8. Pointer spelling, generic instantiation arguments, and function signatures remain outside logical identity.

## Canonical string grammar

Version 1 uses this uniform grammar:

```text
spice:symbol:v1|<kind>|<package-field>|<receiver-field>|<name-field>

field = <canonical-decimal-byte-length> ":" <exact-UTF-8-bytes>
```

Rules:

1. `kind` is one of `package`, `type`, `function`, `method`, `variable`, or `constant`.
2. Field lengths count UTF-8 bytes, not Unicode code points.
3. Zero is encoded as `0`; positive lengths have no leading zero.
4. Components are copied exactly. Do not case-fold, path-clean, Unicode-normalize, percent-decode, or rewrite module escape syntax.
5. The encoder always emits all three fields, including empty receiver and name fields.
6. A decoder, when added, must consume exactly the declared byte lengths, reject trailing bytes, reject unknown versions and kinds, reject overflow, and validate the kind-specific empty/non-empty invariants.
7. The version marker is part of the ID. A future incompatible identity model uses a new version rather than silently reinterpreting stored keys.

Examples:

```text
package example.com/foo.bar
spice:symbol:v1|package|19:example.com/foo.bar|0:|0:

type T in example.com/foo.bar
spice:symbol:v1|type|19:example.com/foo.bar|0:|1:T

function Build in example.com/foo.bar
spice:symbol:v1|function|19:example.com/foo.bar|0:|5:Build

method M on T in example.com/foo.bar
spice:symbol:v1|method|19:example.com/foo.bar|1:T|1:M
```

Length-prefixing remains unambiguous even if a future package driver returns a component containing `|`, `:`, dots, slashes, Unicode, or text resembling another encoded field. The parser trusts lengths, not delimiter absence inside component data.

## Why this encoding

### Collision-free by construction

The ID is an encoding of a typed tuple. Two IDs are equal only when version, kind, package path, receiver, and name are byte-for-byte equal.

### Deterministic

There is exactly one encoding for each valid key. There are no optional escapes, equivalent percent-encoding spellings, map ordering concerns, whitespace choices, or JSON encoder variations.

### Readable enough for repository artifacts

Package paths and declaration names remain visible in generated manifests, logs, golden tests, and architecture documents. The length prefixes add noise but avoid opaque hashes and preserve direct debugging value.

### Versionable

Generated files and future persistent indexes can reject unsupported identity versions explicitly.

### Dependency-free

The encoder requires only standard-library string or byte building and decimal formatting. No new dependency or license obligation is introduced.

### Linear and bounded

Encoding and parsing are O(total component bytes). This work is negligible relative to `packages.Load` and type checking. Implementations should pre-size builders where convenient but must not introduce caching or global registries.

## Rejected alternatives

### Continue flat dot concatenation

Rejected because valid package paths create proven collisions with every supported declaration form.

### Choose a different bare separator

Rejected because it ties identity correctness to current path restrictions, custom package-driver behavior, and identifier grammar. It also requires a future migration if the allowed input domain expands.

### Percent-escaped URI-like IDs

Potentially workable, but only with a strict canonical escape policy. It adds questions around which bytes must be escaped, uppercase versus lowercase hex, invalid sequences, and whether decoding occurs before comparison. Length-prefixing is simpler and has one representation.

### Canonical JSON arrays

A JSON array can represent the tuple without collisions, but it is noisy in generated identifiers and still requires an exact policy for escaping, versioning, and unknown fields. JSON is better suited to a manifest containing a structured `SymbolKey`, not the compact map key itself.

### Content hashes

Rejected as the primary ID. Hashes are opaque, introduce collision probability rather than a mathematical uniqueness guarantee, complicate diagnostics, and require side metadata to recover the declaration. A hash may later fingerprint a signature or generated artifact, but it should not replace logical identity.

### `types.Object.Id`

Rejected because its documented behavior is package-local and export-sensitive, not globally unique across package-level objects and methods.

### `objectpath.Path`

Rejected as the primary ID because it omits package identity, is opaque, does not cover all unexported package-level non-types, and may choose among multiple valid paths.

## Ordering contract

Canonical serialization and user-facing ordering are separate concerns.

Do not rely on the encoded string's lexical order as the architectural presentation order. The length prefix and kind token are part of serialization, not a sorting language.

Sort symbol records by the structured tuple:

1. package path;
2. declaration kind using one documented fixed rank;
3. receiver origin;
4. declaration name;
5. physical source filename and offset as deterministic tie-breakers.

The global uniqueness invariant should make the position tie-breaker unnecessary for addressable symbols, but retaining it in a comparator is safe defensive behavior for diagnostics during malformed-program handling.

## Public API direction

The smallest issue #8 recovery may preserve the existing field shape while centralizing construction:

```go
type SymbolID string

func newSymbolID(kind Kind, packagePath, receiver, name string) SymbolID
```

No caller should concatenate IDs directly. Package symbols and `Package.ID` should use the same canonical package key, while `Package.Path` remains the ordinary Go import path.

A public parser is not required in the bootstrap loader. Before any persistent manifest or third-party compiler API depends on the string, add a strict parser and round-trip tests. Compiler logic should prefer the structured `Kind`, `PackagePath`, `Receiver`, and `Name` fields already present on `Symbol` instead of reparsing IDs.

## Regression contract

### Valid collision matrix

Create one fixture module containing:

```text
example.com/collisionmatrix/p
example.com/collisionmatrix/p.C
example.com/collisionmatrix/p.F
example.com/collisionmatrix/p.T
example.com/collisionmatrix/p.V
example.com/collisionmatrix/p.T.M
```

Package `p` declares:

```go
const C = 1
var V = 1
type T struct{}
func F() {}
func (T) M() {}
```

Load `./...` and assert:

- every expected package and declaration remains present;
- the package symbols do not collide with the declaration symbols;
- every returned `Symbol.ID` is globally unique across the complete `Program`;
- every `Package.ID` matches its package symbol's canonical ID;
- package paths and display labels remain unchanged and readable.

### Encoder tests

Add table-driven tests for:

- every supported kind;
- empty fields required by package and non-method declarations;
- Unicode Go identifiers using byte lengths;
- component text containing dots, slashes, colons, pipes, digits, and prefix-like strings in an isolated encoder test;
- exact deterministic output;
- round-trip decode if a decoder is implemented;
- malformed lengths, leading zeros, overflow, unknown version, unknown kind, missing fields, invalid invariants, and trailing bytes if parsing is implemented.

### Mutation proof

Temporarily restore the old flat-dot encoder. The collision-matrix or global uniqueness test must fail and report the colliding records.

### Full proof

The implementation must still run the issue #8 command set, `make verify`, vendor-only tests, race tests, cgo integration, source-mapping tests, active-cancellation integration, and repeated deterministic tests.

## Security and robustness

- Treat IDs as data, never as filesystem paths, Go identifiers, shell fragments, URLs, or source code.
- Never derive generated output paths by splitting or decoding a display label.
- Do not include absolute source paths, temporary directories, environment values, signatures, or source contents in identity components.
- Reject malformed externally supplied encoded IDs before allocating based on claimed lengths; bounds must be checked against remaining input and integer overflow.
- Preserve exact case. Go package paths can differ by case, even though module storage uses special escaping to survive case-insensitive filesystems.
- Do not Unicode-normalize identifiers. Go source identity follows the exact identifier spelling accepted by the compiler.
- A future JSON or database representation should store structured fields alongside the canonical versioned string when query ergonomics matter.

## Compatibility and migration

Spice is pre-1.0 and issue #8 has not merged, so this correction should replace the invalid bootstrap format now rather than preserve aliases.

Required migration actions:

1. Replace the flat-dot contract in issue #8, `research/type-aware-package-loading.md`, loader documentation, tests, and PR evidence.
2. Centralize all ID creation in one encoder.
3. Update exact-order golden expectations.
4. Search issues #11, #13, #17, generation research, module research, and documentation for direct string concatenation assumptions. References to an opaque stable ID remain valid; examples prescribing the old grammar must change.
5. Do not support both old and new IDs. Dual identities would make generated ownership and graph keys ambiguous before the first release.
6. Record the format version in any future generation manifest or serialized compiler output.

## Spring capability relationship

Spring's application model depends on unambiguous bean definitions, factory methods, injection points, handlers, and module metadata. Spice's compile-time equivalent needs the same logical guarantee before it can safely implement:

- constructor and factory-provider graphs;
- interface binding adapters;
- controller and route ownership;
- generated application bootstrap and lifecycle;
- Spring Modulith-style module nodes, named interfaces, allowed dependencies, and documentation;
- test slices and generated metadata freshness.

The capability relationship is not that Spring uses this string format. It is that both platforms require structured declaration identity. Spice should implement that outcome with a Go-native, deterministic, inspectable encoding rather than JVM reflection objects or runtime class names.

## License and dependency impact

- Go specification, standard-library, and toolchain documentation are provided by the Go project under its existing BSD-style licensing terms.
- `golang.org/x/tools/go/types/objectpath` is BSD-3-Clause and already resides in the dependency family pinned by issue #8.
- The recommendation introduces no new runtime or build dependency.

## Bounded implementation sequence

The active issue #8 lane should perform only this correction:

1. define the versioned symbol-key encoder;
2. route package and declaration ID construction through it;
3. preserve separate human-readable package/name/receiver fields;
4. sort by structured fields rather than treating serialized syntax as presentation policy;
5. add the valid collision matrix and global uniqueness assertions;
6. update the incorrect contract and evidence;
7. run the complete existing proof suite and hand off the exact unchanged SHA.

Parsing externally supplied IDs, persistent indexes, backwards compatibility, signature fingerprints, local declarations, fields, parameters, test variants, and third-party compiler APIs remain out of scope.

## Recommendation

Replace flat dot-concatenated IDs before issue #8 merges. Use a versioned structured key with canonical length-prefixed UTF-8 serialization, keep display labels separate, centralize encoding, and prove global uniqueness across a valid package/declaration collision matrix. This is the smallest correction that makes the typed loader safe for every downstream Spring-style and Modulith-style compiler phase.