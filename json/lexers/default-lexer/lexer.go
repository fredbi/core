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

	expectKey bool
	isAtEOF   bool
	lastStack uint64

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

	return l.errContext
}

// Offset yields the number of bytes consumed by the lexer so far.
func (l *L) Offset() uint64 {
	return l.offset
}

// NextToken returns the next JSON token consumed from the stream or slice of bytes.
//
// The last token is of Kind EOF.
//
// If the lexer is in an errored status, it will keep responding tokens with Kind Unknown.
//
// Tokens are expected to have a short lifespan: when NextToken is
// called again, the memory allocated to support the value of the
// previously returned token is reused for the next token.
//
// If you want to keep tokens for later reuse, you may clone a token
// using its [T.Clone] method.
func (l *L) NextToken() token.T { //nolint: gocognit
	if l.err != nil {
		return token.None
	}

	defer l.setErrContext() // TODO: get rid of defer, which consumes extra CPU

	if l.next.Kind() != token.Unknown {
		// we have already looked ahead: return the Look-Ahead token
		l.current = l.next
		l.next = token.None
		l.lastStack = 0

		return l.current
	}

	for {
		if err := l.readMore(); err != nil {
			return l.errCheck(err)
		}

		for {
			if l.consumed >= l.bufferized {
				break
			}

			b := l.buffer[l.consumed]
			l.offset++
			l.consumed++

			switch b {
			case blank, tab, carriageReturn, lineFeed:
				// ignore blanks
				continue

			case colon:
				if l.current.Kind() != token.String {
					// colon must be after a string litteral
					l.err = codes.ErrMissingKey
					l.next = token.None

					return token.None
				}

				if !l.isInObject() {
					// colon must appear in an object context
					l.err = codes.ErrMissingObject
					l.next = token.None

					return token.None
				}

				l.current = token.MakeDelimiter(token.Colon)
				l.next = token.None

				return l.current

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
				if l.isInObject() {
					// TODO: is it possible to already have expectKey true at this point?
					l.expectKey = true
				}

				l.current = token.MakeDelimiter(token.Comma)
				l.next = token.None

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
						if l.current.Delimiter() != token.OpeningSquareBracket && l.current.Delimiter() != token.Comma {
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
	l.current = token.None
	l.next = token.None

	if errors.Is(err, io.EOF) {
		if l.isInContainer() {
			if l.isInObject() {
				l.err = codes.ErrNotInObject
			} else {
				l.err = codes.ErrNotInArray
			}
		}

		if l.isAtEOF {
			l.err = io.EOF
		} else {
			l.isAtEOF = true
		}

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

	return l.lookAhead(token.NullToken, 0)
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
				return current, token.EOFToken
			}

			l.err = err

			return token.None, token.None
		}

		for {
			if l.consumed >= l.bufferized {
				break
			}

			b = l.buffer[l.consumed]
			l.consumed++
			l.offset++

			switch b {
			case blank, tab, carriageReturn, lineFeed:
				continue

			case colon:
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
			case blank, tab, carriageReturn, lineFeed:
				continue

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
				l.lastStack = l.nestingLevel[len(l.nestingLevel)-1] // save the current stack for the current token
				l.popContainer()

				return current, token.MakeDelimiter(token.ClosingBracket)

			case closingSquareBracket:
				if !l.isInArray() {
					l.err = codes.ErrNotInArray
					l.next = token.None

					return token.None, token.None
				}

				l.lastStack = l.nestingLevel[len(l.nestingLevel)-1] // save the current stack for the current token
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
	l.isAtEOF = false
	l.lastStack = 0
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
