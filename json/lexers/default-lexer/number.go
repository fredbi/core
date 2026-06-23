package lexer

import (
	"errors"
	"io"

	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/lexers/token"
)

// consumeNumberWhole scans a JSON number in whole-buffer mode using tight
// digit-run loops, mirroring jsontext's ConsumeNumberResumable: the grammar is
// validated only at the transitions between runs (sign, integer, fraction,
// exponent), not on every byte. The hot loops use uint() index comparisons so
// the bounds check is elided, and the value aliases the input buffer.
//
// In whole-buffer mode there are no refills, so l.offset == l.consumed
// throughout; both are written back once from the local cursor n. Plain
// integers are handled inline in scanToken and never reach this path; it scans
// fractions, exponents and the leading-zero / malformed forms.
//
// Like the rest of the folded-look-ahead design, a malformed number may be
// surfaced as a shorter valid value with the error deferred to the next token
// (e.g. "1.2.3" -> "1.2" then a rejected ".3"); the document is still rejected.
func (l *L) consumeNumberWhole(start byte) token.T {
	buf := l.buffer[:l.bufferized]
	numStart := l.consumed - 1
	n := l.consumed // index just past start

	fail := func(code error) token.T {
		l.consumed = n
		l.offset = uint64(n)
		l.err = code

		return token.None
	}

	// integer part: optional '-', then '0' alone or [1-9][0-9]*
	if start == minusSign {
		if uint(n) >= uint(len(buf)) {
			return fail(codes.ErrMissingInteger)
		}
		start = buf[n]
		n++
	}

	switch {
	case start == '0':
		// a leading zero is only valid as the lone integer digit "0"
		if uint(n) < uint(len(buf)) && buf[n] >= '0' && buf[n] <= '9' {
			return fail(codes.ErrLeadingZero)
		}
	case start >= '1' && start <= '9':
		for uint(n) < uint(len(buf)) && buf[n] >= '0' && buf[n] <= '9' {
			n++
		}
	default: // start is '.' (or otherwise not a digit): missing integer part
		return fail(codes.ErrMissingInteger)
	}

	// fractional part: '.' 1*digit
	if uint(n) < uint(len(buf)) && buf[n] == decimalPoint {
		n++
		if uint(n) >= uint(len(buf)) || buf[n] < '0' || buf[n] > '9' {
			return fail(codes.ErrInvalidFractional)
		}
		for uint(n) < uint(len(buf)) && buf[n] >= '0' && buf[n] <= '9' {
			n++
		}
	}

	// exponent part: ('e'|'E') ['+'|'-'] 1*digit
	if uint(n) < uint(len(buf)) && (buf[n] == 'e' || buf[n] == 'E') {
		n++
		if uint(n) < uint(len(buf)) && (buf[n] == '+' || buf[n] == '-') {
			n++
		}
		if uint(n) >= uint(len(buf)) || buf[n] < '0' || buf[n] > '9' {
			return fail(codes.ErrInvalidExponent)
		}
		for uint(n) < uint(len(buf)) && buf[n] >= '0' && buf[n] <= '9' {
			n++
		}
	}

	// n stops at the terminator (or end of input); it is left unconsumed, so the
	// next scan validates it via the standard start-of-token checks.
	l.consumed = n
	l.offset = uint64(n)

	return token.MakeWithValue(token.Number, l.buffer[numStart:n:n])
}

// consumeNumberStreaming consumes a JSON number byte-by-byte. It is the general
// path used for streaming input (refillable buffer) and when a value-size cap is
// active; the whole-buffer fast paths handle the common bytes-mode case.
//
// start is the previously consumed byte that decided to parse a number.
func (l *L) consumeNumberStreaming(start byte) token.T {
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

				return token.None
			}

			b := l.buffer[l.consumed]
			l.consumed++
			l.offset++

			switch {
			case b == decimalPoint:
				if hasFractional || isExponent {
					// only 1 decimal separator allowed, exponent is integer
					l.err = codes.ErrRepeatedDecimalSeparator

					return token.None
				}

				hasFractional = true
				isFractional = true

			case b == '+' || b == '-':
				if !isExponent || exponentPart > 0 || exponentSign {
					// a sign is only valid right after the exponent marker,
					// before any exponent digit and only once
					l.err = codes.ErrInvalidSign

					return token.None
				}
				exponentSign = true

			case b == 'e' || b == 'E':
				if isExponent {
					l.err = codes.ErrRepeatedExponent

					return token.None
				}

				isExponent = true
				isFractional = false

			case b == '0':
				if hasLeadingZero && !isFractional && !isExponent {
					// no leading zeroes on integer part, unless this is just 0
					l.err = codes.ErrLeadingZero

					return token.None
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

					return token.None
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

					return token.None
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

			return token.None
		}

		numStart = 0 // the buffer was refilled: the pending segment restarts at 0
	}

	if hasFractional && fractionalPart == 0 {
		// a decimal point must be followed by at least one fractional digit
		l.err = codes.ErrInvalidFractional
		return token.None
	}

	if isExponent && exponentPart == 0 {
		l.err = codes.ErrInvalidExponent

		return token.None
	}

	if hasLeadingZero && integerPart > 1 {
		l.err = codes.ErrLeadingZero

		return token.None
	}

	if integerPart == 0 {
		l.err = codes.ErrMissingInteger

		return token.None
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

	return token.MakeWithValue(token.Number, value)
}
