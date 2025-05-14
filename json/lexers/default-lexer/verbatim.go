package lexer

import (
	"bytes"
	"errors"
	"io"
	"slices"

	codes "github.com/fredbi/core/json/lexers/error-codes"
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
	blanks     []byte
	nextBlanks []byte
	current    token.VT
	next       token.VT // the next token consumed whenever we need to look-ahead

	options
}

// NewVerbatim yields a new JSON verbatim lexer consuming from an io.Reader.
//
// The lexer performs some internal buffering on a fixed size buffer to call the reader on chunks.
//
// Use option WithBufferSize to alter the size of this buffer (defaults to 4KB).
//
// If you plan to allocate many lexers with a short life span, consider using the global pool
// with the BorrowLexer() and RedeemLexer() functions.
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
// with the BorrowLexerWithBytes() and RedeemLexer() functions.
func NewVerbatimWithBytes(data []byte, opts ...Option) *VL {
	l := new(VL)
	l.L = NewWithBytes(data, opts...)
	l.reset()

	return l
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
// using its Clone() method.
func (l *VL) NextToken() token.VT { //nolint: gocognit
	if l.err != nil {
		return token.VNone
	}

	defer l.setErrContext() // TODO: get rid of defer, which consumes extra CPU

	if l.next.Kind() != token.Unknown {
		// we have already looked ahead: return the Look-Ahead token
		l.current = l.next
		l.next = token.VNone
		l.lastStack = 0

		return l.current
	}

	if l.blanks != nil {
		l.blanks = l.blanks[:0]
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
				// save leading blanks
				l.blanks = append(l.blanks, b)

				// ignore blanks
				continue

			case colon:
				if l.current.Kind() != token.String {
					// colon must be after a string litteral
					l.err = codes.ErrMissingKey
					l.next = token.VNone

					return token.VNone
				}

				if !l.isInObject() {
					// colon must appear in an object context
					l.err = codes.ErrMissingObject
					l.next = token.VNone

					return token.VNone
				}

				l.current = token.MakeVerbatimDelimiter(token.Colon, l.blanks)
				l.next = token.VNone

				return l.current

			case closingBracket:
				if l.current.IsComma() {
					// closing bracket shouldn't appear after a comma
					l.err = codes.ErrTrailingComma
					l.next = token.VNone

					return token.VNone
				}

				if !l.isInObject() {
					// closing bracket should only appear in an object context
					l.err = codes.ErrNotInObject
					l.next = token.VNone

					return token.VNone
				}

				l.expectKey = false

				l.popContainer()
				l.current = token.MakeVerbatimDelimiter(token.ClosingBracket, l.blanks)
				l.next = token.VNone

				return l.current

			case closingSquareBracket:
				if l.current.IsComma() {
					// closing square bracket shouldn't appear after a comma
					l.err = codes.ErrTrailingComma
					l.next = token.VNone

					return token.VNone
				}

				if !l.isInArray() {
					// closing square bracket should only appear in an array context
					l.err = codes.ErrNotInArray
					l.next = token.VNone

					return token.VNone
				}

				l.popContainer()
				l.current = token.MakeVerbatimDelimiter(token.ClosingSquareBracket, l.blanks)
				l.next = token.VNone

				return l.current

			case comma:
				if l.current.IsComma() {
					l.err = codes.ErrRepeatedComma
					l.next = token.VNone

					return token.VNone
				}

				if l.expectKey {
					l.err = codes.ErrMissingKey
					l.next = token.VNone

					return token.VNone
				}

				if !l.isInContainer() {
					l.err = codes.ErrCommaInContainer
					l.next = token.VNone

					return token.VNone
				}
				if l.isInObject() {
					// TODO: is it possible to already have expectKey true at this point?
					l.expectKey = true
				}

				l.current = token.MakeVerbatimDelimiter(token.Comma, l.blanks)
				l.next = token.VNone

				return l.current

			case openingBracket:
				if l.current.IsKnown() {
					if l.current.Kind() != token.Delimiter {
						l.err = codes.ErrInvalidToken
						l.next = token.VNone

						return token.VNone
					}

					if l.current.Delimiter().IsClosing() {
						l.err = codes.ErrMissingComma
						l.next = token.VNone

						return token.VNone
					}

					if l.isInArray() {
						if l.current.Delimiter() != token.OpeningSquareBracket && l.current.Delimiter() != token.Comma {
							l.err = codes.ErrMissingComma
							l.next = token.VNone

							return token.VNone
						}
					} else {
						if !l.current.IsColon() {
							l.err = codes.ErrMissingKey
							l.next = token.VNone

							return token.VNone
						}
					}
				}

				if l.expectKey {
					l.err = codes.ErrMissingKey
					l.next = token.VNone

					return token.VNone
				}

				l.expectKey = true
				l.pushObject()
				if l.err != nil {
					l.next = token.VNone

					return token.VNone
				}

				l.current = token.MakeVerbatimDelimiter(token.OpeningBracket, l.blanks)
				l.next = token.VNone

				return l.current

			case openingSquareBracket:
				if l.current.IsKnown() {
					if l.current.Kind() != token.Delimiter {
						l.err = codes.ErrInvalidToken
						l.next = token.VNone

						return token.VNone
					}

					if l.current.Delimiter().IsClosing() {
						l.err = codes.ErrMissingComma
						l.next = token.VNone

						return token.VNone
					}
				}

				if l.expectKey {
					l.err = codes.ErrMissingKey
					l.next = token.VNone

					return token.VNone
				}

				l.pushArray()
				if l.err != nil {
					l.next = token.VNone

					return token.VNone
				}

				l.current = token.MakeVerbatimDelimiter(token.OpeningSquareBracket, l.blanks)
				l.next = token.VNone

				return l.current

			case doubleQuote:
				if l.current.IsKnown() {
					// value should be standalone or appear after a delimiter
					if !l.current.Delimiter().AcceptValue() {
						l.err = codes.ErrDelimitedValue
						l.current = token.VNone
						l.next = token.VNone

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
						l.current = token.VNone
						l.next = token.VNone

						return l.current
					}
				}

				if l.expectKey {
					l.err = codes.ErrMissingKey
					l.next = token.VNone

					return token.VNone
				}

				l.current, l.next = l.consumeBoolean(b)

				return l.current

			case minusSign, decimalPoint, '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				if l.current.IsKnown() {
					// value should be standalone or appear after a delimiter
					if !l.current.Delimiter().AcceptValue() {
						l.err = codes.ErrDelimitedValue
						l.current = token.VNone
						l.next = token.VNone

						return l.current
					}
				}

				if l.expectKey {
					l.err = codes.ErrMissingKey
					l.next = token.VNone

					return token.VNone
				}

				l.current, l.next = l.consumeNumber(b)

				return l.current

			case startOfNull:
				if l.current.IsKnown() {
					// value should be standalone or appear after a delimiter
					if !l.current.Delimiter().AcceptValue() {
						l.err = codes.ErrDelimitedValue
						l.current = token.VNone
						l.next = token.VNone

						return l.current
					}
				}

				if l.expectKey {
					l.err = codes.ErrMissingKey
					l.next = token.VNone

					return token.VNone
				}

				l.current, l.next = l.consumeNull(b)

				return l.current

			default:
				l.err = codes.ErrInvalidToken
				l.current = l.next
				l.next = token.VNone

				return token.VNone
			}
		}
	}
}

func (l *VL) errCheck(err error) token.VT {
	l.current = token.VNone
	l.next = token.VNone

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

		return token.MakeVerbatimEOF(l.blanks)
	}

	l.err = err

	return l.current
}

func (l *VL) consumeString() token.VT {
	var escapeSequence bool
	l.currentValue = l.currentValue[:0]

	for {
		if err := l.readMore(); err != nil {
			if errors.Is(err, io.EOF) {
				l.err = codes.ErrUnterminatedString
			} else {
				l.err = err
			}

			return token.VNone
		}

		for {
			if l.consumed >= l.bufferized {
				break
			}

			if l.maxValueBytes > 0 && len(l.currentValue) > l.maxValueBytes {
				l.err = codes.ErrMaxValueBytes

				return token.VNone
			}

			b := l.buffer[l.consumed]
			l.offset++
			l.consumed++

			switch b {
			case escape:
				if escapeSequence {
					//  "\\"
					l.currentValue = append(l.currentValue, b)
					escapeSequence = false

					continue
				}

				escapeSequence = true

			case doubleQuote:
				if escapeSequence {
					//  "\""
					escapeSequence = false
					l.currentValue = append(l.currentValue, b)

					continue
				}

				if l.expectKey {
					l.current, l.next = l.expectColon(token.MakeVerbatimWithValue(token.Key, l.currentValue, l.blanks))
					l.expectKey = false

					return l.current
				}

				return token.MakeVerbatimWithValue(token.String, l.currentValue, l.blanks)

			case slash:
				if escapeSequence {
					// "\/"
					escapeSequence = false
				}

				l.currentValue = append(l.currentValue, b)

			case 'b', 'f', 'n', 't', 'r':
				if !escapeSequence {
					l.currentValue = append(l.currentValue, b)

					continue
				}
				// shorthand escaped representations of popular characters
				// https://www.rfc-editor.org/rfc/rfc8259#page-9
				escapeSequence = false

				switch b {
				case 'b':
					l.currentValue = append(l.currentValue, '\b')
				case 'f':
					l.currentValue = append(l.currentValue, '\f')
				case 'n':
					l.currentValue = append(l.currentValue, '\n')
				case 't':
					l.currentValue = append(l.currentValue, '\t')
				case 'r':
					l.currentValue = append(l.currentValue, '\r')
				}

			case 'u':
				if !escapeSequence {
					l.currentValue = append(l.currentValue, b)

					continue
				}

				escapeSequence = false
				var buf [4]byte
				if err := l.consumeN(buf[:]); err != nil {
					l.err = codes.ErrUnicodeEscape

					return token.VNone
				}

				l.currentValue = append(l.currentValue, buf[:]...)

			default:
				if escapeSequence {
					l.err = codes.ErrUnknownEscape

					return token.VNone
				}

				l.currentValue = append(l.currentValue, b)
			}
		}
	}
}

// consumeNumber consumes a JSON number as a token.
//
// start is the previously consumed byte that decided to parse a number.
//
// Refer to https://www.rfc-editor.org/rfc/rfc8259#page-7
func (l *VL) consumeNumber(start byte) (token.VT, token.VT) {
	var (
		isExponent     bool
		hasLeadingZero bool
		hasFractional  bool
		isFractional   bool
		integerPart    int
		fractionalPart int
		exponentPart   int
	)

	l.currentValue = l.currentValue[:0]
	l.currentValue = append(l.currentValue, start)

	switch {
	case start == decimalPoint:
		hasFractional = true
		isFractional = hasFractional
	case start == '0':
		hasLeadingZero = true
		integerPart++
	case start >= '1' && start <= '9':
		integerPart++
	}
	start = 0

NUMBER:
	for {
		if err := l.readMore(); err != nil {
			if errors.Is(err, io.EOF) {
				break NUMBER
			}

			l.err = err

			return token.VNone, token.VNone
		}

		for {
			if l.consumed >= l.bufferized {
				break
			}

			if l.maxValueBytes > 0 && len(l.currentValue) > l.maxValueBytes {
				l.err = codes.ErrMaxValueBytes

				return token.VNone, token.VNone
			}

			b := l.buffer[l.consumed]
			l.consumed++
			l.offset++

			switch {
			case b == decimalPoint:
				if hasFractional || isExponent {
					// only 1 decimal separator allowed, exponent is integer
					l.err = codes.ErrRepeatedDecimalSeparator

					return token.VNone, token.VNone
				}

				hasFractional = true
				isFractional = true
				l.currentValue = append(l.currentValue, b)

			case b == '+' || b == '-':
				if !isExponent || exponentPart > 0 {
					l.err = codes.ErrInvalidSign

					return token.VNone, token.VNone
				}
				l.currentValue = append(l.currentValue, b)

			case b == 'e' || b == 'E':
				if isExponent {
					l.err = codes.ErrRepeatedExponent

					return token.VNone, token.VNone
				}

				isExponent = true
				isFractional = false
				l.currentValue = append(l.currentValue, b)

			case b == '0':
				if hasLeadingZero && !isFractional && !isExponent {
					// no leading zeroes on integer part, unless this is just 0
					l.err = codes.ErrLeadingZero

					return token.VNone, token.VNone
				}

				switch {
				case isFractional:
					fractionalPart++
				case isExponent:
					exponentPart++
				default:
					integerPart++
					if integerPart == 1 {
						hasLeadingZero = true
					}
				}

				l.currentValue = append(l.currentValue, b)

			case b >= '1' && b <= '9':
				if hasLeadingZero && !isFractional && !isExponent {
					l.err = codes.ErrLeadingZero

					return token.VNone, token.VNone
				}

				l.currentValue = append(l.currentValue, b)

				switch {
				case isFractional:
					fractionalPart++
				case isExponent:
					exponentPart++
				default:
					integerPart++
				}

			default:
				if b == 0 {
					l.err = codes.ErrInvalidToken

					return token.VNone, token.VNone
				}

				start = b

				break NUMBER
			}
		}
	}

	if isFractional && fractionalPart == 0 {
		l.err = codes.ErrInvalidFractional
		return token.VNone, token.VNone
	}

	if isExponent && exponentPart == 0 {
		l.err = codes.ErrInvalidExponent

		return token.VNone, token.VNone
	}

	if hasLeadingZero && integerPart > 1 {
		l.err = codes.ErrLeadingZero

		return token.VNone, token.VNone
	}

	if integerPart == 0 {
		l.err = codes.ErrMissingInteger

		return token.VNone, token.VNone
	}

	return l.lookAhead(token.MakeVerbatimWithValue(token.Number, l.currentValue, l.blanks), start)
}

func (l *VL) consumeBoolean(start byte) (token.VT, token.VT) {
	var buf [3]byte
	if err := l.consumeN(buf[:]); err != nil {
		l.err = codes.ErrInvalidToken

		return token.VNone, token.VNone
	}

	var value bool

	// t[rue] | f[als][e]
	switch {
	case start == startOfTrue && bytes.Equal(buf[:], []byte("rue")):
		value = true
	case start == startOfFalse && bytes.Equal(buf[:], []byte("als")):
		if l.consumed >= l.bufferized {
			if err := l.readMore(); err != nil {
				l.err = codes.ErrInvalidToken

				return token.VNone, token.VNone
			}
		}
		next := l.buffer[l.consumed]
		l.consumed++
		l.offset++

		if next != 'e' {
			l.err = codes.ErrInvalidToken

			return token.VNone, token.VNone
		}

		value = false
	default:
		l.err = codes.ErrInvalidToken

		return token.VNone, token.VNone
	}

	return l.lookAhead(token.MakeVerbatimBoolean(value, l.blanks), 0)
}

func (l *VL) consumeNull(_ byte) (token.VT, token.VT) {
	var buf [3]byte
	if err := l.consumeN(buf[:]); err != nil {
		l.err = codes.ErrInvalidToken

		return token.VNone, token.VNone
	}

	// n[ull]
	if !bytes.Equal(buf[:], []byte("ull")) {
		l.err = codes.ErrInvalidToken

		return token.VNone, token.VNone
	}

	return l.lookAhead(token.MakeVerbatimNull(l.blanks), 0)
}

// expectColon scans the input for a ":" delimiter after a key string
// has been found.
func (l *VL) expectColon(current token.VT) (token.VT, token.VT) {
	var (
		err error
		b   byte
	)

	if l.nextBlanks != nil {
		l.nextBlanks = l.nextBlanks[:0]
	}

	for {
		if err = l.readMore(); err != nil {
			if errors.Is(err, io.EOF) {
				return current, token.MakeVerbatimEOF(l.nextBlanks)
			}

			l.err = err

			return token.VNone, token.VNone
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
				// save leading blanks
				l.nextBlanks = append(l.nextBlanks, b)

				continue

			case colon:
				return current, token.MakeVerbatimDelimiter(token.Colon, l.nextBlanks)

			default:
				l.err = codes.ErrKeyColon

				return token.VNone, token.VNone
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
func (l *VL) lookAhead(current token.VT, start byte) (token.VT, token.VT) {
	var (
		err error
		b   byte
	)

	if l.nextBlanks != nil {
		l.nextBlanks = l.nextBlanks[:0]
	}

	for {
		if start == 0 {
			if err = l.readMore(); err != nil {
				if errors.Is(err, io.EOF) {
					return current, token.MakeVerbatimEOF(l.nextBlanks)
				}

				l.err = err

				return token.VNone, token.VNone
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
				// save leading blanks
				l.nextBlanks = append(l.nextBlanks, b)

				continue

			case comma:
				if l.isInObject() {
					// TODO: is it possible to already have expectKey true at this point?
					l.expectKey = true
				}

				return current, token.MakeVerbatimDelimiter(token.Comma, l.nextBlanks)

			case closingBracket:
				if !l.isInObject() {
					l.err = codes.ErrNotInObject
					l.next = token.VNone

					return token.VNone, token.VNone
				}

				l.expectKey = false
				l.lastStack = l.nestingLevel[len(l.nestingLevel)-1] // save the current stack for the current token
				l.popContainer()

				return current, token.MakeVerbatimDelimiter(token.ClosingBracket, l.nextBlanks)

			case closingSquareBracket:
				if !l.isInArray() {
					l.err = codes.ErrNotInArray
					l.next = token.VNone

					return token.VNone, token.VNone
				}

				l.lastStack = l.nestingLevel[len(l.nestingLevel)-1] // save the current stack for the current token
				l.popContainer()

				return current, token.MakeVerbatimDelimiter(token.ClosingSquareBracket, l.nextBlanks)

			default:
				l.err = codes.ErrInvalidToken

				return token.VNone, token.VNone
			}
		}
	}
}

func (l *VL) Reset() {
	l.bufferized = 0
	l.L.reset()
	l.reset()

	if cap(l.buffer) < l.bufferSize {
		// reallocates an internal buffer only if options have changed
		l.buffer = slices.Grow(l.buffer, l.bufferSize-cap(l.buffer))[:l.bufferSize]
	}

	if l.keepPreviousBuffer > 0 && cap(l.previousBuffer) < l.keepPreviousBuffer {
		l.previousBuffer = slices.Grow(l.previousBuffer, l.keepPreviousBuffer-cap(l.previousBuffer))
	}
}

func (l *VL) reset() {
	l.next = token.VNone
	l.current = token.VNone
	l.blanks = l.blanks[:0]
	l.nextBlanks = l.nextBlanks[:0]
}
