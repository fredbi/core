package genapp

import (
	"bytes"
	"errors"
	"io/fs"
	"path/filepath"
	"sync"

	repo "github.com/fredbi/core/codegen/templates-repo"
	"github.com/spf13/afero"
)

// GoGenApp is a helper type to generate golang code.
//
// It it is typically embedded in more complex tools.
//
// # Sccope
//
// [GoGenApp] roles and responsibilities are limited to:
//
//   - load a provided template [repo.Repository]
//   - execute templates using [GoGenApp.Render]
//   - apply the configured code formatting rules
//   - write the output target file.
//
// In addition [GoGenApp.GoMod] may be called to create and fill a "go.mod" file once the generation is complete.
//
// # Concurrency
//
// [GoGenApp] may be used to [GoGenApp.Render] files concurrently.
// However [GoGenApp.GoMod] is not safe for a concurrent use.
type GoGenApp struct {
	options
	fs        afero.Fs
	loaded    *sync.Once
	loadError error
}

// New builds a new [GoGenApp] with templates found on the provided [fs.FS], with [Option] s to configure the behavior.
//
// NOTE: templateFS may be left to nil if you provide a fully configured [repo.Repository] using [WithTemplatesRepo].
func New(templateFS fs.FS, opts ...Option) *GoGenApp {
	g := &GoGenApp{
		options: optionsWithDefaults(templateFS, opts),
		loaded:  &sync.Once{},
	}

	g.fs = afero.NewBasePathFs(g.baseFS, g.outputPath)

	return g
}

// Render data using a template from a template repository [repo.Repository].
//
// The repository is initialized on the first call.
//
// The target tells Render where to place the result on the output FS.
// The target path is relative to the output path of the [GoGenApp]
// and must be a valid relative path on the running OS.
//
// By default, the generated output is subject to go format and go imports.
//
// # Concurrency
//
// Render may be used concurrently.
func (g *GoGenApp) Render(template string, target string, data any) error {
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

	if err := tpl.Execute(&buffer, data); err != nil {
		return errors.Join(err, ErrGenApp)
	}

	if !g.skipFormatFunc(target) && (!g.skipFmt || !g.skipCheckImport) {
		fullyQualifiedName := filepath.Join(g.outputPath, target)
		formatted, err := g.goFormat(fullyQualifiedName, buffer.Bytes())
		if err != nil {
			return errors.Join(err, ErrGenApp)
		}
		buffer.Reset()
		_, _ = buffer.Write(formatted)
	}

	if err := afero.WriteReader(g.fs, target, &buffer); err != nil {
		return errors.Join(err, ErrGenApp)
	}

	return nil
}

// Templates returns the inner templates [repo.Repository] used for code generation.
//
// This provides the ability to reload or dump templates.
func (g *GoGenApp) Templates() *repo.Repository {
	return g.templates
}
