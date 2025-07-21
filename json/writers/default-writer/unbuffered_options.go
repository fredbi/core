package writer

// UnbufferedOption configures the [Unbuffered] writer.
type UnbufferedOption func(*unbufferedOptions)

type unbufferedOptions struct{}

// Reset options to default and reset the inner [Buffer].
func (o *unbufferedOptions) Reset() {
}

func (o *unbufferedOptions) redeem() {
}

func unbufferedOptionsWithDefaults(opts []UnbufferedOption) *unbufferedOptions {
	o := poolOfUnbufferedOptions.Borrow()

	for _, apply := range opts {
		apply(o)
	}

	return o
}
