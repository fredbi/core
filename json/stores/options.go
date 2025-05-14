package stores

// Option alters the default settings exposed via the [Store] interface (Get).
type Option func(*Options)

// Options for [Store] methods.
type Options struct {
	buffer []byte
}

// Buffer returns the []byte buffer provided by the caller.
func (o Options) Buffer() []byte {
	return o.buffer
}

// Apply [Option] s.
func (o *Options) Apply(opts []Option) {
	for _, apply := range opts {
		apply(o)
	}
}

// Reset options to defaults.
func (o *Options) Reset() {
	o.buffer = nil
}

// WithBuffer allows the caller of Get to keep ownership of internally allocated buffers,
// e.g. when the [Store] is restoring a [Value] to the caller.
//
// This is useful whenever you don't want to keep the returned [Value] and prefer to recycle memory
// rather than to allocate a new buffer.
func WithBuffer(buffer []byte) Option {
	return func(o *Options) {
		o.buffer = buffer
	}
}
