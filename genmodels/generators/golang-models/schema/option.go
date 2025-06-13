package schema

import "github.com/fredbi/core/genmodels/generators/golang-models/providers"

// Option defines settings for the [Builder].
type Option func(*options)

type options struct {
	nameProvider NameProvider // TODO: used for enums and exported vs not-exported identifiers
}

func optionsWithDefaults(opts []Option) options {
	var o options

	for _, apply := range opts {
		apply(&o)
	}

	if o.nameProvider == nil {
		o.nameProvider = providers.NewNameProvider()
	}

	return o
}

// WithNameProvider equips the schema [Builder] with a [NameProvider] so as to resolve
// names for constants or variables derived from enum values.
func WithNameProvider(provider NameProvider) Option {
	return func(o *options) {
		o.nameProvider = provider
	}
}
