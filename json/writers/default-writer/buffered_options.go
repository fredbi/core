package writer

const (
	defaultBufferSize = 4096
	minBufferSize     = 6
)

// BufferedOption configures the [Buffered] writer.
type BufferedOption func(*bufferedOptions)

// WithBufferSize specifies the size in bytes of the write buffer.
//
// The default is 4096.
//
// The minimum value is 6 bytes.
func WithBufferSize(size int) BufferedOption {
	return func(o *bufferedOptions) {
		o.bufferSize = max(size, minBufferSize)
	}
}

// bufferedOptions carries the [Buffered] configuration plus the redeem handle for the
// working buffer. The working buffer itself lives directly on [buffered] (not here behind
// a pointer) so the hot path avoids an extra indirection — see [buffered.borrowBuffer].
type bufferedOptions struct {
	bufferSize   int
	redeemBuffer func()
}

func (o *bufferedOptions) Reset() {
	o.bufferSize = defaultBufferSize
	o.redeemBuffer = nil
}

func (o *bufferedOptions) redeem() {
	if o.redeemBuffer != nil {
		o.redeemBuffer()
		o.redeemBuffer = nil
	}
}

func bufferedOptionsWithDefaults(opts []BufferedOption) *bufferedOptions {
	o := poolOfBufferedOptions.Borrow()

	for _, apply := range opts {
		apply(o)
	}

	if o.bufferSize <= 0 {
		o.bufferSize = defaultBufferSize
	}

	return o
}
