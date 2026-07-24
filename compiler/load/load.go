package load

import (
	"context"
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"
	"sort"

	"golang.org/x/tools/go/packages"
)

// Options configures one isolated package-loading operation.
type Options struct {
	Dir        string
	Env        []string
	BuildFlags []string
	Overlay    map[string][]byte
	// Tests is reserved for a future test-package model. The bootstrap loader
	// rejects true because go/packages test variants can duplicate logical
	// package and symbol identities.
	Tests bool
}

// Load asks the standard Go package driver to load the requested root package
// patterns once, then builds deterministic package, symbol, and diagnostic
// records from that shared type universe.
func Load(ctx context.Context, options Options, patterns ...string) (*Program, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(patterns) == 0 {
		diagnostic := Diagnostic{Kind: "list", Message: "no package patterns were provided"}
		program := &Program{diagnostics: []Diagnostic{diagnostic}}
		return program, &LoadError{Diagnostics: program.Diagnostics()}
	}
	if options.Tests {
		diagnostic := Diagnostic{
			Kind:    "configuration",
			Message: "test-variant loading is unsupported; load application packages with Tests disabled",
		}
		program := &Program{diagnostics: []Diagnostic{diagnostic}}
		return program, &LoadError{Diagnostics: program.Diagnostics()}
	}

	config := &packages.Config{
		Context:    ctx,
		Mode:       packages.LoadSyntax | packages.NeedModule,
		Dir:        options.Dir,
		Env:        append([]string(nil), options.Env...),
		BuildFlags: append([]string(nil), options.BuildFlags...),
		Overlay:    cloneOverlay(options.Overlay),
		Tests:      false,
	}

	roots, loadErr := packages.Load(config, patterns...)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	program := &Program{}
	if loadErr != nil {
		program.diagnostics = append(program.diagnostics, Diagnostic{
			Kind:    "driver",
			Message: loadErr.Error(),
		})
	}

	for _, root := range roots {
		if root == nil {
			continue
		}
		record := packageRecord(root)
		program.packages = append(program.packages, record)
		program.symbols = append(program.symbols, packageSymbols(root)...)
		program.diagnostics = append(program.diagnostics, packageDiagnostics(root)...)
	}

	sort.SliceStable(program.packages, func(i, j int) bool {
		if program.packages[i].Path != program.packages[j].Path {
			return program.packages[i].Path < program.packages[j].Path
		}
		return program.packages[i].ID < program.packages[j].ID
	})
	sort.SliceStable(program.symbols, func(i, j int) bool {
		left, right := program.symbols[i], program.symbols[j]
		if left.ID != right.ID {
			return left.ID < right.ID
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		if left.Position.Filename != right.Position.Filename {
			return left.Position.Filename < right.Position.Filename
		}
		return left.Position.Offset < right.Position.Offset
	})
	sortDiagnostics(program.diagnostics)

	if len(program.diagnostics) > 0 || loadErr != nil {
		return program, &LoadError{Diagnostics: program.Diagnostics()}
	}
	return program, nil
}

func cloneOverlay(overlay map[string][]byte) map[string][]byte {
	if len(overlay) == 0 {
		return nil
	}
	result := make(map[string][]byte, len(overlay))
	for path, content := range overlay {
		result[path] = append([]byte(nil), content...)
	}
	return result
}

func packageRecord(root *packages.Package) Package {
	files := append([]string(nil), root.CompiledGoFiles...)
	for i := range files {
		files[i] = filepath.Clean(files[i])
	}
	sort.Strings(files)

	dir := ""
	if len(files) > 0 {
		dir = filepath.Dir(files[0])
	}
	modulePath := ""
	if root.Module != nil {
		modulePath = root.Module.Path
	}

	return Package{
		ID:              root.PkgPath,
		Path:            root.PkgPath,
		Name:            root.Name,
		Dir:             dir,
		ModulePath:      modulePath,
		CompiledGoFiles: files,
		IllTyped:        root.IllTyped || len(root.Errors) > 0,
		Types:           root.Types,
		TypesInfo:       root.TypesInfo,
		Syntax:          append([]*ast.File(nil), root.Syntax...),
		Raw:             root,
	}
}

func packageDiagnostics(root *packages.Package) []Diagnostic {
	diagnostics := make([]Diagnostic, 0, len(root.Errors))
	for _, packageError := range root.Errors {
		position := packageError.Pos
		if position != "" {
			position = filepath.Clean(position)
		}
		diagnostics = append(diagnostics, Diagnostic{
			PackagePath: root.PkgPath,
			Position:    position,
			Kind:        errorKindName(packageError.Kind),
			Message:     packageError.Msg,
		})
	}
	return diagnostics
}

func errorKindName(kind packages.ErrorKind) string {
	switch kind {
	case packages.ListError:
		return "list"
	case packages.ParseError:
		return "parse"
	case packages.TypeError:
		return "type"
	default:
		return "unknown"
	}
}

func packageSymbols(root *packages.Package) []Symbol {
	if root.Fset == nil || root.TypesInfo == nil {
		return nil
	}

	symbols := make([]Symbol, 0)
	if len(root.Syntax) > 0 && root.Syntax[0] != nil && root.Syntax[0].Name != nil {
		name := root.Syntax[0].Name
		symbols = append(symbols, Symbol{
			ID:          root.PkgPath,
			Kind:        SymbolPackage,
			Name:        root.Name,
			PackagePath: root.PkgPath,
			Position:    root.Fset.PositionFor(name.Pos(), true),
			Node:        name,
		})
	}

	for _, file := range root.Syntax {
		if file == nil {
			continue
		}
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.GenDecl:
				for _, specification := range declaration.Specs {
					switch specification := specification.(type) {
					case *ast.TypeSpec:
						if specification.Name.Name == "_" {
							continue
						}
						if object := root.TypesInfo.Defs[specification.Name]; object != nil {
							symbols = append(symbols, objectSymbol(root, object, specification, SymbolType, ""))
						}
					case *ast.ValueSpec:
						for _, name := range specification.Names {
							if name.Name == "_" {
								continue
							}
							object := root.TypesInfo.Defs[name]
							if object == nil {
								continue
							}
							kind := SymbolVariable
							if _, ok := object.(*types.Const); ok {
								kind = SymbolConstant
							}
							symbols = append(symbols, objectSymbol(root, object, name, kind, ""))
						}
					}
				}
			case *ast.FuncDecl:
				if declaration.Name.Name == "_" || (declaration.Recv == nil && declaration.Name.Name == "init") {
					continue
				}
				object, _ := root.TypesInfo.Defs[declaration.Name].(*types.Func)
				if object == nil {
					continue
				}
				signature, _ := object.Type().(*types.Signature)
				if declaration.Recv == nil {
					symbol := objectSymbol(root, object, declaration, SymbolFunction, "")
					symbol.Signature = signature
					symbols = append(symbols, symbol)
					continue
				}
				receiver, err := normalizedReceiverName(signature)
				if err != nil {
					// Ill-formed receiver declarations are already reported by go/types.
					continue
				}
				symbol := objectSymbol(root, object, declaration, SymbolMethod, receiver)
				symbol.Signature = signature
				symbols = append(symbols, symbol)
			}
		}
	}
	return symbols
}

func objectSymbol(root *packages.Package, object types.Object, node ast.Node, kind SymbolKind, receiver string) Symbol {
	id := root.PkgPath + "." + object.Name()
	if receiver != "" {
		id = root.PkgPath + "." + receiver + "." + object.Name()
	}
	return Symbol{
		ID:          id,
		Kind:        kind,
		Name:        object.Name(),
		PackagePath: root.PkgPath,
		Receiver:    receiver,
		Position:    root.Fset.PositionFor(object.Pos(), true),
		Object:      object,
		Node:        node,
	}
}

func normalizedReceiverName(signature *types.Signature) (string, error) {
	if signature == nil || signature.Recv() == nil {
		return "", fmt.Errorf("method signature has no receiver")
	}
	receiverType := signature.Recv().Type()
	if pointer, ok := receiverType.(*types.Pointer); ok {
		receiverType = pointer.Elem()
	}
	named, ok := receiverType.(*types.Named)
	if !ok {
		return "", fmt.Errorf("receiver %s is not a named type", types.TypeString(receiverType, nil))
	}
	origin := named.Origin()
	if origin == nil || origin.Obj() == nil {
		return "", fmt.Errorf("receiver %s has no defining origin", types.TypeString(receiverType, nil))
	}
	return origin.Obj().Name(), nil
}
