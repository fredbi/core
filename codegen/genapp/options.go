package genapp

import (
	"io/fs"
	"regexp"
	"strings"
	"sync"

	golangfunc "github.com/fredbi/core/codegen/funcmaps/golang"
	repo "github.com/fredbi/core/codegen/templates-repo"
	"github.com/spf13/afero"
	"golang.org/x/tools/imports"
)

// Option to configure a [GoGenApp].
type Option func(*options)

type options struct {
	baseFS           afero.Fs
	outputPath       string
	templates        *repo.Repository
	templatesOptions []repo.Option
	goFormatter      func(name string, content []byte) ([]byte, error)
	skipFormatFunc   func(name string) (skipped bool)

	goOptions
}

type goOptions struct {
	skipFmt         bool
	skipCheckImport bool
	tabWidth        int
	formatOptions   imports.Options
	localPrefixes   []string
}

// goFormat formats go code and checks imports.
func (o goOptions) goFormat(name string, content []byte) ([]byte, error) {
	return imports.Process(name, content, &o.formatOptions)
}

var globalMx sync.Mutex

func optionsWithDefaults(templatesFS fs.FS, opts []Option) options {
	const defaultTabWidth = 2

	o := options{
		baseFS: afero.NewOsFs(),
		templatesOptions: []repo.Option{
			repo.WithFS(templatesFS),
			repo.WithFuncMap(golangfunc.DefaultFuncMap()),
		},
		goOptions: goOptions{
			formatOptions: imports.Options{
				TabIndent: true,
				TabWidth:  defaultTabWidth,
				Fragment:  true,
				Comments:  true,
			},
			localPrefixes: []string{"github.com/fredbi/core"},
		},
	}

	for _, apply := range opts {
		apply(&o)
	}

	if o.baseFS == nil {
		o.baseFS = afero.NewOsFs()
	}

	if len(o.templatesOptions) == 0 {
		// case of options reset, would panic
		o.templatesOptions = append(o.templatesOptions, repo.WithFS(templatesFS))
	}

	if o.templates == nil {
		o.templates = repo.New(o.templatesOptions...)
	}

	if o.goFormatter == nil {
		o.goFormatter = o.goFormat
	}

	if o.skipFormatFunc == nil {
		goFileRex := regexp.MustCompile(`\.go$`)
		o.skipFormatFunc = func(target string) bool { return !goFileRex.MatchString(target) }
	}

	o.formatOptions.FormatOnly = !o.skipCheckImport
	o.formatOptions.TabWidth = o.tabWidth

	if len(o.localPrefixes) > 0 {
		prefixes := strings.Join(o.localPrefixes, ",")
		globalMx.Lock()
		if imports.LocalPrefix != prefixes {
			imports.LocalPrefix = prefixes
			globalMx.Unlock()
		}
	}

	return o
}

// WithFormatTabWidth specifies the width of indentation tabs used when formatting go code.
//
// The default is 2. Notice that go defaults to 8.
//
// This is disabled when using [WithFormatter].
func WithFormatTabWidth(tabWidth int) Option {
	return func(o *options) {
		o.tabWidth = tabWidth
	}
}

// WithFormatGroupPrefixes adds local prefixes to group imports when formating code.
//
// The default is to group imports to "github.com/fredbi/core".
//
// NOTE: this affects the global setting [imports.LocalPrefix], and may cause side effects on other components
// using [imports.Process].
//
// Applies only if go import is enabled.
// This is disabled when using [WithFormatter].
func WithFormatGroupPrefixes(prefixes ...string) Option {
	return func(o *options) {
		o.localPrefixes = append(o.localPrefixes, prefixes...)
	}
}

// SetFormatGroupPrefixes overrides all default prefixes used to group imports when formatting code.
//
// Setting prefixes to an empty list disables the grouping of imports.
//
// NOTE: this affects the global setting [imports.LocalPrefix], and may cause side effects on other components
// using [imports.Process].
//
// Applies only if go import is enabled.
// This is disabled when using [WithFormatter].
func SetFormatGroupPrefixes(prefixes ...string) Option {
	return func(o *options) {
		o.localPrefixes = prefixes
	}
}

// WithOutputAferoFS specifies the output file system as an [afero.Fs].
//
// By default, the local os file system is used.
// This is primarily intended for testing.
func WithOutputAferoFS(fs afero.Fs) Option {
	return func(o *options) {
		if fs != nil {
			o.baseFS = fs
		}
	}
}

// WithOutputPath specifies the location of generated files on the os file system.
//
// All rendered files are relative to that path.
func WithOutputPath(path string) Option {
	return func(o *options) {
		if path != "" {
			o.outputPath = path
		}
	}
}

// WithSkipFormat instructs the rendering not to format the generated code.
//
// Format is enabled by default. If formatting is disabled, import check is disabled too.
// This option is essentially intended for testing and debugging generated code.
//
// This is disabled when using [WithFormatter].
func WithSkipFormat(skipped bool) Option {
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
// This option is essentially intended for testing and debugging generated code.
//
// This is disabled when using [WithFormatter].
func WithSkipCheckImport(skipped bool) Option {
	return func(o *options) {
		o.skipCheckImport = skipped
	}
}

// WithTemplatesRepo specifies the templates repository directly, so you have full control.
//
// This overrides [WithTemplatesOptions].
//
// By default, a templates repo is created with the templates provided by [WithTemplates].
func WithTemplatesRepo(templates *repo.Repository) Option {
	return func(o *options) {
		o.templates = templates
	}
}

// WithTemplatesRepoOptions adds [repo.Option] s to configure the templates repository.
//
// The default options used for the templates repository are profiled for golang generation by other tools
// in github.com/fredbi/core modules:
//
//   - golang codegen funcmap: [golangfunc.DefaultFuncMap]
func WithTemplatesRepoOptions(templatesOptions ...repo.Option) Option {
	return func(o *options) {
		o.templatesOptions = append(o.templatesOptions, templatesOptions...)
	}
}

// SetTemplateOptions overrides defaults [repo.Option] s to configure the templates repository.
//
// Specifying an empty list disables any default options (such as the default func map).
func SetTemplateOptions(templatesOptions ...repo.Option) Option {
	return func(o *options) {
		o.templatesOptions = templatesOptions
	}
}

// WithFormatter allows for injecting a custom source code formatter.
//
// The default is [imports.Process].
func WithFormatter(formatter func(name string, content []byte) ([]byte, error)) Option {
	return func(o *options) {
		o.goFormatter = formatter
	}
}

// WithSkipFormatFunc specifies a function to skip formatting on targets.
//
// If the function returns true for a given target, rendering will skip formatting.

// The default is to skip formatting on files that don't have the ".go" extension.
// This setting is disabled if you enable [WithSkipFormat] (disabled for all files).
func WithSkipFormatFunc(skipFunc func(name string) (skipped bool)) Option {
	return func(o *options) {
		o.skipFormatFunc = skipFunc
	}
}
