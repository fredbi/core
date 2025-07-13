package writer

var defaultIndent = []byte("  ")

type IndentedOption func(*indentedOptions)

func WithIndent(indent string) IndentedOption {
	return func(o *indentedOptions) {
		o.indent = []byte(indent)
	}
}

func WithBufferedOptions(opts ...BufferedOption) IndentedOption {
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

	return o
}
