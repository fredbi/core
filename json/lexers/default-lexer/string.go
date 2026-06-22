package lexer

import (
	"errors"
	"io"
	"unicode/utf16"
	"unicode/utf8"

	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/lexers/token"
)

// consumeString scans a string value (the opening quote is already consumed).
//
// In whole-buffer mode it takes the fast path: a local-cursor scan that aliases
// the input for unescaped strings (zero copy) and falls back to copying only on
// the first escape. Streaming uses the buffer-refilling path.
func (l *L) consumeString() token.T {
	if l.wholeBuffer {
		return l.consumeStringWhole()
	}

	return l.consumeStringStreaming()
}

// consumeStringWhole scans a string when the whole input is in l.buffer. The
// cursor is a pure local; in whole-buffer mode l.offset always equals the buffer
// index, so it (and l.consumed) are written back only at exit points.
func (l *L) consumeStringWhole() token.T {
	data := l.buffer
	n := l.bufferized
	start := l.consumed // first content byte
	i := start

	// fast path: look for the closing quote or the first escape / control char
	for i < n {
		c := data[i]
		if c == doubleQuote {
			if l.maxValueBytes > 0 && i-start > l.maxValueBytes {
				l.consumed, l.offset = i, uint64(i)
				l.err = codes.ErrMaxValueBytes

				return token.None
			}
			value := data[start:i:i] // alias the input (cap == len)
			i++                      // past the closing quote
			l.consumed, l.offset = i, uint64(i)

			return l.finishStringValue(value)
		}
		if c == escape {
			break
		}
		if c < 0x20 {
			l.consumed, l.offset = i, uint64(i)
			l.err = codes.ErrControlChar

			return token.None
		}
		i++
	}
	if i >= n {
		l.consumed, l.offset = i, uint64(i)
		l.err = codes.ErrUnterminatedString

		return token.None
	}

	// slow path: an escape was found; copy the clean prefix then unescape the rest
	l.currentValue = append(l.currentValue[:0], data[start:i]...)

	for i < n {
		if l.maxValueBytes > 0 && len(l.currentValue) > l.maxValueBytes {
			l.consumed, l.offset = i, uint64(i)
			l.err = codes.ErrMaxValueBytes

			return token.None
		}

		c := data[i]
		switch {
		case c == doubleQuote:
			i++
			l.consumed, l.offset = i, uint64(i)

			return l.finishStringValue(l.currentValue)

		case c == escape:
			i++
			if i >= n {
				l.consumed, l.offset = i, uint64(i)
				l.err = codes.ErrUnterminatedString

				return token.None
			}
			switch data[i] {
			case doubleQuote:
				l.currentValue = append(l.currentValue, '"')
			case escape:
				l.currentValue = append(l.currentValue, '\\')
			case slash:
				l.currentValue = append(l.currentValue, '/')
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
			case 'u':
				// hand off to the surrogate-aware decoder, which reads from
				// l.consumed; offset==index lets us sync trivially
				l.consumed = i + 1 // past 'u'
				l.offset = uint64(l.consumed)
				r, err := l.unescapeUnicodeSequence()
				if err != nil {
					l.err = err

					return token.None
				}
				l.currentValue = utf8.AppendRune(l.currentValue, r)
				i = l.consumed
				continue
			default:
				l.consumed, l.offset = i, uint64(i)
				l.err = codes.ErrUnknownEscape

				return token.None
			}
			i++

		case c < 0x20:
			l.consumed, l.offset = i, uint64(i)
			l.err = codes.ErrControlChar

			return token.None

		default:
			l.currentValue = append(l.currentValue, c)
			i++
		}
	}

	l.consumed, l.offset = i, uint64(i)
	l.err = codes.ErrUnterminatedString

	return token.None
}

// finishStringValue turns a scanned string body into a Key (in object key
// position) or String token, handling the trailing colon for keys.
func (l *L) finishStringValue(value []byte) token.T {
	if l.expectKey {
		// the following colon is validated on the next scan (see l.afterKey)
		l.expectKey = false
		l.afterKey = true

		return token.MakeWithValue(token.Key, value)
	}

	return token.MakeWithValue(token.String, value)
}

func (l *L) consumeStringStreaming() token.T {
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

		for l.consumed < l.bufferized {

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

				return l.finishStringValue(l.currentValue)

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

				if b < 0x20 {
					// RFC 8259: control characters U+0000..U+001F must be escaped
					l.err = codes.ErrControlChar

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
