package writer

var defaultIndent = []byte("  ") //nolint:gochecknoglobals

type IndentedOption func(*indentedOptions)

func WithIndent(indent string) IndentedOption {
	return func(o *indentedOptions) {
		o.indent = []byte(indent)
	}
}

func WithIndentBufferedOptions(opts ...BufferedOption) IndentedOption {
	return func(o *indentedOptions) {
		o.applyBufferedOptions = opts
	}
}

type indentedOptions struct {
	indent               []byte
	applyBufferedOptions []BufferedOption
}

func (o *indentedOptions) Reset() {
	o.indent = defaultIndent
	o.applyBufferedOptions = nil
}

func (o *indentedOptions) redeem() {
	// no inner resources to relinquish
}

func indentedOptionsWithDefaults(opts []IndentedOption) *indentedOptions {
	o := poolOfIndentedOptions.Borrow()

	for _, apply := range opts {
		apply(o)
	}

	if len(o.indent) == 0 {
		o.indent = defaultIndent
	}

	return o
}
