package repo

import (
	"io/fs"
	"strings"
	"text/template"

	"github.com/fredbi/core/swag/mangling"
)

var defaultOptions = options{
	fs:              &osfs{},
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

func WithFS(fs fs.FS) Option {
	return func(o *options) {
		o.fs = &readfilefs{FS: fs}
	}
}

func WithReadFileFS(fs fs.ReadFileFS) Option {
	return func(o *options) {
		o.fs = fs
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
		o.dumpTemplate = text
	}
}

func WithCoverProfile(enabled bool) Option {
	return func(o *options) {
		o.cover = enabled
	}
}

type options struct {
	fs              fs.ReadFileFS
	funcs           template.FuncMap
	mangler         mangling.NameMangler
	extensions      []string
	skipDirectories []string
	dumpTemplate    string
	parseComments   bool
	allowOverride   bool
	cover           bool
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
