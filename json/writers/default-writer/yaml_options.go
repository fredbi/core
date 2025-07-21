package writer

var (
	defaultYAMLIndent = []byte("  ")               //nolint:gochecknoglobals
	yamlElementPrefix = []byte{yamlElement, space} //nolint:gochecknoglobals
)

type YAMLOption func(*yamlOptions)

func WithYAMLIndent(indent string) YAMLOption {
	return func(o *yamlOptions) {
		o.indent = []byte(indent)
	}
}

func WithYAMLDocHeading(enabled bool) YAMLOption {
	return func(o *yamlOptions) {
		o.withDocHeader = enabled
	}
}

func WithYAMLBufferedOptions(opts ...BufferedOption) YAMLOption {
	return func(o *yamlOptions) {
		o.applyBufferedOptions = opts
	}
}

type yamlOptions struct {
	indent               []byte
	applyBufferedOptions []BufferedOption
	withDocHeader        bool
}

func (o *yamlOptions) Reset() {
	o.indent = defaultIndent
	o.applyBufferedOptions = nil
	o.withDocHeader = false
}

func (o *yamlOptions) redeem() {
	// no inner resources to relinquish
}

func yamlOptionsWithDefaults(opts []YAMLOption) *yamlOptions {
	o := poolOfYAMLOptions.Borrow()

	for _, apply := range opts {
		apply(o)
	}

	if len(o.indent) == 0 {
		o.indent = defaultYAMLIndent
	}

	return o
}
