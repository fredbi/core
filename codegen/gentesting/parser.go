package gentest

import (
	"fmt"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// parser parses a go package at the designated location and identifies the exported symbols
type parser struct {
	location string
	pkg      *packages.Package
	symbols  symbolsIndex
	pkgName  string
	pkgPath  string
}

func newParser(location string) *parser {
	return &parser{
		location: location,
		symbols:  make(symbolsIndex),
	}
}

func (p *parser) Parse() error {
	const parseMode = packages.NeedName | packages.NeedImports | packages.NeedDeps |
		packages.NeedTarget | packages.NeedTypesInfo | packages.NeedFiles | packages.NeedTypes

	cfg := &packages.Config{
		Mode: parseMode,
		Dir:  p.location,
	}
	pkgs, err := packages.Load(cfg)
	if err != nil {
		return err
	}

	if len(pkgs) == 0 {
		return fmt.Errorf("internal error: resolved no package")
	}

	if len(pkgs) > 1 {
		return fmt.Errorf("internal error: resolved more than one package")
	}

	pkg := pkgs[0]
	if pkg.Types == nil {
		return fmt.Errorf("internal error: resolved no types in package")
	}

	tpkg := pkg.Types

	p.pkgName = tpkg.Name()
	p.pkgPath = tpkg.Path()

	top := tpkg.Scope()
	names := top.Names()
	for _, name := range names { // we don't recurse here: only the top-level exported symbols are of interest for now
		object := top.Lookup(name)
		if object == nil {
			return fmt.Errorf("internal error nil types object")
		}

		if !object.Exported() {
			continue
		}

		switch object.(type) {
		case *types.Const:
			p.symbols[SymbolConst] = append(p.symbols[SymbolConst], symbol{
				Kind:  SymbolConst,
				Pkg:   object.Pkg().Name(),
				Ident: object.Id(),
			})
		case *types.TypeName:
			p.symbols[SymbolType] = append(p.symbols[SymbolType], symbol{
				Kind:  SymbolType,
				Pkg:   object.Pkg().Name(),
				Ident: object.Id(),
			})
		case *types.Var:
			p.symbols[SymbolVar] = append(p.symbols[SymbolVar], symbol{
				Kind:  SymbolVar,
				Pkg:   object.Pkg().Name(),
				Ident: object.Id(),
			})
		case *types.Func:
			p.symbols[SymbolFunc] = append(p.symbols[SymbolFunc], symbol{
				Kind:  SymbolFunc,
				Pkg:   object.Pkg().Name(),
				Ident: object.Id(),
			})
		}
	}

	return nil
}

// Symbols returns the index of all exported symbols
func (p parser) Symbols() symbolsIndex {
	return p.symbols
}

// PackageName returns the package name
func (p parser) PackageName() string {
	return p.pkgName
}

// PackageName returns the package import path
func (p parser) PackageImportPath() string {
	return p.pkgPath
}
