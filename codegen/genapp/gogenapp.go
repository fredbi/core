package genapp

import (
	"bytes"
	"errors"
	"path/filepath"
	"sync"

	repo "github.com/fredbi/core/codegen/templates-repo"
	"github.com/spf13/afero"
	"golang.org/x/tools/imports"
)

// FileNamer is a type that knows the destination file for generated content.
type FileNamer interface {
	FileName() string
}

// GoGenApp is a helper type to generate golang code.
type GoGenApp struct {
	options
	fs        afero.Fs
	loaded    *sync.Once
	loadError error
}

// New builds a new [GoGenApp], as a pointer.
func New(opts ...Option) *GoGenApp {
	g := Make(opts...)

	return &g
}

// Make builds a new [GoGenApp], as a value.
func Make(opts ...Option) GoGenApp {
	g := GoGenApp{
		options: optionsWithDefaults(opts),
		loaded:  &sync.Once{},
	}

	if err := g.validateOptions(); err != nil {
		panic(err)
	}

	g.fs = afero.NewBasePathFs(g.baseFS, g.outputPath)

	return g
}

// Render data using a template from a template repository.
//
// The only thing that [GoGenApp] needs to know about your data is where to place it
// on the output FS. Hence, every target data is a [FileNamer].
//
// Note: internally, [GoGenApp] uses [afero.FS] to specify the output, so all targets are relative to that path.
//
// By default, the generated output is subject to go format and go imports.
func (g *GoGenApp) Render(template string, target FileNamer) error {
	g.loaded.Do(func() {
		g.loadError = g.templates.Load(".")
	})
	if g.loadError != nil {
		return g.loadError
	}

	tpl, ert := g.templates.Get(template)
	if ert != nil {
		return errors.Join(ert, ErrGenApp)
	}

	var buffer bytes.Buffer

	if err := tpl.Execute(&buffer, target); err != nil {
		return errors.Join(err, ErrGenApp)
	}

	name := target.FileName()
	if !g.skipFmt || !g.skipCheckImport {
		fullyQualifiedName := filepath.Join(g.outputPath, name)
		formatted, err := g.goFormat(fullyQualifiedName, buffer.Bytes())
		if err != nil {
			return errors.Join(err, ErrGenApp)
		}
		buffer.Reset()
		_, _ = buffer.Write(formatted)
	}

	if err := afero.WriteReader(g.fs, name, &buffer); err != nil { // TODO: check permissions on created files.
		return errors.Join(err, ErrGenApp)
	}

	return nil
}

// Templates returns the inner templates [repo.Repository] used for code generation.
func (g *GoGenApp) Templates() *repo.Repository {
	return g.templates
}

// goFormat formats go code and checks imports.
func (g *GoGenApp) goFormat(name string, content []byte) ([]byte, error) {
	return imports.Process(name, content, &g.formatOptions)
}
