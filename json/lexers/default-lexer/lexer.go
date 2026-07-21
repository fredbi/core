package lexer

import (
	"bytes"
	"errors"
	"io"
	"slices"

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
	r          io.Reader
	err        error
	errContext *codes.ErrContext

	buffer         []byte   // determined by bufferSize
	currentValue   []byte   // capped if maxValueBytes > 0
	previousBuffer []byte   // used when keepPreviousBuffer=true
	nestingLevel   []uint64 // the stack of nested containers. Every bit represent an extra nesting. Capped if maxContainerStack > 0
	blanks         []byte   // preceding whitespace run, accumulated by the unified core when trackBlanks is set (verbatim lexer)

	current token.T

	offset     uint64
	consumed   int
	bufferized int

	line      int    // current line, 1-based
	lineStart uint64 // offset of the first byte of the current line
	tokLine   int    // line of the most recent token's start
	tokCol    int    // column of the most recent token's start, 1-based

	expectKey   bool
	afterKey    bool // the previous token was an object key: a ':' must follow
	isAtEOF       bool
	wholeBuffer   bool // the buffer holds the entire input (no refills): values may alias it
	needFirstFill bool // streaming: the initial read (and whole-buffer short-circuit) is pending
	trackBlanks   bool // accumulate preceding blanks into l.blanks (verbatim lexer)

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
	l.r = r
	l.buffer = make([]byte, l.bufferSize)
	l.bufferized = 0
	l.wholeBuffer = false  // streaming: the buffer is refilled, values must be copied
	l.needFirstFill = true // §10.5f: the initial read + whole-buffer short-circuit is pending

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
	l.r = noopReader
	l.buffer = data
	l.bufferized = len(data)
	l.previousBuffer = nil
	l.keepPreviousBuffer = 0 // disabled option
	l.wholeBuffer = true     // the whole input is in the buffer: values may alias it
	l.needFirstFill = false

	l.reset()

	return l
}

// Ok yields the error status of the lexer.
//
// True means that no error has occurred so far.
func (l *L) Ok() bool {
	return l.err == nil
}

// Err returns an error that happened during lexing.
func (l *L) Err() error {
	return l.err
}

// ErrInContext returns any error that happened during lexing, with the error context.
func (l *L) ErrInContext() *codes.ErrContext {
	if l.err == nil {
		return nil
	}

	if l.errContext == nil {
		l.setErrContext()
	}

	return l.errContext
}

// Offset yields the number of bytes consumed by the lexer so far.
func (l *L) Offset() uint64 {
	return l.offset
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
	if l.wholeBuffer {
		return scanTokenBufferSemantic(l, semanticPolicy{})
	}
	if l.needFirstFill {
		l.firstFill()
		if l.wholeBuffer {
			return scanTokenBufferSemantic(l, semanticPolicy{})
		}
	}

	return scanTokenStreamSemantic(l, semanticPolicy{})
}

// firstFill performs the initial read for a streaming lexer and SHORT-CIRCUITS to
// whole-buffer mode when the entire input fits in the buffer (§10.5f). It reads into
// l.buffer until the buffer is full or the reader is exhausted; if EOF arrives before
// the buffer fills, the whole input is now buffered, so the lexer flips to wholeBuffer
// mode — thereafter the fast in-buffer cores run (and, for Tokens(), the native
// whole-buffer push core instead of the NextToken+closure fallthrough) with no
// per-token streaming overhead. Inputs larger than the buffer stay in streaming mode
// with the first window pre-filled. A non-EOF read error is recorded in l.err.
//
// It is done LAZILY (first NextToken/Tokens, gated by needFirstFill) rather than at
// construction, so a caller that reslices l.buffer to force a narrow window (tests)
// still gets the window it asked for. Runs exactly once per bound input.
func (l *L) firstFill() {
	l.needFirstFill = false

	n := 0
	for n < len(l.buffer) {
		m, err := l.r.Read(l.buffer[n:])
		n += m
		if err != nil {
			l.bufferized = n
			if errors.Is(err, io.EOF) {
				l.wholeBuffer = true // whole input fits: run the fast in-buffer cores
			} else {
				l.err = err
			}

			return
		}
		if m == 0 {
			break // reader returned (0, nil): don't spin — stay streaming
		}
	}
	l.bufferized = n
}

// primeStream runs the pending first fill (see [L.firstFill]) if needed. Used by the
// push paths (Tokens), which must resolve the whole-buffer short-circuit before they
// choose the native push core vs the NextToken loop.
func (l *L) primeStream() {
	if l.needFirstFill {
		l.firstFill()
	}
}

// readMore provides more input from the internal buffer or
// consumes from the input stream.
//
// This is a private implementation for a simplified buffered reader,
// allowing us to scan bytes without nested function calls
// at every single byte.
func (l *L) readMore() error {
	if l.consumed < l.bufferized {
		return nil
	}

	if l.keepPreviousBuffer > 0 {
		// copy the start of the buffer before reuse, for error context
		if l.previousBuffer == nil {
			l.previousBuffer = make([]byte, 0, l.keepPreviousBuffer)
		}

		copied := min(l.keepPreviousBuffer, l.bufferized)
		l.previousBuffer = l.previousBuffer[0:copied]
		copy(l.previousBuffer, l.buffer[:copied])
	}

	var err error
	l.bufferized, err = l.r.Read(l.buffer)
	if err != nil {
		l.bufferized = 0

		return err
	}

	l.consumed = 0

	return nil
}

func (l *L) consumeNull(_ byte) token.T {
	var buf [3]byte
	if err := l.consumeN(buf[:]); err != nil {
		l.err = codes.ErrInvalidToken

		return token.None
	}

	// n[ull]
	if !bytes.Equal(buf[:], []byte("ull")) {
		l.err = codes.ErrInvalidToken

		return token.None
	}

	return token.NullToken
}

// Reset returns the lexer to a clean, source-less state so it can be recycled.
//
// It is the method the pool invokes on both Borrow and Redeem, so it must be
// idempotent and must drop every reference to caller-supplied memory: in
// whole-buffer mode l.buffer aliases the caller's data, and l.r may hold the
// caller's reader. Leaving them in place would pin (and expose) user memory for
// as long as the recycled lexer sits in the pool.
//
// Reset does NOT rebind an input: call [L.ResetWithBytes] / [L.ResetWithReader]
// (or a Borrow*/New* constructor) to lex a new source. Configured options are
// preserved, and the streaming-owned buffer keeps its capacity for reuse.
func (l *L) Reset() {
	if l.wholeBuffer && l.r == noopReader {
		// whole-buffer via NewWithBytes/ResetWithBytes: l.buffer aliases the caller's
		// data — drop it so the pool does not pin user memory. A STREAMING lexer that
		// promoted to whole-buffer (§10.5f) still OWNS its buffer (l.r is the caller's
		// reader, not noopReader), so keep its capacity for reuse — it was refilled,
		// not aliased.
		l.buffer = nil
	}
	l.r = noopReader
	l.wholeBuffer = false
	l.needFirstFill = false // source-less until re-bound via ResetWith*
	l.bufferized = 0
	l.previousBuffer = l.previousBuffer[:0]
	l.reset()
}

// ResetWithBytes rebinds the lexer to a new input buffer and resets all scanning
// state, so a single lexer can be reused across inputs with no allocation (the
// no-pool equivalent of [BorrowLexerWithBytes]). Configured options are kept.
//
// The lexer operates in whole-buffer mode over data: emitted token values may
// alias data, which must therefore stay stable until the lexer is done with it.
func (l *L) ResetWithBytes(data []byte) {
	l.r = noopReader
	l.buffer = data
	l.bufferized = len(data)
	l.previousBuffer = nil
	l.keepPreviousBuffer = 0 // disabled option
	l.wholeBuffer = true     // the whole input is in the buffer: values may alias it
	l.needFirstFill = false
	l.reset()
}

// ResetWithReader rebinds the lexer to a new reader and resets all scanning
// state, so a single lexer can be reused across inputs (the no-pool equivalent
// of [BorrowLexerWithReader]). Configured options are kept and the internal
// buffer is reused.
func (l *L) ResetWithReader(r io.Reader) {
	l.r = r
	l.bufferized = 0
	l.wholeBuffer = false  // streaming: the buffer is refilled, values must be copied
	l.needFirstFill = true // §10.5f: the initial read + whole-buffer short-circuit is pending
	l.reset()

	if cap(l.buffer) < l.bufferSize {
		l.buffer = slices.Grow(l.buffer, l.bufferSize-cap(l.buffer))[:l.bufferSize]
	}

	if l.keepPreviousBuffer > 0 && cap(l.previousBuffer) < l.keepPreviousBuffer {
		l.previousBuffer = slices.Grow(l.previousBuffer, l.keepPreviousBuffer-cap(l.previousBuffer))
	}
}

func (l *L) reset() {
	l.err = nil
	l.current = token.None
	l.offset = 0
	l.consumed = 0
	l.errContext = nil
	l.expectKey = false
	l.afterKey = false
	l.isAtEOF = false
	l.line = 1
	l.lineStart = 0
	l.tokLine = 0
	l.tokCol = 0
	l.currentValue = l.currentValue[:0] // TODO: possibly preallocate value buffer to some configurable size

	if l.nestingLevel != nil {
		l.nestingLevel = l.nestingLevel[:1]
	} else {
		l.nestingLevel = make([]uint64, 1) // TODO: preallocate more if maxContainerStack > 63
	}
	l.nestingLevel[0] = 1 // the initial value for the stack must be 1: this bit is thereafter shifted right or left
}

func (l *L) SetErr(err error) {
	l.err = err
}

// setErrContext populates the error context if an error has been detected.
func (l *L) setErrContext() {
	if l.err == nil {
		return
	}

	var (
		window string
		pos    int
	)

	if l.keepPreviousBuffer > 0 && l.previousBuffer != nil && l.consumed < l.keepPreviousBuffer {
		// add the copied previousBuffer if the error came early in the main buffer
		window = string(l.previousBuffer[0:len(l.previousBuffer)]) + string(l.buffer[:l.bufferized])
		pos = l.consumed + len(l.previousBuffer)
	} else {
		window = string(l.buffer[:l.bufferized])
		pos = l.consumed
	}

	l.errContext = &codes.ErrContext{
		Err:      l.err,
		Offset:   l.offset,
		Buffer:   window,
		Position: pos,
	}
}

// consumeN consumes a small buffer of n bytes to decide tokens
// such as "true", "false" or "null".
func (l *L) consumeN(buffer []byte) error {
	minReadSize := len(buffer)
	n := 0

	for {
		if err := l.readMore(); err != nil {
			return err
		}

		// need is how many more bytes this token still requires.
		need := minReadSize - n
		if delta := l.bufferized - l.consumed; delta < need {
			// the window holds fewer than we need: take all of it, then refill.
			copy(buffer[n:], l.buffer[l.consumed:l.bufferized])
			l.consumed += delta
			l.offset += uint64(delta)
			n += delta

			continue
		}

		// the window holds at least `need` bytes: take EXACTLY need, advancing
		// consumed by need — NOT by the whole window. Advancing by the full delta
		// (the old `delta < minReadSize` form) skipped the surplus, which belongs to
		// the following token, breaking a literal read through a window smaller than
		// the literal (e.g. "true"/"null" via a 2-byte buffer dropped the trailing
		// separator). Unreachable once WithBufferSize floors the window, but correct
		// at any size.
		copy(buffer[n:minReadSize], l.buffer[l.consumed:l.consumed+need])
		l.consumed += need
		l.offset += uint64(need)

		break
	}

	return nil
}
