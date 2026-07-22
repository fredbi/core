package lexer

import (
	"io"
	"slices"

	"github.com/fredbi/core/json/lexers/default-lexer/internal/input"
	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/lexers/token"
)

// L is a lexer for JSON.
//
// It splits JSON input into tokens [token.T] using [L.NextToken].
//
// The lexer may operate from a stream of bytes (consuming from an [io.Reader]) or from a provided buffer of bytes.
// Whenever consuming from an [io.Reader], the stream is buffered so it is not useful to use a buffered reader
// in front of the lexer.
//
// The lexer is almost a parser: it performs JSON grammar checks such as opening/closing brackets,
// expected sequence of delimiter/values etc.
//
// If you plan to allocate many lexers with a short life span, consider using the global pool
// with the [BorrowLexerWithBytes] / [BorrowLexerWithReader] functions and the
// returned redeem closure.
type L struct {
	in input.Input // buffered-input + scan-cursor state (see [Input])

	errContext   *codes.ErrContext
	nestingLevel []uint64 // the stack of nested containers. Every bit represent an extra nesting. Capped if maxContainerStack > 0
	blanks       []byte   // preceding whitespace run, accumulated by the unified core when trackBlanks is set (verbatim lexer)

	current token.T

	line      int    // current line, 1-based
	lineStart uint64 // offset of the first byte of the current line
	tokLine   int    // line of the most recent token's start
	tokCol    int    // column of the most recent token's start, 1-based

	isAtEOF     bool
	trackBlanks bool // verbatim mode: the cores read this (hot whitespace-skip path); mirrored to in.TrackBlanks for consumeString's dispatch. Set by VL setup.

	options
}

// New JSON lexer consuming from an [io.Reader].
//
// The lexer performs some internal buffering on a fixed size buffer to call the reader on chunks.
//
// Use option [WithBufferSize] to alter the size of this buffer (the default is 4KB).
//
// If you plan to allocate many lexers with a short life span, consider using the global pool
// with the [BorrowLexerWithBytes] / [BorrowLexerWithReader] functions and the
// returned redeem closure.
func New(r io.Reader, opts ...Option) *L {
	l := new(L)
	l.applyWithDefaults(opts)
	l.in.R = r
	l.in.Buffer = make([]byte, l.bufferSize)
	l.in.Bufferized = 0
	l.in.WholeBuffer = false  // streaming: the buffer is refilled, values must be copied
	l.in.NeedFirstFill = true // §10.5f: the initial read + whole-buffer short-circuit is pending

	l.reset()

	return l
}

// NewWithBytes yields a new JSON lexer consuming from a provided fixed buffer of bytes.
//
// Since the full buffer is provided by the caller, there is no additional internal buffering.
//
// If you plan to allocate many lexers with a short life span, consider using the global pool
// with the [BorrowLexerWithBytes] function and the returned redeem closure.
func NewWithBytes(data []byte, opts ...Option) *L {
	l := new(L)
	l.applyWithDefaults(opts)
	l.in.R = noopReader
	l.in.Buffer = data
	l.in.Bufferized = len(data)
	l.in.PreviousBuffer = nil
	l.keepPreviousBuffer = 0 // disabled option
	l.in.WholeBuffer = true     // the whole input is in the buffer: values may alias it
	l.in.NeedFirstFill = false

	l.reset()

	return l
}

// Ok yields the error status of the lexer.
//
// True means that no error has occurred so far.
func (l *L) Ok() bool {
	return l.in.Err == nil
}

// Err returns an error that happened during lexing.
func (l *L) Err() error {
	return l.in.Err
}

// ErrInContext returns any error that happened during lexing, with the error context.
func (l *L) ErrInContext() *codes.ErrContext {
	if l.in.Err == nil {
		return nil
	}

	if l.errContext == nil {
		l.setErrContext()
	}

	return l.errContext
}

// Offset yields the number of bytes consumed by the lexer so far.
func (l *L) Offset() uint64 {
	return l.in.Offset
}

// The semantic lexer deliberately does NOT expose line/column. Tracking them is
// costly on whitespace-heavy input (it forces newline counting in the whitespace
// skip — measured ~-28% on citm; see ramblings/2026-06-linecol-cost-decomposition),
// and it cannot be computed lazily either, because a streaming buffer discards past
// bytes. Position is a verbatim-lexer concern: use [VL] / [token.VT] (VL.Line /
// VL.Column, or VT.Line / VT.Col), which carry it. The byte position is always
// available via [L.Offset].

// NextToken returns the next JSON token consumed from the stream or slice of bytes.
//
// The last token is of Kind EOF.
//
// If the lexer is in an errored status, it will keep responding tokens with Kind Unknown.
//
// By default the structural separators "," and ":" are validated but not emitted
// (see [WithElideSeparator]); pass WithElideSeparator(false) to receive them.
//
// Tokens are expected to have a short lifespan: when NextToken is
// called again, the memory allocated to support the value of the
// previously returned token is reused for the next token.
//
// If you want to keep tokens for later reuse, you may clone a token
// using its [T.Clone] method.
func (l *L) NextToken() token.T {
	// devirtualized pull core (adopted 2026-06-27: +4.23% geomean over the generic
	// core, no regressions; see plan §5.1 + devirt_bench_test). Dispatch once per
	// token on wholeBuffer (§10): the whole-buffer lane (local cursor, no readMore,
	// zero-copy blanks) is the frozen champion; the stream lane is optimized
	// separately. The generic scanTokenBufferG / scanTokenStreamG are lexgen's
	// source-of-truth and the A/B baseline.
	if l.in.WholeBuffer {
		return scanTokenBufferSemantic(l, semanticPolicy{})
	}
	if l.in.NeedFirstFill {
		l.in.FirstFill()
		if l.in.WholeBuffer {
			return scanTokenBufferSemantic(l, semanticPolicy{})
		}
	}

	return scanTokenStreamSemantic(l, semanticPolicy{})
}

// primeStream runs the pending first fill (see [input.Input.FirstFill]) if needed. Used by the
// push paths (Tokens), which must resolve the whole-buffer short-circuit before they
// choose the native push core vs the NextToken loop.
func (l *L) primeStream() {
	if l.in.NeedFirstFill {
		l.in.FirstFill()
	}
}

// Reset returns the lexer to a clean, source-less state so it can be recycled.
//
// It is the method the pool invokes on both Borrow and Redeem, so it must be
// idempotent and must drop every reference to caller-supplied memory: in
// whole-buffer mode l.in.Buffer aliases the caller's data, and l.in.R may hold the
// caller's reader. Leaving them in place would pin (and expose) user memory for
// as long as the recycled lexer sits in the pool.
//
// Reset does NOT rebind an input: call [L.ResetWithBytes] / [L.ResetWithReader]
// (or a Borrow*/New* constructor) to lex a new source. Configured options are
// preserved, and the streaming-owned buffer keeps its capacity for reuse.
func (l *L) Reset() {
	if l.in.WholeBuffer && l.in.R == noopReader {
		// whole-buffer via NewWithBytes/ResetWithBytes: l.in.Buffer aliases the caller's
		// data — drop it so the pool does not pin user memory. A STREAMING lexer that
		// promoted to whole-buffer (§10.5f) still OWNS its buffer (l.in.R is the caller's
		// reader, not noopReader), so keep its capacity for reuse — it was refilled,
		// not aliased.
		l.in.Buffer = nil
	}
	l.in.R = noopReader
	l.in.WholeBuffer = false
	l.in.NeedFirstFill = false // source-less until re-bound via ResetWith*
	l.in.Bufferized = 0
	l.in.PreviousBuffer = l.in.PreviousBuffer[:0]
	l.reset()
}

// ResetWithBytes rebinds the lexer to a new input buffer and resets all scanning
// state, so a single lexer can be reused across inputs with no allocation (the
// no-pool equivalent of [BorrowLexerWithBytes]). Configured options are kept.
//
// The lexer operates in whole-buffer mode over data: emitted token values may
// alias data, which must therefore stay stable until the lexer is done with it.
func (l *L) ResetWithBytes(data []byte) {
	l.in.R = noopReader
	l.in.Buffer = data
	l.in.Bufferized = len(data)
	l.in.PreviousBuffer = nil
	l.keepPreviousBuffer = 0 // disabled option
	l.in.WholeBuffer = true     // the whole input is in the buffer: values may alias it
	l.in.NeedFirstFill = false
	l.reset()
}

// ResetWithReader rebinds the lexer to a new reader and resets all scanning
// state, so a single lexer can be reused across inputs (the no-pool equivalent
// of [BorrowLexerWithReader]). Configured options are kept and the internal
// buffer is reused.
func (l *L) ResetWithReader(r io.Reader) {
	l.in.R = r
	l.in.Bufferized = 0
	l.in.WholeBuffer = false  // streaming: the buffer is refilled, values must be copied
	l.in.NeedFirstFill = true // §10.5f: the initial read + whole-buffer short-circuit is pending
	l.reset()

	if cap(l.in.Buffer) < l.bufferSize {
		l.in.Buffer = slices.Grow(l.in.Buffer, l.bufferSize-cap(l.in.Buffer))[:l.bufferSize]
	}

	if l.keepPreviousBuffer > 0 && cap(l.in.PreviousBuffer) < l.keepPreviousBuffer {
		l.in.PreviousBuffer = slices.Grow(l.in.PreviousBuffer, l.keepPreviousBuffer-cap(l.in.PreviousBuffer))
	}
}

func (l *L) reset() {
	l.in.Err = nil
	l.in.MaxValueBytes = l.maxValueBytes
	l.in.KeepPreviousBuffer = l.keepPreviousBuffer
	l.in.NoAVX2 = l.noAVX2
	l.current = token.None
	l.in.Offset = 0
	l.in.Consumed = 0
	l.errContext = nil
	l.in.ExpectKey = false
	l.in.AfterKey = false
	l.isAtEOF = false
	l.line = 1
	l.lineStart = 0
	l.tokLine = 0
	l.tokCol = 0
	l.in.CurrentValue = l.in.CurrentValue[:0] // TODO: possibly preallocate value buffer to some configurable size

	if l.nestingLevel != nil {
		l.nestingLevel = l.nestingLevel[:1]
	} else {
		l.nestingLevel = make([]uint64, 1) // TODO: preallocate more if maxContainerStack > 63
	}
	l.nestingLevel[0] = 1 // the initial value for the stack must be 1: this bit is thereafter shifted right or left
}

func (l *L) SetErr(err error) {
	l.in.Err = err
}

// setErrContext populates the error context if an error has been detected.
func (l *L) setErrContext() {
	if l.in.Err == nil {
		return
	}

	var (
		window string
		pos    int
	)

	if l.keepPreviousBuffer > 0 && l.in.PreviousBuffer != nil && l.in.Consumed < l.keepPreviousBuffer {
		// add the copied previousBuffer if the error came early in the main buffer
		window = string(l.in.PreviousBuffer[0:len(l.in.PreviousBuffer)]) + string(l.in.Buffer[:l.in.Bufferized])
		pos = l.in.Consumed + len(l.in.PreviousBuffer)
	} else {
		window = string(l.in.Buffer[:l.in.Bufferized])
		pos = l.in.Consumed
	}

	l.errContext = &codes.ErrContext{
		Err:      l.in.Err,
		Offset:   l.in.Offset,
		Buffer:   window,
		Position: pos,
	}
}

