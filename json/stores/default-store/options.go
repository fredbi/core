package store

const (
	defaultMinArenaSize      = 4096
	defaultEnableCompression = true
)

// Options configures a store ([Store], [ConcurrentStore] or [VerbatimStore]).
//
// It is an immutable value built by chaining With* methods from [DefaultOptions]. Each method copies
// the receiver, sets one field and returns the copy, so a builder chain stays on the stack and
// allocates nothing:
//
//	o := store.DefaultOptions().
//		WithCompressionLevel(9).
//		WithCompressionDict(trainedDict).
//		WithArenaSize(8192)
//	s := store.New(o)
//
// Pass the result to [New], [NewConcurrent], [NewVerbatim] or [BorrowStore]. Those constructors are
// variadic for convenience — calling them with no Options yields the defaults — but only the last
// Options is used; compose configuration through the builder chain, not by passing several Options.
type Options struct {
	resolved options
}

// options is the resolved configuration embedded in a store.
type options struct {
	compressionOptions

	enableCompression bool
	minArenaSize      int
}

// DefaultOptions returns the default [Options], the seed for a builder chain.
func DefaultOptions() Options {
	return Options{
		resolved: options{
			enableCompression: defaultEnableCompression,
			minArenaSize:      defaultMinArenaSize,
			compressionOptions: compressionOptions{
				compressionThreshold: defaultCompressionThreshold,
				compressionLevel:     defaultCompressionLevel,
				// dict is nil; cw is built lazily on first compression.
			},
		},
	}
}

// WithArenaSize sets the initial capacity of the inner arena that stores large values.
func (o Options) WithArenaSize(size int) Options {
	o.resolved.minArenaSize = size

	return o
}

// WithEnableCompression enables or disables compression of long strings in the store.
//
// Compression is enabled by default and uses the DEFLATE method from the standard library package
// [compress/flate]. By default it kicks in for strings longer than 128 bytes, at level
// [flate.DefaultCompression] (level 6). Tune it with [Options.WithCompressionLevel],
// [Options.WithCompressionThreshold] and [Options.WithCompressionDict].
//
// A store with compression disabled never allocates a DEFLATE writer (see
// [compressionOptions.compressWriter]) and stores long strings verbatim in the arena.
func (o Options) WithEnableCompression(enabled bool) Options {
	o.resolved.enableCompression = enabled

	return o
}

// resolveOptions picks the effective configuration from a variadic builder list: the last Options
// wins, and an empty list yields the defaults.
func resolveOptions(opts []Options) options {
	if len(opts) == 0 {
		return DefaultOptions().resolved
	}

	return opts[len(opts)-1].resolved
}

// getBuffer allocates a fresh buffer for a value decoded by [Store.Get].
//
// It is always a fresh allocation: the returned buffer is aliased by the [values.Value] that Get
// hands back, which the caller may keep, share and read concurrently. For a zero-allocation,
// caller-owned alternative for transient values, see [Store.AppendValueBytes].
func (o *options) getBuffer(capacity int) []byte {
	return make([]byte, 0, capacity)
}
