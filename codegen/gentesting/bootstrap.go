package gentest

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	"golang.org/x/tools/imports"
)

type reExported struct {
	Pkg       string
	Import    pair
	Constants []pair
	Variables []pair
	Types     []pair
	Functions []pair
}

type bootstrapper struct {
	imported string
	alias    string
	target   string
	symbols  symbolsIndex
}

func newBootStrapper(symbols symbolsIndex, imported string, alias string, target string) *bootstrapper {
	return &bootstrapper{
		symbols:  symbols,
		imported: imported,
		alias:    alias,
		target:   target,
	}
}

// Generate a bootstrap main source to build a plugin that imports the package of interest.
func (b *bootstrapper) Generate() error {
	tpl, erp := template.New("bootstrap").Parse(pluginWrapperTemplate)
	if erp != nil {
		return fmt.Errorf("internal error: can't parse plugin bootstrap template: %w", erp)
	}

	var buf bytes.Buffer
	data := b.buildData()
	if err := tpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("internal error: can't execute plugin bootstrap template: %w", err)
	}

	formatOptions := &imports.Options{
		TabIndent: true,
		Fragment:  true,
		Comments:  true,
	}

	content, erp := imports.Process(b.target, buf.Bytes(), formatOptions)
	if erp != nil {
		log.Println(buf.String())
		return fmt.Errorf("internal error: can't gofmt/goimports plugin bootstrap: %w", erp)
	}

	if err := os.WriteFile(b.target, content, 0644); err != nil { // TODO: use afero?
		return err
	}

	return nil
}

// buildData builds the data structure to feed to the bootstrap template
func (b *bootstrapper) buildData() reExported {
	d := reExported{
		Pkg: b.alias,
		Import: pair{
			Ident:  b.alias,
			Target: b.imported,
		},
	}

	for _, constant := range b.symbols[SymbolConst] {
		d.Constants = append(d.Constants, pair{
			Ident:  constant.Ident,
			Target: strings.Join([]string{constant.Pkg, constant.Ident}, "."),
		})
	}

	for _, variable := range b.symbols[SymbolVar] {
		d.Variables = append(d.Variables, pair{
			Ident:  variable.Ident,
			Target: strings.Join([]string{variable.Pkg, variable.Ident}, "."),
		})
	}

	for _, goType := range b.symbols[SymbolType] {
		d.Types = append(d.Types, pair{
			Ident:  goType.Ident,
			Target: strings.Join([]string{goType.Pkg, goType.Ident}, "."),
		})
	}

	for _, function := range b.symbols[SymbolFunc] {
		d.Functions = append(d.Functions, pair{
			Ident:  function.Ident,
			Target: strings.Join([]string{function.Pkg, function.Ident}, "."),
		})
	}

	return d
}

const pluginWrapperTemplate = `
// Package main bootstraps a plugin.
//
// It reexports all symbols from package {{ .Import.Target }}
package main

// Generated code. DO NOT EDIT

import (
	{{- with .Import }}
	  {{ .Ident }} "{{ .Target}}"
	{{- end }}
)
{{ with .Constants }}
var (
	// reexported constants as variables
	{{- range . }}
	  {{ .Ident }} = {{ .Target }}
	{{- end }}
)
{{- end }}
{{ with .Variables }}
var (
	// reexported variables
	{{- range . }}
	  {{ .Ident }} = {{ .Target }}
	{{- end }}
)
{{- end }}
{{ with .Types }}
var (
	// reexported types
	{{- range . }}
	  {{ .Ident }} {{ .Target }}
	{{- end }}
)
{{- end }}
{{ with .Functions }}
var (
	// reexported functions
	{{- range . }}
	  {{ .Ident }} = {{ .Target }}
	{{- end }}
)
{{- end }}
`
