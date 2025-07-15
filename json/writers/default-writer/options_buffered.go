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

type bufferedOptions struct {
	bufferSize   int
	buffer       []byte
	redeemBuffer func()
}

func (o *bufferedOptions) Reset() {
	o.bufferSize = defaultBufferSize
	o.buffer = o.buffer[:0]
	o.redeemBuffer = nil
}

func (o *bufferedOptions) redeem() {
	if o.redeemBuffer != nil {
		o.redeemBuffer()
		o.redeemBuffer = nil
	}
}

func (o *bufferedOptions) updateOptions(opts []BufferedOption) {
	newOptions := poolOfBufferedOptions.Borrow()
	for _, apply := range opts {
		apply(newOptions)
	}

	if newOptions.bufferSize == o.bufferSize {
		// no change needed
		poolOfBufferedOptions.Redeem(newOptions)

		return
	}

	// replace the internal buffer
	o.redeemBuffer()
	bufHolder, redeem := poolOfBuffers.BorrowWithSizeAndRedeem(newOptions.bufferSize)
	o.buffer = bufHolder.Slice()
	o.buffer = o.buffer[:0:o.bufferSize] // clip to the desired capacity, since the pool might provide higher capacity than requested
	o.redeemBuffer = redeem
	poolOfBufferedOptions.Redeem(newOptions)
}

func bufferedOptionsWithDefaults(opts []BufferedOption) *bufferedOptions {
	o := poolOfBufferedOptions.Borrow()

	for _, apply := range opts {
		apply(o)
	}

	if o.bufferSize <= 0 {
		o.bufferSize = defaultBufferSize
	}

	bufHolder, redeem := poolOfBuffers.BorrowWithSizeAndRedeem(o.bufferSize)
	o.buffer = bufHolder.Slice()
	o.buffer = o.buffer[:0:o.bufferSize] // clip to the desired capacity, since the pool might provide higher capacity
	o.redeemBuffer = redeem

	return o
}
