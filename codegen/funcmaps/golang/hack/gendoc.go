//go:build ignore

package main

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"text/template"

	"github.com/fredbi/core/codegen/funcmaps/golang"
)

func main() {
	if err := generateMarkdown(); err != nil {
		panic(fmt.Errorf("%w: %w", err, golang.ErrFuncMap))
	}
}

func generateMarkdown() error {
	render, err := template.New("dumpFuncMap").Funcs(template.FuncMap{"backtick": func() string { return "`" }}).Parse(docTemplate)
	if err != nil {
		return errors.Join(err, golang.ErrFuncMap)
	}

	m := golang.DefaultFuncMap()

	entries := slices.Sorted(maps.Keys(m))

	data := dumpFuncMap{
		Title:   "golang default funcmap",
		Entries: make([]dumpEntry, 0, len(entries)),
	}

	for _, fn := range entries {
		data.Entries = append(data.Entries, dumpEntry{
			Name: fn,
		})
	}

	w, err := os.Create("README.md")
	if err != nil {
		return err
	}

	if err = render.Execute(w, data); err != nil {
		return errors.Join(err, golang.ErrFuncMap)
	}

	return nil
}

/*
TODO: idea parse comment on private funcs
func parse() error {
	const parseMode = packages.NeedName | packages.NeedImports | packages.NeedDeps |
		packages.NeedTarget | packages.NeedTypesInfo | packages.NeedFiles | packages.NeedTypes

	cfg := &packages.Config{
		Mode: parseMode,
		Dir:  ".",
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
	tpkg :=
	top := pkg.Scope()
}
*/

type (
	dumpFuncMap struct {
		Title   string
		Entries []dumpEntry
	}

	dumpEntry struct {
		Name string
	}
)

const docTemplate = `# {{ .Title }}

List of functions exposed to templates.

## Available functions
{{ range .Entries }}
	{{ .Name }}
{{- end }}

---

> This documentation has been generated from source.
> Please run {{ backtick }}go generate{{ backtick }} to update this file.
`
