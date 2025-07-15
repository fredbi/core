package bufio

import (
	"unicode/utf8"
)

type EscapeFlags uint8

const (
	NoEscapeHTML EscapeFlags = 1 << iota
	EscapeHTML
)

const chars = "0123456789abcdef"

func isNotEscapedSingleChar(c byte, escapeHTML bool) bool {
	// Note: might make sense to use a table if there are more chars to escape. With 4 chars
	// it benchmarks the same.
	if escapeHTML {
		return c != '<' && c != '>' && c != '&' && c != '\\' && c != '"' && c >= 0x20 &&
			c < utf8.RuneSelf
	}
	return c != '\\' && c != '"' && c >= 0x20 && c < utf8.RuneSelf
}

func writeEscapedBytes[T string | []byte](
	data T,
	flags EscapeFlags,
	writeByte func(byte) error,
	writeBytes func([]byte) (int, error),
) (int, error) {
	buf := []byte(data)
	var written int
	// Portions of the string that contain no escapes are appended as
	// byte slices.
	p := 0 // last non-escape symbol

	for i := 0; i < len(buf); {
		c := buf[i]

		if isNotEscapedSingleChar(c, flags&EscapeHTML > 0) {
			// single-width character, no escaping is required
			i++

			continue
		}

		if c < utf8.RuneSelf {
			// single-width character, need to escape
			n, err := writeBytes(buf[p:i])
			if err != nil {
				return written, err
			}
			written += n

			switch c {
			case '\t':
				if err := writeByte('\\'); err != nil {
					return written, err
				}
				written++
				if err := writeByte('t'); err != nil {
					return written, err
				}
				written++

			case '\r':
				if err := writeByte('\\'); err != nil {
					return written, err
				}
				written++
				if err := writeByte('r'); err != nil {
					return written, err
				}
				written++
			case '\n':
				if err := writeByte('\\'); err != nil {
					return written, err
				}
				written++
				if err := writeByte('n'); err != nil {
					return written, err
				}
				written++
			case '\\':
				if err := writeByte('\\'); err != nil {
					return written, err
				}
				written++
				if err := writeByte('\\'); err != nil {
					return written, err
				}
				written++
			case '"':
				if err := writeByte('\\'); err != nil {
					return written, err
				}
				written++
				if err := writeByte('"'); err != nil {
					return written, err
				}
				written++
			default:
				n, err := writeBytes([]byte(`\u00`))
				if err != nil {
					return written, err
				}
				written++
				written += n
				if err := writeByte(chars[c>>4]); err != nil {
					return written, err
				}
				written++
				if err := writeByte(chars[c&0xf]); err != nil {
					return written, err
				}
				written++
			}

			i++
			p = i

			continue
		}

		// broken utf
		runeValue, runeWidth := utf8.DecodeRune(buf[i:])
		if runeValue == utf8.RuneError && runeWidth == 1 {
			n, err := writeBytes(buf[p:i])
			if err != nil {
				return written, err
			}
			written += n
			n, err = writeBytes([]byte(`\ufffd`))
			if err != nil {
				return written, err
			}
			written += n
			i++
			p = i

			continue
		}

		// jsonp stuff - tab separator and line separator
		if runeValue == '\u2028' || runeValue == '\u2029' {
			n, err := writeBytes(buf[p:i])
			if err != nil {
				return written, err
			}
			written += n
			n, err = writeBytes([]byte(`\u202`))
			if err != nil {
				return written, err
			}
			written += n
			if err = writeByte(chars[runeValue&0xf]); err != nil {
				return written, err
			}
			written++
			i += runeWidth
			p = i

			continue
		}

		i += runeWidth
	}

	n, err := writeBytes(buf[p:])
	if err != nil {
		return written, err
	}
	written += n

	return written, nil
}
