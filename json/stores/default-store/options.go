package store

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
}

func applyOptionsWithDefaults(opts []Option) options {
	o := options{
		enableCompression: defaultEnableCompression,
		minArenaSize:      defaultMinArenaSize,
		// Compression defaults are always applied: with a lazy writer and a nil default dictionary
		// this costs nothing (two ints), and whether compression actually happens is gated by
		// enableCompression on the write path (see Store.putString). A Store that never compresses
		// thus never allocates a flate writer.
		compressionOptions: applyCompressionOptionsWithDefaults(nil),
	}

	for _, apply := range opts {
		apply(&o)
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

// TODO: SetWriter method to clone existing store with a writer?

func (o *options) Reset() {
	if o.enableCompression {
		o.compressionOptions.Reset()
	}
}

// getBuffer allocates a fresh buffer for a value decoded by [Store.Get].
//
// It is always a fresh allocation: the returned buffer is aliased by the [values.Value] that Get
// hands back, which the caller may keep, share and read concurrently. For a zero-allocation,
// caller-owned alternative for transient values, see [Store.AppendValueBytes].
func (o *options) getBuffer(capacity int) []byte {
	return make([]byte, 0, capacity)
}
