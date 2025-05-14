package store

import "github.com/fredbi/core/json/writers"

const (
	defaultMinArenaSize      = 4096
	defaultEnableCompression = true
)

// Option alters the default settings of a store ([Store], [ConcurrentStore] or [VerbatimStore]).
type Option = func(*options)

type options struct {
	compressionOptions
	enableCompression bool
	minArenaSize      int
	writer            writers.Writer
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

// WithWriter enables the [Store.Write] methods, sending retrieved values directly to the configured [writers.Writer].
//
// Using [Store.Write] without this setting will panic.
func WithWriter(w writers.Writer) Option {
	return func(o *options) {
		o.writer = w
	}
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

func (o *options) Reset() {
	if o.enableCompression {
		o.compressionOptions.Reset()
	}
}
