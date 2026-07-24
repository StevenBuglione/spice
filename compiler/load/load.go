// Package load provides Spice's single type-aware Go package loading boundary.
//
// Each Load call creates an independent go/types universe. Objects and types
// returned by one Program must never be combined with values from another.
package load

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Config controls one package-loading operation. Its zero value uses the
// current process directory, environment, and active Go build context.
type Config struct {
	Dir        string
	Env        []string
	BuildFlags []string
	Overlay    map[string][]byte
	Tests      bool
}

// Program is the immutable logical result of one Load call. Its AST and type
// references belong to this Program's type universe and must be treated as
// read-only.
type Program struct {
	packages    []Package
	symbols     []Symbol
	diagnostics []Diagnostic
}

// Packages returns package records in deterministic package-path order.
func (program *Program) Packages() []Package {
	if program == nil {
		return nil
	}
	result := make([]Package, len(program.packages))
	copy(result, program.packages)
	return result
}

// Symbols returns declaration records in deterministic stable-ID order.
func (program *Program) Symbols() []Symbol {
	if program == nil {
		return nil
	}
	result := make([]Symbol, len(program.symbols))
	copy(result, program.symbols)
	return result
}

// Diagnostics returns normalized load diagnostics in deterministic order.
func (program *Program) Diagnostics() []Diagnostic {
	if program == nil {
		return nil
	}
	result := make([]Diagnostic, len(program.diagnostics))
	copy(result, program.diagnostics)
	return result
}

// Package describes one requested root package.
type Package struct {
	path            string
	name            string
	dir             string
	modulePath      string
	compiledGoFiles []string
	illTyped        bool
	typesPackage    *types.Package
	typesInfo       *types.Info
	fileSet         *token.FileSet
	syntax          []*ast.File
}

func (pkg Package) Path() string            { return pkg.path }
func (pkg Package) Name() string            { return pkg.name }
func (pkg Package) Dir() string             { return pkg.dir }
func (pkg Package) ModulePath() string      { return pkg.modulePath }
func (pkg Package) IllTyped() bool          { return pkg.illTyped }
func (pkg Package) Types() *types.Package   { return pkg.typesPackage }
func (pkg Package) TypesInfo() *types.Info  { return pkg.typesInfo }
func (pkg Package) FileSet() *token.FileSet { return pkg.fileSet }

// CompiledGoFiles returns a copy of the active build's compiled file set.
func (pkg Package) CompiledGoFiles() []string {
	result := make([]string, len(pkg.compiledGoFiles))
	copy(result, pkg.compiledGoFiles)
	return result
}

// Syntax returns a copy of the package's active syntax tree slice. The AST
// nodes themselves belong to the Program and must be treated as read-only.
func (pkg Package) Syntax() []*ast.File {
	result := make([]*ast.File, len(pkg.syntax))
	copy(result, pkg.syntax)
	return result
}

// SymbolID is a stable, serializable logical declaration identity.
type SymbolID string

// SymbolKind identifies a supported top-level declaration kind.
type SymbolKind string

const (
	SymbolPackage  SymbolKind = "package"
	SymbolType     SymbolKind = "type"
	SymbolFunction SymbolKind = "function"
	SymbolMethod   SymbolKind = "method"
	SymbolVariable SymbolKind = "variable"
	SymbolConstant SymbolKind = "constant"
)

// Symbol associates a stable ID with the live Go object and syntax node from
// one Program. Package symbols have no types.Object.
type Symbol struct {
	ID          SymbolID
	Kind        SymbolKind
	PackagePath string
	Name        string
	Receiver    string
	Position    token.Position
	Object      types.Object
	Node        ast.Node
	Signature   *types.Signature
}

// DiagnosticKind categorizes package-list, parse, type, and driver failures.
type DiagnosticKind string

const (
	DiagnosticList  DiagnosticKind = "list"
	DiagnosticParse DiagnosticKind = "parse"
	DiagnosticType  DiagnosticKind = "type"
	DiagnosticOther DiagnosticKind = "other"
)

// Diagnostic is one normalized, source-positioned package loading failure.
type Diagnostic struct {
	Kind        DiagnosticKind
	PackagePath string
	Position    token.Position
	Message     string
}

func (diagnostic Diagnostic) Error() string {
	position := diagnostic.Position
	if position.Filename == "" {
		position.Filename = diagnostic.PackagePath
	}
	if position.Line > 0 {
		if position.Column > 0 {
			return fmt.Sprintf("%s:%d:%d: %s", position.Filename, position.Line, position.Column, diagnostic.Message)
		}
		return fmt.Sprintf("%s:%d: %s", position.Filename, position.Line, diagnostic.Message)
	}
	return fmt.Sprintf("%s: %s", position.Filename, diagnostic.Message)
}

// LoadError reports that requested root packages could not be safely used for
// semantic generation. The returned Program still contains normalized
// diagnostics and any successfully loaded root metadata.
type LoadError struct {
	Diagnostics []Diagnostic
	Cause       error
}

func (loadError *LoadError) Error() string {
	if len(loadError.Diagnostics) == 0 {
		if loadError.Cause != nil {
			return fmt.Sprintf("load Go packages: %v", loadError.Cause)
		}
		return "load Go packages failed"
	}
	if len(loadError.Diagnostics) == 1 {
		return loadError.Diagnostics[0].Error()
	}
	return fmt.Sprintf("load Go packages: %d diagnostics", len(loadError.Diagnostics))
}

func (loadError *LoadError) Unwrap() error { return loadError.Cause }

// Load performs exactly one go/packages load for the supplied patterns.
// Patterns are passed to the Go package driver unchanged.
func Load(ctx context.Context, config Config, patterns ...string) (*Program, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return &Program{}, &LoadError{Cause: err}
	}

	packageConfig := &packages.Config{
		Mode:       packages.LoadSyntax | packages.NeedModule,
		Context:    ctx,
		Dir:        config.Dir,
		Env:        cloneStrings(config.Env),
		BuildFlags: cloneStrings(config.BuildFlags),
		Overlay:    cloneOverlay(config.Overlay),
		Tests:      config.Tests,
		Logf:       func(string, ...any) {},
	}

	loaded, loadErr := packages.Load(packageConfig, patterns...)
	program := &Program{}
	for _, loadedPackage := range loaded {
		if loadedPackage == nil {
			continue
		}
		program.packages = append(program.packages, packageRecord(loadedPackage))
		program.symbols = append(program.symbols, packageSymbols(loadedPackage)...)
		program.diagnostics = append(program.diagnostics, packageDiagnostics(loadedPackage)...)
	}

	sort.Slice(program.packages, func(i, j int) bool {
		return program.packages[i].path < program.packages[j].path
	})
	sortSymbols(program.symbols)
	sortDiagnostics(program.diagnostics)

	if ctxErr := ctx.Err(); ctxErr != nil {
		return program, &LoadError{Diagnostics: program.Diagnostics(), Cause: ctxErr}
	}
	if loadErr != nil || len(program.packages) == 0 || len(program.diagnostics) > 0 || hasIllTyped(program.packages) {
		if len(program.packages) == 0 && len(program.diagnostics) == 0 {
			program.diagnostics = append(program.diagnostics, Diagnostic{
				Kind:     DiagnosticList,
				Position: token.Position{Filename: strings.Join(patterns, ", ")},
				Message:  "package pattern matched no packages",
			})
		}
		return program, &LoadError{Diagnostics: program.Diagnostics(), Cause: loadErr}
	}
	return program, nil
}

func packageRecord(pkg *packages.Package) Package {
	files := cloneStrings(pkg.CompiledGoFiles)
	syntax := make([]*ast.File, len(pkg.Syntax))
	copy(syntax, pkg.Syntax)
	sort.Strings(files)
	sort.Slice(syntax, func(i, j int) bool {
		return pkg.Fset.Position(syntax[i].Pos()).Filename < pkg.Fset.Position(syntax[j].Pos()).Filename
	})
	modulePath := ""
	if pkg.Module != nil {
		modulePath = pkg.Module.Path
	}
	return Package{
		path:            pkg.PkgPath,
		name:            pkg.Name,
		dir:             pkg.Dir,
		modulePath:      modulePath,
		compiledGoFiles: files,
		illTyped:        pkg.IllTyped,
		typesPackage:    pkg.Types,
		typesInfo:       pkg.TypesInfo,
		fileSet:         pkg.Fset,
		syntax:          syntax,
	}
}

func packageSymbols(pkg *packages.Package) []Symbol {
	if pkg.TypesInfo == nil || pkg.Fset == nil {
		return nil
	}
	result := []Symbol{{
		ID:          SymbolID(pkg.PkgPath),
		Kind:        SymbolPackage,
		PackagePath: pkg.PkgPath,
		Name:        pkg.Name,
		Position:    packagePosition(pkg),
	}}

	for _, file := range pkg.Syntax {
		for _, declaration := range file.Decls {
			switch node := declaration.(type) {
			case *ast.FuncDecl:
				object, ok := pkg.TypesInfo.Defs[node.Name].(*types.Func)
				if !ok || object == nil {
					continue
				}
				signature, _ := object.Type().(*types.Signature)
				symbol := Symbol{
					Kind:        SymbolFunction,
					PackagePath: pkg.PkgPath,
					Name:        object.Name(),
					Position:    pkg.Fset.Position(node.Name.Pos()),
					Object:      object,
					Node:        node,
					Signature:   signature,
				}
				if node.Recv != nil {
					symbol.Kind = SymbolMethod
					symbol.Receiver = receiverOriginName(signature)
					symbol.ID = SymbolID(pkg.PkgPath + "." + symbol.Receiver + "." + object.Name())
				} else {
					symbol.ID = SymbolID(pkg.PkgPath + "." + object.Name())
				}
				result = append(result, symbol)
			case *ast.GenDecl:
				for _, spec := range node.Specs {
					switch typedSpec := spec.(type) {
					case *ast.TypeSpec:
						if object := pkg.TypesInfo.Defs[typedSpec.Name]; object != nil {
							result = append(result, objectSymbol(pkg, object, typedSpec.Name, typedSpec, SymbolType))
						}
					case *ast.ValueSpec:
						for _, name := range typedSpec.Names {
							object := pkg.TypesInfo.Defs[name]
							if object == nil {
								continue
							}
							kind := SymbolVariable
							if _, ok := object.(*types.Const); ok {
								kind = SymbolConstant
							}
							result = append(result, objectSymbol(pkg, object, name, typedSpec, kind))
						}
					}
				}
			}
		}
	}
	return result
}

func objectSymbol(pkg *packages.Package, object types.Object, name *ast.Ident, node ast.Node, kind SymbolKind) Symbol {
	return Symbol{
		ID:          SymbolID(pkg.PkgPath + "." + object.Name()),
		Kind:        kind,
		PackagePath: pkg.PkgPath,
		Name:        object.Name(),
		Position:    pkg.Fset.Position(name.Pos()),
		Object:      object,
		Node:        node,
	}
}

func receiverOriginName(signature *types.Signature) string {
	if signature == nil || signature.Recv() == nil {
		return "<receiver>"
	}
	receiver := signature.Recv().Type()
	if pointer, ok := receiver.(*types.Pointer); ok {
		receiver = pointer.Elem()
	}
	named, ok := receiver.(*types.Named)
	if !ok {
		return types.TypeString(receiver, packagePathQualifier)
	}
	origin := named.Origin()
	if origin != nil {
		named = origin
	}
	if named.Obj() == nil {
		return types.TypeString(named, packagePathQualifier)
	}
	return named.Obj().Name()
}

func packagePathQualifier(pkg *types.Package) string {
	if pkg == nil {
		return ""
	}
	return pkg.Path()
}

func packagePosition(pkg *packages.Package) token.Position {
	if pkg.Fset == nil || len(pkg.Syntax) == 0 || pkg.Syntax[0] == nil {
		return token.Position{Filename: pkg.Dir}
	}
	return pkg.Fset.Position(pkg.Syntax[0].Package)
}

func packageDiagnostics(pkg *packages.Package) []Diagnostic {
	result := make([]Diagnostic, 0, len(pkg.Errors))
	for _, packageError := range pkg.Errors {
		result = append(result, Diagnostic{
			Kind:        diagnosticKind(packageError.Kind),
			PackagePath: pkg.PkgPath,
			Position:    parseErrorPosition(packageError.Pos, pkg.PkgPath),
			Message:     strings.TrimSpace(packageError.Msg),
		})
	}
	return result
}

func diagnosticKind(kind packages.ErrorKind) DiagnosticKind {
	switch kind {
	case packages.ListError:
		return DiagnosticList
	case packages.ParseError:
		return DiagnosticParse
	case packages.TypeError:
		return DiagnosticType
	default:
		return DiagnosticOther
	}
}

func parseErrorPosition(value, fallback string) token.Position {
	value = strings.TrimSpace(value)
	if value == "" || value == "-" {
		return token.Position{Filename: fallback}
	}
	position := token.Position{Filename: value}
	last := strings.LastIndex(value, ":")
	if last < 0 {
		return position
	}
	lastValue, err := strconv.Atoi(value[last+1:])
	if err != nil {
		return position
	}
	before := value[:last]
	second := strings.LastIndex(before, ":")
	if second < 0 {
		position.Filename = before
		position.Line = lastValue
		return position
	}
	secondValue, err := strconv.Atoi(before[second+1:])
	if err != nil {
		position.Filename = before
		position.Line = lastValue
		return position
	}
	position.Filename = before[:second]
	position.Line = secondValue
	position.Column = lastValue
	return position
}

func sortSymbols(symbols []Symbol) {
	sort.Slice(symbols, func(i, j int) bool {
		left, right := symbols[i], symbols[j]
		if left.ID != right.ID {
			return left.ID < right.ID
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		if left.Position.Filename != right.Position.Filename {
			return left.Position.Filename < right.Position.Filename
		}
		if left.Position.Line != right.Position.Line {
			return left.Position.Line < right.Position.Line
		}
		return left.Position.Column < right.Position.Column
	})
}

func sortDiagnostics(diagnostics []Diagnostic) {
	sort.Slice(diagnostics, func(i, j int) bool {
		left, right := diagnostics[i], diagnostics[j]
		if left.Position.Filename != right.Position.Filename {
			return left.Position.Filename < right.Position.Filename
		}
		if left.Position.Line != right.Position.Line {
			return left.Position.Line < right.Position.Line
		}
		if left.Position.Column != right.Position.Column {
			return left.Position.Column < right.Position.Column
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		if left.PackagePath != right.PackagePath {
			return left.PackagePath < right.PackagePath
		}
		return left.Message < right.Message
	})
}

func hasIllTyped(packages []Package) bool {
	for _, pkg := range packages {
		if pkg.illTyped {
			return true
		}
	}
	return false
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	result := make([]string, len(values))
	copy(result, values)
	return result
}

func cloneOverlay(values map[string][]byte) map[string][]byte {
	if values == nil {
		return nil
	}
	result := make(map[string][]byte, len(values))
	for path, contents := range values {
		copyContents := make([]byte, len(contents))
		copy(copyContents, contents)
		result[path] = copyContents
	}
	return result
}

// IsContextError reports whether err was caused by context cancellation or timeout.
func IsContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// Summary returns a deterministic serialization-friendly view for tests,
// diagnostics, generated metadata, and architecture tooling.
func (program *Program) Summary() Summary {
	summary := Summary{}
	for _, pkg := range program.packages {
		summary.Packages = append(summary.Packages, PackageSummary{
			Path:            pkg.path,
			Name:            pkg.name,
			ModulePath:      pkg.modulePath,
			CompiledGoFiles: baseNames(pkg.compiledGoFiles),
			IllTyped:        pkg.illTyped,
		})
	}
	for _, symbol := range program.symbols {
		summary.Symbols = append(summary.Symbols, SymbolSummary{
			ID:       symbol.ID,
			Kind:     symbol.Kind,
			Name:     symbol.Name,
			Receiver: symbol.Receiver,
		})
	}
	for _, diagnostic := range program.diagnostics {
		summary.Diagnostics = append(summary.Diagnostics, DiagnosticSummary{
			Kind:        diagnostic.Kind,
			PackagePath: diagnostic.PackagePath,
			File:        filepath.Base(diagnostic.Position.Filename),
			Line:        diagnostic.Position.Line,
			Column:      diagnostic.Position.Column,
			Message:     diagnostic.Message,
		})
	}
	return summary
}

// Summary is deterministic and independent of temporary absolute directories.
type Summary struct {
	Packages    []PackageSummary
	Symbols     []SymbolSummary
	Diagnostics []DiagnosticSummary
}

type PackageSummary struct {
	Path            string
	Name            string
	ModulePath      string
	CompiledGoFiles []string
	IllTyped        bool
}

type SymbolSummary struct {
	ID       SymbolID
	Kind     SymbolKind
	Name     string
	Receiver string
}

type DiagnosticSummary struct {
	Kind        DiagnosticKind
	PackagePath string
	File        string
	Line        int
	Column      int
	Message     string
}

func baseNames(paths []string) []string {
	result := make([]string, len(paths))
	for index, path := range paths {
		result[index] = filepath.Base(path)
	}
	sort.Strings(result)
	return result
}
