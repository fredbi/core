package writer

import (
	"io"
	"unicode/utf8"
)

// Bool writes a boolean value.
func (w *W) Bool(v bool) {
	w.buffer.EnsureSpace(5)
	if v {
		w.buffer.Buf = append(w.buffer.Buf, "true"...)
	} else {
		w.buffer.Buf = append(w.buffer.Buf, "false"...)
	}
}

// Raw appends raw data to the buffer
func (w *W) Raw(data []byte) {
	if data == nil {
		w.buffer.AppendString(null)
		return
	}

	w.buffer.AppendBytes(data)
}

func (w *W) StringBytes(data []byte) {
	if data == nil {
		w.buffer.AppendString(null)
		return
	}
	// TODO
}

func (w *W) StringRunes(data []rune) {
	// TODO
}

func (w *W) NumberBytes(data []byte) {
	w.Raw(data)
}

func (w *W) StringCopy(r io.Reader) {
	// TODO
}

func (w *W) NumberCopy(r io.Reader) {
	// TODO
}

func (w *W) RawCopy(r io.Reader) {
	// TODO
}

// String writes a JSON string value enclosed by double quotes.
func (w *W) String(s string) {
	w.buffer.AppendByte('"')

	// Portions of the string that contain no escapes are appended as
	// byte slices.

	p := 0 // last non-escape symbol

	for i := 0; i < len(s); {
		c := s[i]

		if isNotEscapedSingleChar(c, !w.noEscapeHTML) {
			// single-width character, no escaping is required
			i++
			continue
		} else if c < utf8.RuneSelf {
			// single-with character, need to escape
			w.buffer.AppendString(s[p:i])
			switch c {
			case '\t':
				w.buffer.AppendString(`\t`)
			case '\r':
				w.buffer.AppendString(`\r`)
			case '\n':
				w.buffer.AppendString(`\n`)
			case '\\':
				w.buffer.AppendString(`\\`)
			case '"':
				w.buffer.AppendString(`\"`)
			default:
				w.buffer.AppendString(`\u00`)
				w.buffer.AppendByte(chars[c>>4])
				w.buffer.AppendByte(chars[c&0xf])
			}

			i++
			p = i
			continue
		}

		// broken utf
		runeValue, runeWidth := utf8.DecodeRuneInString(s[i:])
		if runeValue == utf8.RuneError && runeWidth == 1 {
			w.buffer.AppendString(s[p:i])
			w.buffer.AppendString(`\ufffd`)
			i++
			p = i
			continue
		}

		// jsonp stuff - tab separator and line separator
		if runeValue == '\u2028' || runeValue == '\u2029' {
			w.buffer.AppendString(s[p:i])
			w.buffer.AppendString(`\u202`)
			w.buffer.AppendByte(chars[runeValue&0xf])
			i += runeWidth
			p = i
			continue
		}
		i += runeWidth
	}
	w.buffer.AppendString(s[p:])
	w.buffer.AppendByte('"')
}
