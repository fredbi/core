package lexer

import (
	"io"

	"github.com/fredbi/core/json/lexers/token"
)

// VL is a verbatim lexer for JSON.
//
// It splits JSON input into tokens using NextToken().
//
// The lexer may operate from a stream of bytes (consuming from an io.Reader) or from a provided []byte buffer.
// Whenever consuming from an io.Reader, the stream is buffered so it is not useful to use a buffered reader.
//
// The lexer performs JSON grammar check such as opening/closing brackets, expected sequence of delimiter/values etc.
type VL struct {
	*L
}

// NewVerbatim yields a new JSON verbatim lexer consuming from an io.Reader.
//
// The lexer performs some internal buffering on a fixed size buffer to call the reader on chunks.
//
// Use option WithBufferSize to alter the size of this buffer (defaults to 4KB).
//
// If you plan to allocate many lexers with a short life span, consider using the global pool
// with the BorrowLexerWithBytes() / BorrowLexerWithReader() functions and the
// returned redeem closure.
func NewVerbatim(r io.Reader, opts ...Option) *VL {
	l := new(VL)
	l.L = New(r, opts...)
	l.reset()

	return l
}

// NeVerbatimWithBytes yields a new verbatim JSON lexer consuming from a provided fixed []bytes buffer.
//
// Since the full buffer is provided by the caller, there is no additional internal buffering.
//
// If you plan to allocate many lexers with a short life span, consider using the global pool
// with the BorrowLexerWithBytes() function and the returned redeem closure.
func NewVerbatimWithBytes(data []byte, opts ...Option) *VL {
	l := new(VL)
	l.L = NewWithBytes(data, opts...)
	l.reset()

	return l
}

// NextToken returns the next verbatim token consumed from the stream or slice of
// bytes. The last token is of Kind EOF; in an errored state it keeps returning
// tokens of Kind Unknown.
//
// It is driven by the unified generic pull core (verbatim policy) — the same
// core that backs L.NextToken. Values are decoded by L's scanners (so VL gets
// L's fast paths and its \u-decoding fix) and the preceding blanks + position
// are attached by the policy.
//
// Tokens are expected to have a short lifespan: when NextToken is called again,
// the memory backing the previous token's value is reused. To keep a token, use
// its Clone() method.
func (l *VL) NextToken() token.VT {
	return scanTokenG[token.VT, verbatimPolicy](l.L, verbatimPolicy{})
}

// Reset returns the verbatim lexer to a clean, source-less state for reuse,
// scrubbing the embedded L (which drops references to caller-supplied memory)
// and re-arming the verbatim-specific blanks tracking. See [L.Reset].
func (l *VL) Reset() {
	l.L.Reset()
	l.reset()
}

// ResetWithBytes rebinds the verbatim lexer to a new input buffer and resets all
// scanning state, so a single lexer can be reused across inputs with no
// allocation. See [L.ResetWithBytes].
func (l *VL) ResetWithBytes(data []byte) {
	l.L.ResetWithBytes(data)
	l.reset()
}

// ResetWithReader rebinds the verbatim lexer to a new reader and resets all
// scanning state, so a single lexer can be reused across inputs. See
// [L.ResetWithReader].
func (l *VL) ResetWithReader(r io.Reader) {
	l.L.ResetWithReader(r)
	l.reset()
}

func (l *VL) reset() {
	l.blanks = l.blanks[:0]
	// the verbatim lexer never elides separators, and the unified core
	// accumulates the preceding blanks for it; both are read off the embedded L.
	l.elideSeparator = false
	l.trackBlanks = true
}
