package repo

import (
	"fmt"
	"io"
	"sort"

	"text/template"
)

// Dump prints out a dump of all the defined templates to some [io.Writer], where they are defined and what their dependencies are.
//
// This is intended to produce a documentation for your templates. The default formatting is a markdown document.
// You may customize how the dump is formatted with the option [WithDumpTemplate] in [New].
func (r *Repository) Dump(w io.Writer) error {
	render, err := template.New("dumpRepository").Funcs(r.funcs).Parse(r.dumpTemplate)
	if err != nil {
		panic(fmt.Errorf("%w: %w", err, ErrTemplateRepo))
	}
	r.mux.RLock()
	defer r.mux.RUnlock()

	sorted := make([]string, 0, len(r.templates))
	for name := range r.templates {
		sorted = append(sorted, name)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return r.files[sorted[i]] < r.files[sorted[j]]
	})

	var data dumpRepository
	for _, name := range sorted {
		tpl := r.templates[name]
		data.Templates = append(data.Templates, dumpTemplate{
			Name:         name,
			SourceAsset:  r.files[name],
			DocStrings:   r.docstrings[name],
			Dependencies: findDependencies(tpl.Root),
		})
	}

	err = render.Execute(w, data)
	if err != nil {
		return fmt.Errorf("%w: %w", err, ErrTemplateRepo)
	}

	return nil
}

type (
	dumpRepository struct {
		Templates []dumpTemplate
	}

	dumpTemplate struct {
		Name         string
		SourceAsset  string
		DocStrings   []string
		Dependencies []string
	}
)

// markdownTemplate is the default layout for [Repository.Dump].
const markdownTemplate = `
# Templates

{{ range .Templates }}
## {{ .Name }}

Defined in {{.SourceAsset}}

{{ range .DocStrings }}
  {{ . }}
{{ end }}

{{ if .Dependencies }}
#### Requires

{{ range .Dependencies }}
- {{ . }}
{{ end }}
---
{{ end }}
{{ end }}
`
