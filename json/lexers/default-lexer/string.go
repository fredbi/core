package lexer

import (
	"errors"
	"io"
	"unicode/utf16"
	"unicode/utf8"

	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/lexers/token"
)

func (l *L) consumeString() token.T {
	var escapeSequence bool
	l.currentValue = l.currentValue[:0]

	for {
		if err := l.readMore(); err != nil {
			if errors.Is(err, io.EOF) {
				l.err = codes.ErrUnterminatedString
			} else {
				l.err = err
			}

			return token.None
		}

		for {
			if l.consumed >= l.bufferized {
				break
			}

			if l.maxValueBytes > 0 && len(l.currentValue) > l.maxValueBytes {
				l.err = codes.ErrMaxValueBytes

				return token.None
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
					l.current, l.next = l.expectColon(token.MakeWithValue(token.Key, l.currentValue))
					l.expectKey = false

					return l.current
				}

				return token.MakeWithValue(token.String, l.currentValue)

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
				r, err := l.unescapeUnicodeSequence()
				if err != nil {
					l.err = err

					return token.None
				}

				l.currentValue = utf8.AppendRune(l.currentValue, r)

			default:
				if escapeSequence {
					l.err = codes.ErrUnknownEscape

					return token.None
				}

				l.currentValue = append(l.currentValue, b)
			}
		}
	}
}

func (l *L) unescapeUnicodeSequence() (rune, error) {
	var buf [4]byte
	if err := l.consumeN(buf[:]); err != nil {
		return utf8.RuneError, codes.ErrUnicodeEscape
	}

	high1, highOK1 := unhex(buf[0])
	low1, lowOK1 := unhex(buf[1])
	high2, highOK2 := unhex(buf[2])
	low2, lowOK2 := unhex(buf[3])
	if !lowOK1 || !highOK1 || !lowOK2 || !highOK2 {
		return utf8.RuneError, codes.ErrUnicodeEscape
	}

	unicodeEscape := uint32(high1)<<12 + uint32(low1)<<8 + uint32(high2)<<4 + uint32(low2)
	r := rune(unicodeEscape)
	if utf16.IsSurrogate(r) {
		// this is a surrogate pair to encode a UTF-16 codepoint in 2 pairs
		// expect this to follow: \uXXXX
		var nextBuf [6]byte
		if err := l.consumeN(nextBuf[:]); err != nil {
			return utf8.RuneError, codes.ErrSurrogateEscape
		}

		if nextBuf[0] != escape || nextBuf[1] != 'u' {
			return utf8.RuneError, codes.ErrSurrogateEscape
		}

		high1, highOK1 = unhex(nextBuf[2])
		low1, lowOK1 = unhex(nextBuf[3])
		high2, highOK2 = unhex(nextBuf[4])
		low2, lowOK2 = unhex(nextBuf[5])
		if !lowOK1 || !highOK1 || !lowOK2 || !highOK2 {
			return utf8.RuneError, codes.ErrUnicodeEscape
		}

		unicodeEscape2 := uint32(high1)<<12 + uint32(low1)<<8 + uint32(high2)<<4 + uint32(low2)
		r1 := r
		r2 := rune(unicodeEscape2)

		r = utf16.DecodeRune(r1, r2)
	}

	if !utf8.ValidRune(r) {
		return utf8.RuneError, codes.ErrInvalidRune
	}

	return r, nil
}

func unhex(c byte) (byte, bool) {
	const asciiOffset = 10
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + asciiOffset, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + asciiOffset, true
	default:
		return 0, false
	}
}
