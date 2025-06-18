package schema

import (
	"github.com/fredbi/core/genmodels/generators/golang-models/ifaces"
	"github.com/fredbi/core/genmodels/generators/golang-models/providers"
)

// Option defines settings for the [Builder].
type Option func(*options)

type options struct {
	nameProvider ifaces.EnumNameProvider // TODO: used for enums and exported vs not-exported identifiers
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

// WithNameProvider equips the schema [Builder] with an [ifaces.EnumNameProvider] so as to resolve
// names for constants or variables derived from enum values.
func WithNameProvider(provider ifaces.EnumNameProvider) Option {
	return func(o *options) {
		o.nameProvider = provider
	}
}
