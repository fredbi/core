package providers

import "github.com/fredbi/core/swag/mangling"

type Option func(o *options)

type options struct {
	manglingOptions []mangling.Option
}

func optionsWithDefaults(opts []Option) options {
	o := options{}

	for _, apply := range opts {
		apply(&o)
	}

	return o
}

func WithManglerOptions(opts ...mangling.Option) Option {
	return func(o *options) {
		o.manglingOptions = opts
	}
}
