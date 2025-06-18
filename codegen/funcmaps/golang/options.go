package golang

import "github.com/fredbi/core/mangling"

type Option func(o *options)

type options struct {
	manglerOptions []mangling.Option
}

var defaultOptions = options{
	manglerOptions: []mangling.Option{
		mangling.WithGoNamePrefixFunc(prefixForName),
	},
}

func optionsWithDefaults(opts []Option) options {
	o := defaultOptions
	for _, apply := range opts {
		apply(&o)
	}

	return o
}

// WithNameManglerOptions defines options to tune the [mangling.NameMangler] used by this [template.FuncMap].
func WithNameManglerOptions(opts ...mangling.Option) Option {
	return func(o *options) {
		o.manglerOptions = opts
	}
}
