package token

import (
	"bytes"
	"unicode/utf16"
	"unicode/utf8"
)

// Unescaped returns the decoded form of a verbatim token's value.
//
// A verbatim token [VT] keeps a string/key value exactly as it appeared in the
// source (escape sequences intact — see the [String]/[Key] doc), so it can be
// round-tripped byte-for-byte. Unescaped expands the JSON escapes on demand:
// the shorthand escapes (\", \\, \/, \b, \f, \n, \r, \t) and \uXXXX sequences
// (surrogate pairs combined) become their UTF-8 bytes.
//
// For non-string tokens, and for strings that contain no escape, the raw value
// is returned unchanged with no allocation. Otherwise a fresh slice is returned.
//
// The escapes were validated when the token was scanned by the verbatim lexer,
// so decoding here cannot fail; a malformed sequence would have errored at scan
// time (any residual bad input is passed through rather than panicking).
func (t VT) Unescaped() []byte {
	v := t.value
	if (t.kind != String && t.kind != Key) || bytes.IndexByte(v, '\\') < 0 {
		return v
	}

	return unescape(v)
}

// UnescapedString is [VT.Unescaped] as a string. It always allocates (the string
// header cannot alias the token's buffer, which is reused on the next token).
func (t VT) UnescapedString() string {
	return string(t.Unescaped())
}

// unescape decodes a raw JSON string body (no surrounding quotes) whose escapes
// were already validated. It is the token-side counterpart of the lexer's
// unescaping scanner, kept standalone so [token] has no dependency on the lexer.
func unescape(raw []byte) []byte {
	// the decoded form is never longer than the raw form (every escape shrinks),
	// so one allocation of len(raw) suffices.
	out := make([]byte, 0, len(raw))

	for i := 0; i < len(raw); {
		c := raw[i]
		if c != '\\' {
			out = append(out, c)
			i++

			continue
		}

		// c == '\\'; a validated escape follows
		i++
		if i >= len(raw) {
			// unreachable on validated input; pass the lone backslash through
			out = append(out, '\\')

			break
		}
		switch raw[i] {
		case '"':
			out = append(out, '"')
			i++
		case '\\':
			out = append(out, '\\')
			i++
		case '/':
			out = append(out, '/')
			i++
		case 'b':
			out = append(out, '\b')
			i++
		case 'f':
			out = append(out, '\f')
			i++
		case 'n':
			out = append(out, '\n')
			i++
		case 'r':
			out = append(out, '\r')
			i++
		case 't':
			out = append(out, '\t')
			i++
		case 'u':
			r, next := decodeUnicode(raw, i+1)
			out = utf8.AppendRune(out, r)
			i = next
		default:
			// unreachable on validated input; pass through
			out = append(out, '\\', raw[i])
			i++
		}
	}

	return out
}

// decodeUnicode decodes a \uXXXX sequence (and a following \uXXXX low surrogate
// when the first is a high surrogate) starting at pos (the first hex digit, past
// the 'u'). It returns the rune and the index just past the consumed sequence.
// Input is assumed validated.
func decodeUnicode(raw []byte, pos int) (rune, int) {
	if pos+4 > len(raw) {
		return utf8.RuneError, len(raw)
	}
	r := rune(hex4(raw, pos))
	pos += 4

	if utf16.IsSurrogate(r) {
		// a validated high surrogate is followed by "\uYYYY"
		if pos+6 <= len(raw) && raw[pos] == '\\' && raw[pos+1] == 'u' {
			r2 := rune(hex4(raw, pos+2))
			if dec := utf16.DecodeRune(r, r2); dec != utf8.RuneError {
				return dec, pos + 6
			}
		}

		return utf8.RuneError, pos
	}

	return r, pos
}

// hex4 decodes 4 hex digits at pos into a uint16-range value. Input is assumed
// validated (each byte a hex digit).
func hex4(raw []byte, pos int) uint32 {
	return uint32(unhexDigit(raw[pos]))<<12 |
		uint32(unhexDigit(raw[pos+1]))<<8 |
		uint32(unhexDigit(raw[pos+2]))<<4 |
		uint32(unhexDigit(raw[pos+3]))
}

func unhexDigit(c byte) byte {
	const asciiOffset = 10
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + asciiOffset
	case 'A' <= c && c <= 'F':
		return c - 'A' + asciiOffset
	default:
		return 0
	}
}
