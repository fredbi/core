package structural

// Option customizes the behavior of the [SchemaAnalyzer].
type Option func(*options)

type options struct {
	withValidations  bool
	extensionMappers []ExtensionMapper
}

func applyOptionsWithDefaults(opts []Option) options {
	o := options{}

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

// WithExtensionMappers adds chained mappers to the analyzer, so that it may validate and convert extensions.
func WithExtensionMappers(mappers ...ExtensionMapper) Option {
	return func(o *options) {
		o.extensionMappers = append(o.extensionMappers, mappers...)
	}
}
