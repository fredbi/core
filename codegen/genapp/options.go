package genapp

import (
	"fmt"
	"io/fs"

	golangfunc "github.com/fredbi/core/codegen/funcmaps/golang"
	repo "github.com/fredbi/core/codegen/templates-repo"
	"github.com/spf13/afero"
	"golang.org/x/tools/imports"
)

// Option to configure a [GenApp] helper.
type Option func(*options)

type options struct {
	baseFS     afero.Fs
	outputPath string
	templates  *repo.Repository

	goOptions
}

type goOptions struct {
	skipFmt         bool
	skipCheckImport bool
	tabWidth        int
	formatOptions   imports.Options
}

func optionsWithDefaults(opts []Option) options {
	const defaultTabWidth = 2

	o := options{
		baseFS: afero.NewOsFs(),
		goOptions: goOptions{
			formatOptions: imports.Options{
				TabIndent: true,
				TabWidth:  defaultTabWidth,
				Fragment:  true,
				Comments:  true,
			},
		},
	}

	for _, apply := range opts {
		apply(&o)
	}

	o.formatOptions.FormatOnly = !o.skipCheckImport
	o.formatOptions.TabWidth = o.tabWidth

	return o
}

func (o options) validateOptions() error {
	if o.baseFS == nil {
		return fmt.Errorf("GenApp requires an output FS to be specified in options. Use WithAferoFS(): %w", ErrGenApp)
	}

	if o.templates == nil {
		return fmt.Errorf("GenApp requires a templates repository to be specified in options. Use WithTemplatesRepo() or WithTemplates(): %w", ErrGenApp)
	}

	return nil
}

// WithFormatTabWidth specifies the width of indentation tabs used when formatting go code.
//
// The default is 2. Notice that go defaults to 8.
func WithFormatTabWidth(tabWidth int) Option {
	return func(o *options) {
		o.tabWidth = tabWidth
	}
}

// WithOutputAferoFS specifies the output file system as an [afero.Fs].
//
// By default, the local os file system is used.
//
// This is primarily intended for testing.
func WithOutputAferoFS(fs afero.Fs) Option {
	return func(o *options) {
		if fs != nil {
			o.baseFS = fs
		}
	}
}

// WithOutputPath specifies the location of generated files.
func WithOutputPath(path string) Option {
	return func(o *options) {
		if path != "" {
			o.outputPath = path
		}
	}
}

// WithSkipFmt instructs the rendering not to format the generated code.
//
// Format is enabled by default. If formatting is disabled, import check is disabled too.
func WithSkipFmt(skipped bool) Option {
	return func(o *options) {
		o.skipFmt = skipped
		if skipped {
			o.skipCheckImport = skipped
		}
	}
}

// WithSkipCheckImport instructs the rendering not to run go imports on the generated code.
//
// Import check is enabled by default. You may disable imports check and keep simple code formatting.
func WithSkipCheckImport(skipped bool) Option {
	return func(o *options) {
		o.skipCheckImport = skipped
	}
}

// WithTemplateRepo specifies the templates repository directly, so you have full control.
func WithTemplateRepo(templates *repo.Repository) Option {
	return func(o *options) {
		o.templates = templates
	}
}

// WithTemplates creates and configures a templates repository from a [fs.FS] location.
//
// The default options used for the templates repository are profiled for golang generation for go-openapi:
// * golang codegen funcmap
func WithTemplates(templatesFS fs.FS, templatesOptions ...repo.Option) Option {
	defaultTemplatesOptions := []repo.Option{
		repo.WithFS(templatesFS),
		repo.WithFuncMap(golangfunc.DefaultFuncMap()),
	}
	defaultTemplatesOptions = append(defaultTemplatesOptions, templatesOptions...)
	return func(o *options) {
		o.templates = repo.New(defaultTemplatesOptions...)
	}
}
