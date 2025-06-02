package structural

// Option customizes the behavior of the [SchemaAnalyzer].
type Option func(*options)

type options struct {
	withValidations bool
	extensionMapper ExtensionMapper
}

func applyOptionsWithDefaults(opts []Option) options {
	o := options{
		extensionMapper: ExtensionMapper(passThroughMapper),
	}

	for _, apply := range opts {
		apply(&o)
	}

	return o
}

// WithValidations instructs the [SchemaAnalyzer] to carry out further analysis on validations.
func WithValidations(enabled bool) Option {
	return func(o *options) {
		o.withValidations = enabled
	}
}

// WithExtensionMapper equips the analyzer with a mapper for extensions.
//
// The mapper may be used to validate extensions.
func WithExtensionMapper(mapper ExtensionMapper) Option {
	return func(o *options) {
		o.extensionMapper = mapper
	}
}
