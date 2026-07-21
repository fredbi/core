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
		elideSeparator     bool
		noAVX2             bool
	}
)

const defaultBufferBytes = 4096

// bufferSizeAlignment is the granularity to which a caller-supplied buffer size is
// rounded up. It covers one full AVX2 stride (32 bytes) — which also subsumes the
// 8-byte SWAR word — so the streaming window is never smaller than a single vector
// or SWAR step. This keeps the in-window fast paths (string/number stop scans) on
// their vectorized/word footing for any WithBufferSize, and floors out the
// pathological tiny-window cases (a window narrower than a token separator) that
// would otherwise stress the byte-by-byte refill seam.
const bufferSizeAlignment = 32

// alignBufferSize rounds size up to the next multiple of [bufferSizeAlignment].
func alignBufferSize(size int) int {
	return (size + bufferSizeAlignment - 1) &^ (bufferSizeAlignment - 1)
}

var defaultOptions = options{
	strictNumbers:  true,
	bufferSize:     defaultBufferBytes,
	elideSeparator: true,
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

// WithElideSeparator controls whether the structural separators "," and ":"
// are emitted as tokens by the semantic lexer [L].
//
// When enabled (the default for [L]), these separators are validated against the
// JSON grammar but not surfaced by [L.NextToken] / [L.Tokens]: the token stream
// carries only values and the container delimiters "{", "}", "[", "]". This
// matches the behavior of the standard encoding/json/jsontext lexer and yields
// a simpler walk — the structure is unambiguous from the value/Key tokens and
// the container delimiters.
//
// Disable it (WithElideSeparator(false)) to receive every separator token.
//
// The verbatim lexer [VL] honors this option too, but defaults to NOT eliding
// (elide-off) so its stream stays faithful and round-trippable. Pass
// WithElideSeparator(true) to a verbatim constructor to drop "," and ":" there
// as well; it is up to the caller to decide.
func WithElideSeparator(enabled bool) Option {
	return func(o *options) {
		o.elideSeparator = enabled
	}
}

// WithoutAVX2 disables the AVX2-accelerated long-string scan, keeping the lexer on
// the portable 8-byte SWAR path everywhere: once set, a long string value is
// scanned by the inline SWAR word loop instead of being handed to the vector
// kernel (see internal/strscan).
//
// The AVX2 gate is on by default on amd64 CPUs that support it: once a string
// value stays clean past a short inline probe it is scanned 32 bytes at a time,
// which is a large win on long values (descriptions, documentation, text payloads)
// and neutral on short ones. On non-amd64, or CPUs without AVX2, the SWAR path is
// already used and this option is a no-op.
//
// Most callers should leave the gate on. Reach for WithoutAVX2 when you want
// deterministic, CPU-independent behavior (identical code path regardless of the
// host's vector support — useful for reproducibility or differential testing), or
// if profiling a specific workload shows the vector path is not paying off. It does
// not change the token stream, only how the scan is performed.
func WithoutAVX2(disabled bool) Option {
	return func(o *options) {
		o.noAVX2 = disabled
	}
}

// WithBufferSize specifies the size in bytes of the internal buffer used
// by the lexer.
//
// The default is 4kB.
//
// The requested size is rounded up to the next multiple of 32 bytes (one AVX2
// stride, which also subsumes the 8-byte SWAR word). This guarantees the streaming
// window is always at least one vector step wide, keeping the in-window string and
// number fast paths on their vectorized footing and avoiding pathologically small
// windows. A size <= 0 is ignored and the default is kept.
func WithBufferSize(size int) Option {
	return func(o *options) {
		if size > 0 {
			o.bufferSize = alignBufferSize(size)
		}
	}
}

// WithMaxContainerStack sets a circuit breaker on the maximum
// level of nested containers.
//
// This avoids edge cases when memory is exhausted by faulty inputs
// (e.g. a nasty stream pushes an infinite sequence of "{" or "[").
//
// The lexer allocates 8 bytes (a uint64) every additional 63 nested levels.
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
// (e.g. a nasty stream pushes an infinite sequence after an opening double quote).
//
// For the verbatim lexer, this also bounds the buffer that accumulates a run of
// non-significant whitespace, so a flood of blanks between tokens cannot exhaust
// memory either.
//
// NOTE: this option is primarily intended to secure the lexing of JSON streams,
// and should not be needed for lexers built with a full data buffer.
//
// It does NOT bound the total input size: to limit how many bytes are consumed
// from an [io.Reader], wrap it with [io.LimitReader] (see the package overview).
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
