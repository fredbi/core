package lexer

import (
	"errors"
	"io"

	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/lexers/token"
)

// consumeNumber consumes a JSON number as a token.
//
// start is the previously consumed byte that decided to parse a number.
//
// Refer to https://www.rfc-editor.org/rfc/rfc8259#page-7
func (l *L) consumeNumber(start byte) (token.T, token.T) {
	var (
		isExponent     bool
		exponentSign   bool
		hasLeadingZero bool
		hasFractional  bool
		isFractional   bool
		integerPart    int
		fractionalPart int
		exponentPart   int
	)

	// The number is scanned without copying byte-by-byte: numStart marks the
	// start of the pending segment in l.buffer. In whole-buffer mode the value
	// aliases the input; otherwise the pending segment is bulk-copied into
	// currentValue (once at the end, or flushed when a streaming buffer is
	// refilled mid-number). This keeps the hot loop free of per-byte branches.
	numStart := l.consumed - 1
	l.currentValue = l.currentValue[:0]

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
		for l.consumed < l.bufferized {

			if l.maxValueBytes > 0 && len(l.currentValue)+l.consumed-numStart > l.maxValueBytes {
				l.err = codes.ErrMaxValueBytes

				return token.None, token.None
			}

			b := l.buffer[l.consumed]
			l.consumed++
			l.offset++

			switch {
			case b == decimalPoint:
				if hasFractional || isExponent {
					// only 1 decimal separator allowed, exponent is integer
					l.err = codes.ErrRepeatedDecimalSeparator

					return token.None, token.None
				}

				hasFractional = true
				isFractional = true

			case b == '+' || b == '-':
				if !isExponent || exponentPart > 0 || exponentSign {
					// a sign is only valid right after the exponent marker,
					// before any exponent digit and only once
					l.err = codes.ErrInvalidSign

					return token.None, token.None
				}
				exponentSign = true

			case b == 'e' || b == 'E':
				if isExponent {
					l.err = codes.ErrRepeatedExponent

					return token.None, token.None
				}

				isExponent = true
				isFractional = false

			case b == '0':
				if hasLeadingZero && !isFractional && !isExponent {
					// no leading zeroes on integer part, unless this is just 0
					l.err = codes.ErrLeadingZero

					return token.None, token.None
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

			case b >= '1' && b <= '9':
				if hasLeadingZero && !isFractional && !isExponent {
					l.err = codes.ErrLeadingZero

					return token.None, token.None
				}

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

					return token.None, token.None
				}

				start = b

				break NUMBER
			}
		}

		// buffer exhausted mid-number: in streaming mode preserve the pending
		// segment before readMore overwrites it
		if !l.wholeBuffer {
			l.currentValue = append(l.currentValue, l.buffer[numStart:l.consumed]...)
			numStart = l.consumed
		}

		if err := l.readMore(); err != nil {
			if errors.Is(err, io.EOF) {
				break NUMBER
			}

			l.err = err

			return token.None, token.None
		}

		numStart = 0 // the buffer was refilled: the pending segment restarts at 0
	}

	if hasFractional && fractionalPart == 0 {
		// a decimal point must be followed by at least one fractional digit
		l.err = codes.ErrInvalidFractional
		return token.None, token.None
	}

	if isExponent && exponentPart == 0 {
		l.err = codes.ErrInvalidExponent

		return token.None, token.None
	}

	if hasLeadingZero && integerPart > 1 {
		l.err = codes.ErrLeadingZero

		return token.None, token.None
	}

	if integerPart == 0 {
		l.err = codes.ErrMissingInteger

		return token.None, token.None
	}

	// a terminator byte (start != 0) was consumed past the number; EOF (start == 0) was not
	numEnd := l.consumed
	if start != 0 {
		// un-consume the terminator: with the look-ahead folded out, the next
		// scan validates it via the standard start-of-token checks
		numEnd--
		l.consumed = numEnd
		l.offset--
	}

	var value []byte
	if l.wholeBuffer {
		// alias the contiguous number bytes in the input buffer (cap == len)
		value = l.buffer[numStart:numEnd:numEnd]
	} else {
		// bulk-copy the final pending segment after any earlier flushed segments
		l.currentValue = append(l.currentValue, l.buffer[numStart:numEnd]...)
		value = l.currentValue
	}

	return token.MakeWithValue(token.Number, value), token.None
}
