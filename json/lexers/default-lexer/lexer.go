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
// with the [BorrowLexer] and [RedeemLexer] functions.
type L struct {
	r          io.Reader
	err        error
	errContext *codes.ErrContext

	buffer         []byte   // determined by bufferSize
	currentValue   []byte   // capped if maxValueBytes > 0
	previousBuffer []byte   // used when keepPreviousBuffer=true
	nestingLevel   []uint64 // the stack of nested containers. Every bit represent an extra nesting. Capped if maxContainerStack > 0

	current token.T
	next    token.T // the next token consumed whenever we need to look-ahead

	offset     uint64
	consumed   int
	bufferized int

	line      int    // current line, 1-based
	lineStart uint64 // offset of the first byte of the current line
	tokLine   int    // line of the most recent token's start
	tokCol    int    // column of the most recent token's start, 1-based
	nextLine  int    // pending position of the looked-ahead token (l.next)
	nextCol   int

	expectKey   bool
	afterKey    bool // the previous token was an object key: a ':' must follow
	isAtEOF     bool
	wholeBuffer bool // the buffer holds the entire input (no refills): values may alias it
	lastStack   uint64

	options
}

// New JSON lexer consuming from an [io.Reader].
//
// The lexer performs some internal buffering on a fixed size buffer to call the reader on chunks.
//
// Use option [WithBufferSize] to alter the size of this buffer (the default is 4KB).
//
// If you plan to allocate many lexers with a short life span, consider using the global pool
// with the [BorrowLexer] and [RedeemLexer] functions.
func New(r io.Reader, opts ...Option) *L {
	l := new(L)
	l.applyWithDefaults(opts)
	l.r = r
	l.buffer = make([]byte, l.bufferSize)
	l.bufferized = 0
	l.wholeBuffer = false // streaming: the buffer is refilled, values must be copied

	l.reset()

	return l
}

// NewWithBytes yields a new JSON lexer consuming from a provided fixed buffer of bytes.
//
// Since the full buffer is provided by the caller, there is no additional internal buffering.
//
// If you plan to allocate many lexers with a short life span, consider using the global pool
// with the [BorrowLexerWithBytes] and [RedeemLexer] functions.
func NewWithBytes(data []byte, opts ...Option) *L {
	l := new(L)
	l.applyWithDefaults(opts)
	l.r = noopReader
	l.buffer = data
	l.bufferized = len(data)
	l.previousBuffer = nil
	l.keepPreviousBuffer = 0 // disabled option
	l.wholeBuffer = true     // the whole input is in the buffer: values may alias it

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

// Line yields the 1-based line number at which the most recently returned token
// starts. It is 0 before the first token.
//
// Line tracking is always on; its cost is one increment per newline byte.
func (l *L) Line() int {
	return l.tokLine
}

// Column yields the 1-based column at which the most recently returned token
// starts. It is 0 before the first token.
func (l *L) Column() int {
	return l.tokCol
}

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
	return l.scanToken()
}

// scanToken scans and returns the next token. When elideSeparator is set, the
// structural separators "," and ":" are validated and skipped inline (their
// context is recorded in l.current) rather than surfaced — no per-separator
// re-entry. The key->colon path still stashes the colon in l.next.
func (l *L) scanToken() token.T { //nolint: gocognit
	if l.err != nil {
		return token.None
	}

	for {
		if err := l.readMore(); err != nil {
			return l.errCheck(err)
		}

		for l.consumed < l.bufferized {

			b := l.buffer[l.consumed]
			l.offset++
			l.consumed++

			switch b {
			case lineFeed:
				l.line++
				l.lineStart = l.offset

				continue

			case blank, tab, carriageReturn:
				// ignore non-significant blank space
				continue
			}

			// a significant byte starts a token: snapshot its position
			l.tokLine = l.line
			l.tokCol = int(l.offset - l.lineStart)

			// an object key must be followed by the ':' name-separator
			if l.afterKey {
				l.afterKey = false
				if b != colon {
					l.err = codes.ErrKeyColon
					l.next = token.None

					return token.None
				}

				l.current = token.MakeDelimiter(token.Colon)
				if l.elideSeparator {
					continue // skip the colon; context recorded in l.current
				}

				return l.current
			}

			switch b {
			case colon:
				// a stray colon: only an object key (handled above) may precede one
				if l.current.Kind() == token.String {
					// a string value (not a key) followed by a colon
					l.err = codes.ErrMissingObject
				} else {
					l.err = codes.ErrMissingKey
				}
				l.next = token.None

				return token.None

			case closingBracket:
				if l.current.IsComma() {
					// closing bracket shouldn't appear after a comma
					l.err = codes.ErrTrailingComma
					l.next = token.None

					return token.None
				}

				if !l.isInObject() {
					// closing bracket should only appear in an object context
					l.err = codes.ErrNotInObject
					l.next = token.None

					return token.None
				}

				l.expectKey = false

				l.popContainer()
				l.current = token.MakeDelimiter(token.ClosingBracket)
				l.next = token.None

				return l.current

			case closingSquareBracket:
				if l.current.IsComma() {
					// closing square bracket shouldn't appear after a comma
					l.err = codes.ErrTrailingComma
					l.next = token.None

					return token.None
				}

				if !l.isInArray() {
					// closing square bracket should only appear in an array context
					l.err = codes.ErrNotInArray
					l.next = token.None

					return token.None
				}

				l.popContainer()
				l.current = token.MakeDelimiter(token.ClosingSquareBracket)
				l.next = token.None

				return l.current

			case comma:
				if l.current.IsComma() {
					l.err = codes.ErrRepeatedComma
					l.next = token.None

					return token.None
				}

				if l.expectKey {
					l.err = codes.ErrMissingKey
					l.next = token.None

					return token.None
				}

				if !l.isInContainer() {
					l.err = codes.ErrCommaInContainer
					l.next = token.None

					return token.None
				}

				if l.current.IsStartObject() || l.current.IsStartArray() || l.current.IsColon() {
					// a comma must follow a value or a closing delimiter,
					// never an opening delimiter or a colon
					l.err = codes.ErrMissingValue
					l.next = token.None

					return token.None
				}

				if l.isInObject() {
					// TODO: is it possible to already have expectKey true at this point?
					l.expectKey = true
				}

				l.current = token.MakeDelimiter(token.Comma)
				l.next = token.None

				if l.elideSeparator {
					continue // skip the comma; context recorded in l.current
				}

				return l.current

			case openingBracket:
				if l.current.IsKnown() {
					if l.current.Kind() != token.Delimiter {
						l.err = codes.ErrInvalidToken
						l.next = token.None

						return token.None
					}

					if l.current.Delimiter().IsClosing() {
						l.err = codes.ErrMissingComma
						l.next = token.None

						return token.None
					}

					if l.isInArray() {
						if l.current.Delimiter() != token.OpeningSquareBracket &&
							l.current.Delimiter() != token.Comma {
							l.err = codes.ErrMissingComma
							l.next = token.None

							return token.None
						}
					} else {
						if !l.current.IsColon() {
							l.err = codes.ErrMissingKey
							l.next = token.None

							return token.None
						}
					}
				}

				if l.expectKey {
					l.err = codes.ErrMissingKey
					l.next = token.None

					return token.None
				}

				l.expectKey = true
				l.pushObject()
				if l.err != nil {
					l.next = token.None

					return token.None
				}

				l.current = token.MakeDelimiter(token.OpeningBracket)
				l.next = token.None

				return l.current

			case openingSquareBracket:
				if l.current.IsKnown() {
					if l.current.Kind() != token.Delimiter {
						l.err = codes.ErrInvalidToken
						l.next = token.None

						return token.None
					}

					if l.current.Delimiter().IsClosing() {
						l.err = codes.ErrMissingComma
						l.next = token.None

						return token.None
					}
				}

				if l.expectKey {
					l.err = codes.ErrMissingKey
					l.next = token.None

					return token.None
				}

				l.pushArray()
				if l.err != nil {
					l.next = token.None

					return token.None
				}

				l.current = token.MakeDelimiter(token.OpeningSquareBracket)
				l.next = token.None

				return l.current

			case doubleQuote:
				if l.current.IsKnown() {
					// value should be standalone or appear after a delimiter
					if !l.current.Delimiter().AcceptValue() {
						l.err = codes.ErrDelimitedValue
						l.current = token.None
						l.next = token.None

						return l.current
					}
				}

				l.current = l.consumeString()

				return l.current

			case startOfTrue, startOfFalse:
				if l.current.IsKnown() {
					// value should be standalone or appear after a delimiter
					if !l.current.Delimiter().AcceptValue() {
						l.err = codes.ErrDelimitedValue
						l.current = token.None
						l.next = token.None

						return l.current
					}
				}

				if l.expectKey {
					l.err = codes.ErrMissingKey
					l.next = token.None

					return token.None
				}

				l.current, l.next = l.consumeBoolean(b)

				return l.current

			case minusSign, decimalPoint, '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				if l.current.IsKnown() {
					// value should be standalone or appear after a delimiter
					if !l.current.Delimiter().AcceptValue() {
						l.err = codes.ErrDelimitedValue
						l.current = token.None
						l.next = token.None

						return l.current
					}
				}

				if l.expectKey {
					l.err = codes.ErrMissingKey
					l.next = token.None

					return token.None
				}

				l.current, l.next = l.consumeNumber(b)

				return l.current

			case startOfNull:
				if l.current.IsKnown() {
					// value should be standalone or appear after a delimiter
					if !l.current.Delimiter().AcceptValue() {
						l.err = codes.ErrDelimitedValue
						l.current = token.None
						l.next = token.None

						return l.current
					}
				}

				if l.expectKey {
					l.err = codes.ErrMissingKey
					l.next = token.None

					return token.None
				}

				l.current, l.next = l.consumeNull(b)

				return l.current

			default:
				l.err = codes.ErrInvalidToken
				l.current = l.next
				l.next = token.None

				return token.None
			}
		}
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

func (l *L) errCheck(err error) token.T {
	hadToken := l.current.IsKnown()
	l.current = token.None
	l.next = token.None

	if errors.Is(err, io.EOF) {
		switch {
		case l.isInContainer():
			if l.isInObject() {
				l.err = codes.ErrNotInObject
			} else {
				l.err = codes.ErrNotInArray
			}
		case l.isAtEOF:
			l.err = io.EOF
		case !hadToken:
			// end of input reached without any top-level value
			l.err = codes.ErrNoData
		}

		l.isAtEOF = true

		return token.EOFToken
	}

	l.err = err

	return l.current
}

func (l *L) consumeNull(_ byte) (token.T, token.T) {
	var buf [3]byte
	if err := l.consumeN(buf[:]); err != nil {
		l.err = codes.ErrInvalidToken

		return token.None, token.None
	}

	// n[ull]
	if !bytes.Equal(buf[:], []byte("ull")) {
		l.err = codes.ErrInvalidToken

		return token.None, token.None
	}

	return token.NullToken, token.None
}

// expectColon scans the input for a ":" delimiter after a key string
// has been found.
func (l *L) expectColon(current token.T) (token.T, token.T) {
	var (
		err error
		b   byte
	)

	for {
		if err = l.readMore(); err != nil {
			if errors.Is(err, io.EOF) {
				// EOF reached while expecting ':' after an object key
				l.err = codes.ErrKeyColon

				return token.None, token.None
			}

			l.err = err

			return token.None, token.None
		}

		for l.consumed < l.bufferized {

			b = l.buffer[l.consumed]
			l.consumed++
			l.offset++

			switch b {
			case lineFeed:
				l.line++
				l.lineStart = l.offset

				continue

			case blank, tab, carriageReturn:
				continue

			case colon:
				l.nextLine = l.line
				l.nextCol = int(l.offset - l.lineStart)

				return current, token.MakeDelimiter(token.Colon)

			default:
				l.err = codes.ErrKeyColon

				return token.None, token.None
			}
		}
	}
}

// lookAhead to verify the end of a value token,
// i.e. followed by EOF, a comma or a closing delimiter.
//
// We do not expect a value to appear here.
//
// start is the byte that may have already been consumed.
func (l *L) lookAhead(current token.T, start byte) (token.T, token.T) {
	var (
		err error
		b   byte
	)

	for {
		if start == 0 {
			if err = l.readMore(); err != nil {
				if errors.Is(err, io.EOF) {
					if l.isInContainer() {
						// EOF reached before the container was closed
						if l.isInObject() {
							l.err = codes.ErrNotInObject
						} else {
							l.err = codes.ErrNotInArray
						}

						return token.None, token.None
					}

					return current, token.EOFToken
				}

				l.err = err

				return token.None, token.None
			}
		}

		for {
			if start == 0 {
				if l.consumed >= l.bufferized {
					break
				}

				b = l.buffer[l.consumed]
				l.consumed++
				l.offset++
			} else {
				b = start
				start = 0
			}

			switch b {
			case lineFeed:
				l.line++
				l.lineStart = l.offset

				continue

			case blank, tab, carriageReturn:
				continue
			}

			// the look-ahead token starts here: snapshot its position for when
			// it is later surfaced as l.next
			l.nextLine = l.line
			l.nextCol = int(l.offset - l.lineStart)

			switch b {
			case comma:
				if l.isInObject() {
					// TODO: is it possible to already have expectKey true at this point?
					l.expectKey = true
				}

				return current, token.MakeDelimiter(token.Comma)

			case closingBracket:
				if !l.isInObject() {
					l.err = codes.ErrNotInObject
					l.next = token.None

					return token.None, token.None
				}

				l.expectKey = false
				l.lastStack = uint64(l.depth()) // save the value's depth before the look-ahead pop
				l.popContainer()

				return current, token.MakeDelimiter(token.ClosingBracket)

			case closingSquareBracket:
				if !l.isInArray() {
					l.err = codes.ErrNotInArray
					l.next = token.None

					return token.None, token.None
				}

				l.lastStack = uint64(l.depth()) // save the value's depth before the look-ahead pop
				l.popContainer()

				return current, token.MakeDelimiter(token.ClosingSquareBracket)

			default:
				l.err = codes.ErrInvalidToken

				return token.None, token.None
			}
		}
	}
}

func (l *L) Reset() {
	l.bufferized = 0
	l.reset()

	if cap(l.buffer) < l.bufferSize {
		// reallocates an internal buffer only if options have changed
		l.buffer = slices.Grow(l.buffer, l.bufferSize-cap(l.buffer))[:l.bufferSize]
	}

	if l.keepPreviousBuffer > 0 && cap(l.previousBuffer) < l.keepPreviousBuffer {
		l.previousBuffer = slices.Grow(l.previousBuffer, l.keepPreviousBuffer-cap(l.previousBuffer))
	}
}

func (l *L) reset() {
	l.err = nil
	l.next = token.None
	l.current = token.None
	l.offset = 0
	l.consumed = 0
	l.errContext = nil
	l.expectKey = false
	l.afterKey = false
	l.isAtEOF = false
	l.lastStack = 0
	l.line = 1
	l.lineStart = 0
	l.tokLine = 0
	l.tokCol = 0
	l.nextLine = 0
	l.nextCol = 0
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

		if delta := l.bufferized - l.consumed; delta < minReadSize {
			copy(buffer[n:], l.buffer[l.consumed:l.bufferized])
			l.consumed += delta
			l.offset += uint64(delta)
			n += delta

			if n < minReadSize {
				continue
			}

			return nil
		}

		copy(buffer[n:minReadSize], l.buffer[l.consumed:l.consumed+minReadSize-n])
		l.consumed += minReadSize - n
		l.offset += uint64(minReadSize - n)

		break
	}

	return nil
}
