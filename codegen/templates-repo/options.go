package repo

import (
	"fmt"
	"io/fs"
	"strings"
	"text/template"

	fsutils "github.com/fredbi/core/swag/fs"
	"github.com/fredbi/core/swag/mangling"
)

var defaultOptions = options{
	baseFS:          fsutils.NewReadOnlyOsFS(),
	funcs:           make(template.FuncMap),
	mangler:         mangling.Make(),
	allowOverride:   false,
	extensions:      []string{".gotmpl"},
	skipDirectories: []string{"contrib"},
	parseComments:   false,
	dumpTemplate:    markdownTemplate,
	cover:           false,
}

// Option defines settings for the template repository.
//
// # Settings
//
// * provide a file system, including an "embed.FS" (the default is the actual file system supported by the os)
// * include a functions map [template.FuncMap] to supplement template builtins (none is provided by default),
// * define protected templates (none is protected by default)
// * disable check on protected templates (check is enabled by default)
// * define supported template file extensions (the default is ".gotmpl")
// * define skipped subdirectories when using [Repository.Load] (the default is to skip "contrib" folders)
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
	allowOverride   bool
	cover           bool
}

func WithSkipDirectories(dir ...string) Option {
	return func(o *options) {
		o.skipDirectories = dir
	}
}

func WithExtensions(ext ...string) Option {
	return func(o *options) {
		o.extensions = ext
	}
}

func WithFS(base fs.FS) Option {
	return func(o *options) {
		if readFS, ok := base.(fs.ReadFileFS); ok {
			o.baseFS = readFS

			return
		}

		o.baseFS = fsutils.NewFileReaderFS(base)
	}
}

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

func WithOverlays(overlays ...fs.FS) Option {
	return func(o *options) {
		o.overlays = overlays
	}
}

func WithManglingOptions(opts ...mangling.Option) Option {
	return func(o *options) {
		o.mangler = mangling.Make(opts...)
	}
}

func WithFuncMap(funcs template.FuncMap) Option {
	return func(o *options) {
		o.funcs = funcs
	}
}

func WithParseComments(enabled bool) Option {
	return func(o *options) {
		o.parseComments = enabled
	}
}

func WithAllowOverride(enabled bool) Option {
	return func(o *options) {
		o.allowOverride = enabled
	}
}

func WithDumpTemplate(text string) Option {
	return func(o *options) {
		if text != "" {
			o.dumpTemplate = text
		}
	}
}

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
