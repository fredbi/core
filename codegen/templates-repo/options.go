package repo

import (
	"fmt"
	"io/fs"
	"strings"
	"text/template"

	fsutils "github.com/fredbi/core/swag/fs"
	"github.com/fredbi/core/mangling"
)

var defaultOptions = options{
	baseFS:          fsutils.NewReadOnlyOsFS(),
	funcs:           make(template.FuncMap),
	mangler:         mangling.Make(),
	extensions:      []string{".gotmpl"},
	skipDirectories: []string{"contrib"},
	dumpTemplate:    markdownTemplate,
}

// Option defines settings for the template repository.
//
// # Supported settings
//
//   - provide a file system, including an [embed.FS] (the default is the actual file system supported by the os)
//   - include a functions map [template.FuncMap] to supplement template builtins (none is provided by default),
//   - define supported template file extensions (the default is ".gotmpl")
//   - define skipped subdirectories when using [Repository.Load] (the default is to skip "contrib" folders)
type Option func(*options)

type options struct {
	baseFS          fs.ReadFileFS
	overlays        []fs.FS
	funcs           template.FuncMap
	mangler         mangling.NameMangler
	extensions      []string
	skipDirectories []string
	dumpTemplate    string
	parseComments   bool
	cover           bool
}

// WithSkipDirectories alters how [Repository.Load] will resolve templates:
// loading will skip the directories with the specified suffixes from the [fs.FS].
//
// The default is to skip folder ending with "contrib".
func WithSkipDirectories(suffixes ...string) Option {
	return func(o *options) {
		o.skipDirectories = suffixes
	}
}

// WithExtensions sets supported file extensions recognized as go templates.
//
// The default is ".gotmpl"
func WithExtensions(ext ...string) Option {
	return func(o *options) {
		o.extensions = ext
	}
}

// WithFS provides a base [fs.FS] from where to load templates.
//
// This is mutually exclusive with [WithLocalPath].
func WithFS(base fs.FS) Option {
	if base == nil {
		panic(fmt.Errorf("the provided base fs.FS is nil: %w", ErrTemplateRepo))
	}

	return func(o *options) {
		if readFS, ok := base.(fs.ReadFileFS); ok {
			o.baseFS = readFS

			return
		}

		o.baseFS = fsutils.NewFileReaderFS(base)
	}
}

// WithLocalPath loads templates from a folder in the os file system.
//
// All template are resolved relative to the path of this folder.
//
// This is mutually exclusive with [WithFS].
func WithLocalPath(folder string) Option {
	localFS := fsutils.NewReadOnlyOsFS()
	subFS, err := fs.Sub(localFS, folder)
	if err != nil {
		panic(fmt.Errorf("could not access %q: %w: %w", folder, err, ErrTemplateRepo))
	}
	subReadFileFS := fsutils.NewFileReaderFS(subFS)

	return func(o *options) {
		o.baseFS = subReadFileFS
	}
}

// WithOverlays specifies overlay [fs.FS] to override the base [fs.FS] provided.
func WithOverlays(overlays ...fs.FS) Option {
	return func(o *options) {
		o.overlays = overlays
	}
}

// WithManglingOptions alters how template name resolution is done.
//
// By default, template naming convention uses [mangling.NameMangler.ToJSONName].
// These options affect the [mangling.NameMangler].
func WithManglingOptions(opts ...mangling.Option) Option {
	return func(o *options) {
		o.mangler = mangling.Make(opts...)
	}
}

// WithFuncMap injects a [template.FuncMap] to bind to the templates.
//
// By default, there is no funcmap added and only built-in functions are available to templates.
func WithFuncMap(funcs template.FuncMap) Option {
	return func(o *options) {
		o.funcs = funcs
	}
}

// WithParseComments instructs the [Repository] to parse comments in templates
// (i.e. "{{/* ... */}"" constructs).
//
// This is used when producing a documentation for templates using [Repository.Dump]
func WithParseComments(enabled bool) Option {
	return func(o *options) {
		o.parseComments = enabled
	}
}

// WithDumpTemplate provides a template to be used by [Repository.Dump] when reporting
// about the templates structure.
//
// By default, a simple markdown template is provided.
//
// When [WithParseComments] is enabled, the [Repository] may use comments in templates as docstrings
// in a dump produced by [Repository.Dump].
func WithDumpTemplate(text string) Option {
	return func(o *options) {
		if text != "" {
			o.dumpTemplate = text
		}
	}
}

// WithCoverProfile enable an experimental feature that captures test coverage when executing templates.
func WithCoverProfile(enabled bool) Option {
	return func(o *options) {
		o.cover = enabled
	}
}

func (o options) cloneOptions(opts []Option) options {
	clone := o

	for _, apply := range opts {
		apply(&clone)
	}

	return clone
}

func (o options) trimExtension(name string) string {
	for _, ext := range o.extensions {
		if trimmed, ok := strings.CutSuffix(name, ext); ok {
			return trimmed
		}
	}

	return name
}

func optionsWithDefaults(opts []Option) options {
	o := defaultOptions

	for _, apply := range opts {
		apply(&o)
	}

	return o
}
