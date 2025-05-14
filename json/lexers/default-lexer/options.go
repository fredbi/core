package lexer

type (
	// Option for the lexer.
	Option func(*options)

	options struct {
		bufferSize         int
		maxContainerStack  int
		maxValueBytes      int
		keepPreviousBuffer int
		strictNumbers      bool
	}
)

const defaultBufferBytes = 4096

var defaultOptions = options{
	strictNumbers: true,
	bufferSize:    defaultBufferBytes,
}

func (o *options) applyWithDefaults(opts []Option) {
	*o = defaultOptions
	for _, apply := range opts {
		apply(o)
	}
}

// WithStrictNumber abides by the definition
// of numbers given by [RFC 8259](https://www.rfc-editor.org/rfc/rfc8259).
//
// This is true by default.
//
// Whenever disabled, a laxer parsing of numbers is carried on, allowing
// for leading '+' and zeroes.
func WithStrictNumber(enabled bool) Option {
	return func(o *options) {
		o.strictNumbers = enabled
	}
}

// WithBufferSize specifies the size in bytes of the internal buffer used
// by the lexer.
//
// The defaut is 4kB.
func WithBufferSize(size int) Option {
	return func(o *options) {
		if size > 0 {
			o.bufferSize = size
		}
	}
}

// WithMaxContainerStack sets a circuit breaker on the maximum
// level of nested containers.
//
// This avoids edge cases when memory is exhausted by faulty inputs
// (e.g. a nasty stream pushes an infinite sequence of "{" or "[").
//
// The lexer allocates 4 bytes (a uint64) every additional 63 nested levels.
//
// NOTE: this option is primarily intended to secure the lexing of JSON streams,
// and should not be needed for lexers built with a full data buffer.
//
// The default value is zero: there is no maximum and no circuit breaker enabled.
func WithMaxContainerStack(maxDepth int) Option {
	return func(o *options) {
		o.maxContainerStack = maxDepth
	}
}

// WithMaxValueBytes sets a circuit breaker on the maximum
// size of a string or number.
//
// This avoids edge cases when memory is exhausted by faulty inputs
// (e.g. a nasty stream pushes an infinite sequence after an opening double quote.
//
// NOTE: this option is primarily intended to secure the lexing of JSON streams,
// and should not be needed for lexers built with a full data buffer.
//
// The default value is zero: there is no maximum and no circuit breaker enabled.
func WithMaxValueBytes(size int) Option {
	return func(o *options) {
		o.maxValueBytes = size
	}
}

// WithEnsureErrorContext guarantees that all errors will
// get at least min(maxLength,bufferSize) bytes of context.
//
// This option maximizes the readability of the ErrorContext returned
// by ErrInContext(), at the cost of an extra copy of maxLength bytes
// every time an internal buffer is overturned.
//
// Without this, the lexer only remembers the current buffer, and
// the error context may be truncated for error that occur toward
// the beginning of the buffer.
//
// NOTE: this option is primarily intended to improve error readability on JSON streams,
// and is not needed by lexers built with a full data buffer.
//
// By default, the value is zero and there is no such buffer copy.
func WithEnsureErrorContext(maxLength int) Option {
	return func(o *options) {
		o.keepPreviousBuffer = maxLength
	}
}
