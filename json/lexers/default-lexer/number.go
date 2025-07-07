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

			return token.None, token.None
		}

		for l.consumed < l.bufferized {

			if l.maxValueBytes > 0 && len(l.currentValue) > l.maxValueBytes {
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
				l.currentValue = append(l.currentValue, b)

			case b == '+' || b == '-':
				if !isExponent || exponentPart > 0 {
					l.err = codes.ErrInvalidSign

					return token.None, token.None
				}
				l.currentValue = append(l.currentValue, b)

			case b == 'e' || b == 'E':
				if isExponent {
					l.err = codes.ErrRepeatedExponent

					return token.None, token.None
				}

				isExponent = true
				isFractional = false
				l.currentValue = append(l.currentValue, b)

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

				l.currentValue = append(l.currentValue, b)

			case b >= '1' && b <= '9':
				if hasLeadingZero && !isFractional && !isExponent {
					l.err = codes.ErrLeadingZero

					return token.None, token.None
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

					return token.None, token.None
				}

				start = b

				break NUMBER
			}
		}
	}

	if isFractional && fractionalPart == 0 {
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

	return l.lookAhead(token.MakeWithValue(token.Number, l.currentValue), start)
}
