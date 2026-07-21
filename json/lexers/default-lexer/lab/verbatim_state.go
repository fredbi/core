package lab

import (
	"io"
	"iter"

	"github.com/fredbi/core/json/lexers/token"
)

// VS is a PROTOTYPE state-based verbatim lexer (§10.5b — the token-vs-state
// arbitrage).
//
// It produces the SAME token stream as the semantic lexer [L] — light [token.T]
// values — but, like the verbatim lexer [VL], it preserves the "verbatim feature"
// so the input can be reconstructed faithfully:
//
//   - string and number values are kept RAW (escapes intact, no decoding) — this
//     comes for free from the trackBlanks branch in consumeString, exactly as VL;
//   - the whitespace run preceding each token is exposed via [VS.LeadingSpace];
//   - the 1-based source position is exposed via [VS.Line] / [VS.Column].
//
// The difference from [VL] is WHERE that information lives: VL bakes blanks and
// position INTO each token (the 72B [token.VT]), paying a per-token
// construct-and-return-by-value cost that the §10.5a sizing showed to be 84–100%
// of VL's throughput tax (VL/pull ran at 27% of L/pull). VS instead keeps the
// blanks and position as LEXER STATE, read back through accessors valid until the
// next NextToken — so the emitted token stays the 32B [token.T] and the pull path
// runs at ~85–93% of L. The trade is that the accessors reflect only the
// most-recently-returned token, matching the existing short-lifespan token contract.
//
// This is a lab prototype to validate the design end-to-end (throughput + faithful
// round-trip) before deciding whether to reshape the shipped verbatim API. It reuses
// the same generic scan cores as L and VL via the statePolicy monomorphization
// (scan_gen.go).
type VS struct {
	*L
}

// NewVerbatimState yields a new state-based verbatim lexer consuming from an
// io.Reader. Separators are not elided by default (a faithful, round-trippable
// stream); pass WithElideSeparator(true) to drop them. See [NewVerbatim].
func NewVerbatimState(r io.Reader, opts ...Option) *VS {
	l := new(VS)
	l.L = New(r, verbatimOpts(opts)...)
	l.reset()

	return l
}

// NewVerbatimStateWithBytes yields a new state-based verbatim lexer over a provided
// []byte buffer. See [NewVerbatimWithBytes].
func NewVerbatimStateWithBytes(data []byte, opts ...Option) *VS {
	l := new(VS)
	l.L = NewWithBytes(data, verbatimOpts(opts)...)
	l.reset()

	return l
}

// NextToken returns the next token consumed from the input as a light [token.T]
// (string/number values kept raw). The preceding blanks and the token position are
// available via [VS.LeadingSpace] / [VS.Line] / [VS.Column] until the next call.
//
// The last token is of Kind EOF; in an errored state it keeps returning tokens of
// Kind Unknown. Tokens have a short lifespan: the backing memory (and the lexer
// state the accessors read) is reused on the next call.
func (l *VS) NextToken() token.T {
	// dispatch once per token on wholeBuffer (§10): whole-buffer lane vs stream lane,
	// both monomorphized onto statePolicy (emits token.T, stashes blanks/position).
	if l.wholeBuffer {
		return scanTokenBufferState(l.L, statePolicy{})
	}
	if l.needFirstFill {
		l.firstFill() // §10.5f: promote to whole-buffer if the input fits
		if l.wholeBuffer {
			return scanTokenBufferState(l.L, statePolicy{})
		}
	}

	return scanTokenStreamState(l.L, statePolicy{})
}

// Tokens returns an iterator over the tokens, the state-based counterpart of
// [L.Tokens] / [VL.Tokens]. In whole-buffer mode it takes the native push path; the
// accessors ([VS.LeadingSpace] / [VS.Line] / [VS.Column]) are valid inside each
// iteration of the range, before the next token is produced.
func (l *VS) Tokens() iter.Seq[token.T] {
	return func(yield func(token.T) bool) {
		l.primeStream() // §10.5f: resolve the whole-buffer short-circuit before choosing the core
		if l.wholeBuffer && l.maxValueBytes == 0 {
			l.scanPushStateDevirt(yield)

			return
		}

		for {
			tok := l.NextToken()
			if l.err != nil {
				return
			}
			if tok.Kind() == token.EOF {
				return
			}
			if !yield(tok) {
				return
			}
		}
	}
}

// LeadingSpace returns the run of insignificant whitespace that preceded the
// most-recently-returned token (empty if none), sliced zero-copy from the input.
// It is the state-based equivalent of [token.VT.Blanks]; valid until the next
// NextToken. For the EOF token it is the trailing whitespace of the document.
func (l *VS) LeadingSpace() []byte { return l.blanks }

// Line yields the 1-based line number at which the most-recently-returned token
// starts (0 before the first token). Mirrors [VL.Line].
func (l *VS) Line() int { return l.tokLine }

// Column yields the 1-based column at which the most-recently-returned token starts
// (0 before the first token). Mirrors [VL.Column].
func (l *VS) Column() int { return l.tokCol }

// Reset returns the lexer to a clean, source-less state for reuse. See [VL.Reset].
func (l *VS) Reset() {
	l.L.Reset()
	l.reset()
}

// ResetWithBytes rebinds the lexer to a new input buffer and resets all scanning
// state. See [VL.ResetWithBytes].
func (l *VS) ResetWithBytes(data []byte) {
	l.L.ResetWithBytes(data)
	l.reset()
}

// ResetWithReader rebinds the lexer to a new reader and resets all scanning state.
// See [VL.ResetWithReader].
func (l *VS) ResetWithReader(r io.Reader) {
	l.L.ResetWithReader(r)
	l.reset()
}

func (l *VS) reset() {
	l.blanks = l.blanks[:0]
	// trackBlanks routes consumeString to the RAW (validate-not-decode) scanners so
	// string values round-trip, and drives the stream core's per-refill blank
	// accumulation; the buffer/push cores stash the zero-copy blanks via storesBlanks.
	// elideSeparator is seeded at construction (verbatimOpts) and preserved across
	// L.Reset*; do NOT clobber it here.
	l.trackBlanks = true
}
