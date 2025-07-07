package store

import (
	"slices"

	"github.com/fredbi/core/swag/pools"
)

const (
	defaultMinArenaSize      = 4096
	defaultEnableCompression = true
)

// Option alters the default settings of a store ([Store], [ConcurrentStore] or [VerbatimStore]).
type Option = func(*options)

type options struct {
	compressionOptions
	enableCompression  bool
	minArenaSize       int
	bytesFactory       func() []byte
	pooledBytesFactory func() *pools.Slice[byte]
}

func applyOptionsWithDefaults(opts []Option) options {
	o := options{
		enableCompression: defaultEnableCompression,
		minArenaSize:      defaultMinArenaSize,
	}

	for _, apply := range opts {
		apply(&o)
	}

	if o.enableCompression && !o.isCompressionInitialized() {
		o.compressionOptions = applyCompressionOptionsWithDefaults(nil)
	}

	return o
}

// WithArenaSize sets the initial capacity of the inner arena that stores large values.
func WithArenaSize(size int) Option {
	return func(o *options) {
		o.minArenaSize = size
	}
}

// WithEnableCompression enables compression of long strings in the [Store].
//
// Compression is enabled by default and uses the DEFLATE compression method implemented
// by the standard library package [compress/flate].
//
// By default, compression kicks in for strings longer than 128 bytes.
//
// The default compression level is [flate.DefaultCompression), which corresponds to a compression level of 6.
//
// Compression may be disabled or altered using [WithCompressionOptions] with some [CompressionOption] s.
func WithEnableCompression(enabled bool) Option {
	return func(o *options) {
		o.enableCompression = enabled
	}
}

func WithCompressionOptions(opts ...CompressionOption) Option {
	return func(o *options) {
		o.enableCompression = true
		o.compressionOptions = applyCompressionOptionsWithDefaults(opts)
	}
}

// WithBytesFactory affects how Get allocates the returned buffer.
func WithBytesFactory(bytesFactory func() []byte) Option {
	return func(o *options) {
		o.bytesFactory = bytesFactory
	}
}

func WithPooledBytesFactory(pooledBytesFactory func() *pools.Slice[byte]) Option {
	return func(o *options) {
		o.pooledBytesFactory = pooledBytesFactory
	}
}

// TODO: SetWriter method to clone existing store with a writer?

func (o *options) Reset() {
	if o.enableCompression {
		o.compressionOptions.Reset()
	}
}

func (o *options) getBuffer(capacity int) []byte {
	switch {
	case o.pooledBytesFactory != nil: // preferred, because the pools.Slice may recycle capacity from previous growths
		buffer := o.pooledBytesFactory()
		buffer.Grow(capacity)

		return buffer.Slice()

	case o.bytesFactory != nil:
		buffer := o.bytesFactory()
		buffer = slices.Grow(buffer, capacity)

		return buffer[:0]

	default:
		return make([]byte, 0, capacity)
	}
}
