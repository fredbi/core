package repo

import (
	"io/fs"
	"os"
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

type options struct {
	fs              fs.ReadFileFS
	funcs           template.FuncMap
	mangler         mangling.NameMangler
	allowOverride   bool
	extensions      []string
	skipDirectories []string
	parseComments   bool
	dumpTemplate    string
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

// osfs exposes package os features as an fs.FS, without having to use [os.Root].
type osfs struct {
}

func (f *osfs) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (f *osfs) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (f *osfs) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

// readfilefs makes a [fs.FS] into a [fs.ReadFileFS]
type readfilefs struct {
	fs.FS
}

func (f *readfilefs) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(f.FS, name)
}

func optionsWithDefaults(opts []Option) options {
	o := defaultOptions

	for _, apply := range opts {
		apply(&o)
	}

	return o
}
