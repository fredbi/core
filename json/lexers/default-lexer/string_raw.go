package lexer

import (
	"encoding/binary"
	"errors"
	"io"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/fredbi/core/json/lexers/default-lexer/internal/strscan"
	"github.com/fredbi/core/json/lexers/default-lexer/internal/swar"
	codes "github.com/fredbi/core/json/lexers/error-codes"
	scan "github.com/fredbi/core/json/lexers/internal/scan"
	"github.com/fredbi/core/json/lexers/token"
)

// The raw string scanners back the VERBATIM lexer. Unlike consumeString*, they do
// NOT decode escapes: they validate the whole string grammar (every escape, so a
// later decode via token.VT.Unescaped cannot fail) but keep the RAW source bytes.
// This is both the faithful round-trip contract for [VT] and strictly less work
// than decoding (no output materialization; whole-buffer aliases with zero copy).
// consumeString dispatches here when l.trackBlanks is set (see string.go).

// consumeStringRawWhole is the whole-buffer raw scan. The clean-prefix fast path
// is identical to consumeStringWhole (SWAR to the first stop); the difference is
// only in how an escape is handled — validated in place, never decoded — so the
// returned value always aliases the raw input.
func (l *L) consumeStringRawWhole() token.T {
	data := l.buffer
	n := l.bufferized
	start := l.consumed

	i := start
	guard := start + guessLong
	if l.noAVX2 {
		guard = n + 1 // WithoutAVX2: never delegate, pure inline SWAR (see consumeStringWhole)
	}
	for i+8 <= n {
		if m := swar.StringStopMask(binary.LittleEndian.Uint64(data[i:])); m != 0 {
			i += swar.FirstByte(m)

			break
		}
		i += 8
		if i >= guard {
			break // guessLong clean bytes in — delegate below (call kept out of the loop)
		}
	}
	// clean past guessLong → guess long, AVX2 scan of the rest (same heuristic and
	// out-of-loop placement as consumeStringWhole).
	if i >= guard && i+8 <= n {
		if c := data[i]; c != doubleQuote && c != escape && c >= 0x20 {
			i += strscan.ScanStop(data[i:n])
		}
	}
	for ; i < n; i++ {
		if c := data[i]; c == doubleQuote || c == escape || c < 0x20 {
			break
		}
	}
	if i >= n {
		l.consumed, l.offset = i, uint64(i)
		l.err = codes.ErrUnterminatedString

		return token.None
	}

	switch c := data[i]; {
	case c == doubleQuote:
		// no escapes: raw == decoded, same aliasing exit as consumeStringWhole
		if l.maxValueBytes > 0 && i-start > l.maxValueBytes {
			l.consumed, l.offset = i, uint64(i)
			l.err = codes.ErrMaxValueBytes

			return token.None
		}
		value := data[start:i:i]
		i++
		l.consumed, l.offset = i, uint64(i)

		return l.finishStringValue(value)

	case c < 0x20:
		l.consumed, l.offset = i, uint64(i)
		l.err = codes.ErrControlChar

		return token.None
	}

	// an escape was found at i: validate the rest but keep the raw bytes.
	return l.consumeStringRawEscaped(start, i)
}

// consumeStringRawEscaped validates a string that contains at least one escape
// (data[i] == escape) without decoding it, and returns the raw content aliased
// from the input. Clean runs between escapes are skipped with the same adaptive
// scalar-probe-then-SWAR scan the decoder uses (consumeStringEscaped), but with
// no copying — so a sparse-escape string with a long clean tail stays fast.
func (l *L) consumeStringRawEscaped(start, i int) token.T {
	data := l.buffer
	n := l.bufferized

	for i < n {
		switch c := data[i]; {
		case c == doubleQuote:
			if l.maxValueBytes > 0 && i-start > l.maxValueBytes {
				l.consumed, l.offset = i, uint64(i)
				l.err = codes.ErrMaxValueBytes

				return token.None
			}
			value := data[start:i:i]
			i++
			l.consumed, l.offset = i, uint64(i)

			return l.finishStringValue(value)

		case c == escape:
			i++
			if i >= n {
				l.consumed, l.offset = i, uint64(i)
				l.err = codes.ErrUnterminatedString

				return token.None
			}
			switch data[i] {
			case doubleQuote, escape, slash, 'b', 'f', 'n', 'r', 't':
				i++
			case 'u':
				next, err := validateUnicodeWhole(data, i+1, n)
				if err != nil {
					l.consumed, l.offset = i, uint64(i)
					l.err = err

					return token.None
				}
				i = next
			default:
				l.consumed, l.offset = i, uint64(i)
				l.err = codes.ErrUnknownEscape

				return token.None
			}

		case c < 0x20:
			l.consumed, l.offset = i, uint64(i)
			l.err = codes.ErrControlChar

			return token.None

		default:
			// skip the clean run to the next stop (quote/escape/control) without
			// copying — adaptive scalar probe then SWAR (see consumeStringEscaped).
			run := i
			stop := i + 1
			probe := min(stop+swarProbe, n)
			for ; stop < probe; stop++ {
				if b := data[stop]; b == doubleQuote || b == escape || b < 0x20 {
					break
				}
			}
			if stop == probe && stop < n { // outran the scalar probe → SWAR, guess long past guessLong
				for stop+8 <= n {
					if m := swar.StringStopMask(binary.LittleEndian.Uint64(data[stop:])); m != 0 {
						stop += swar.FirstByte(m)

						break
					}
					stop += 8
					if stop-run >= guessLong && !l.noAVX2 {
						stop += strscan.ScanStop(data[stop:n])

						break
					}
				}
				for ; stop < n; stop++ {
					if b := data[stop]; b == doubleQuote || b == escape || b < 0x20 {
						break
					}
				}
			}
			i = stop
		}
	}

	l.consumed, l.offset = i, uint64(i)
	l.err = codes.ErrUnterminatedString

	return token.None
}

// consumeStringRawStreamFast is the raw streaming string fast path (§10.5c): the raw
// analogue of consumeStringStreamFast. It treats the current window like whole-buffer
// mode — SWAR/AVX2-scans for the closing quote / escape / control — and, when a clean
// string completes inside the window, ALIASES l.buffer zero-copy (a clean raw string
// IS its own value, no copy into currentValue), the common case. It hands off to the
// byte-by-byte consumeStringRawStreaming only on an escape or a value that spans a
// refill. This is what closes the verbatim reader-mode string gap (VS/VL used the
// byte-by-byte path unconditionally; L had this fast path since Phase 1a).
//
// Like consumeStringStreamFast, reaching l.bufferized is NOT end-of-input (delegate,
// don't error) and advances are RELATIVE (l.offset is the absolute stream offset,
// l.consumed the window index).
func (l *L) consumeStringRawStreamFast() token.T {
	data := l.buffer
	n := l.bufferized
	start := l.consumed // first content byte (opening quote already consumed)

	i := start
	guard := start + guessLong
	if l.noAVX2 {
		guard = n + 1
	}
	for i+8 <= n {
		if m := swar.StringStopMask(binary.LittleEndian.Uint64(data[i:])); m != 0 {
			i += swar.FirstByte(m)

			break
		}
		i += 8
		if i >= guard {
			break
		}
	}
	if i >= guard && i+8 <= n {
		if c := data[i]; c != doubleQuote && c != escape && c >= 0x20 {
			i += strscan.ScanStop(data[i:n])
		}
	}
	for ; i < n; i++ {
		if c := data[i]; c == doubleQuote || c == escape || c < 0x20 {
			break
		}
	}

	if i >= n {
		// window end reached without a stop byte: the string may span a refill →
		// hand off to the byte-by-byte raw path (l.consumed still start, re-scans clean).
		return l.consumeStringRawStreaming()
	}

	switch c := data[i]; {
	case c == doubleQuote:
		if l.maxValueBytes > 0 && i-start > l.maxValueBytes {
			l.offset += uint64(i - start)
			l.consumed = i
			l.err = codes.ErrMaxValueBytes

			return token.None
		}
		value := data[start:i:i] // alias the window (raw == value; valid until refill)
		end := i + 1             // past the closing quote
		l.offset += uint64(end - start)
		l.consumed = end

		return l.finishStringValue(value)

	case c < 0x20:
		l.offset += uint64(i - start)
		l.consumed = i
		l.err = codes.ErrControlChar

		return token.None
	}

	// an escape was found inside the window: delegate to the byte-by-byte raw path,
	// which keeps escapes verbatim and handles refills (re-scans from l.consumed==start).
	return l.consumeStringRawStreaming()
}

// consumeStringRawStreaming is the raw scan over a refilling buffer: it copies the
// source bytes verbatim (escapes intact) into l.currentValue while validating
// them, so the value survives buffer turnover.
func (l *L) consumeStringRawStreaming() token.T {
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
			l.consumed++
			l.offset++

			switch {
			case b == doubleQuote:
				return l.finishStringValue(l.currentValue)

			case b == escape:
				l.currentValue = append(l.currentValue, escape)
				if err := l.rawEscapeStreaming(); err != nil {
					l.err = err

					return token.None
				}

			case b < 0x20:
				l.err = codes.ErrControlChar

				return token.None

			default:
				l.currentValue = append(l.currentValue, b)

				// bulk-scan the rest of this clean run within the current window (§10.5c,
				// the raw analogue of consumeStringStreaming's Phase-1c copy): a long clean
				// stretch — between escapes, or a clean string that spilled past the fast
				// path's window — is copied in ONE shot (adaptive scalar-probe → SWAR →
				// AVX2) instead of one byte per loop turn. Raw copies verbatim, so there is
				// no decode; a run that reaches the window end stops at bufferized and the
				// outer loop refills for the continuation. This is what closes gsoc-style
				// escaped-long-string reader mode for VS/VL.
				data := l.buffer
				n := l.bufferized
				runStart := l.consumed
				stop := runStart
				probe := min(stop+swarProbe, n)
				for ; stop < probe; stop++ {
					if c := data[stop]; c == doubleQuote || c == escape || c < 0x20 {
						break
					}
				}
				if stop == probe && stop < n { // run outran the scalar probe → SWAR/AVX2
					for stop+8 <= n {
						if m := swar.StringStopMask(binary.LittleEndian.Uint64(data[stop:])); m != 0 {
							stop += swar.FirstByte(m)

							break
						}
						stop += 8
						if stop-runStart >= guessLong && !l.noAVX2 {
							stop += strscan.ScanStop(data[stop:n])

							break
						}
					}
					for ; stop < n; stop++ {
						if c := data[stop]; c == doubleQuote || c == escape || c < 0x20 {
							break
						}
					}
				}
				if stop > runStart {
					if l.maxValueBytes > 0 && len(l.currentValue)+(stop-runStart) > l.maxValueBytes {
						l.err = codes.ErrMaxValueBytes

						return token.None
					}
					l.currentValue = append(l.currentValue, data[runStart:stop]...)
					l.offset += uint64(stop - runStart)
					l.consumed = stop
				}
			}
		}
	}
}

// rawEscapeStreaming validates the escape sequence following a backslash (already
// appended) in the streaming raw scan, appending its raw bytes to l.currentValue.
func (l *L) rawEscapeStreaming() error {
	var one [1]byte
	if err := l.consumeN(one[:]); err != nil {
		return codes.ErrUnterminatedString
	}
	e := one[0]
	l.currentValue = append(l.currentValue, e)

	switch e {
	case doubleQuote, escape, slash, 'b', 'f', 'n', 'r', 't':
		return nil
	case 'u':
		return l.rawUnicodeStreaming()
	default:
		return codes.ErrUnknownEscape
	}
}

// rawUnicodeStreaming validates a \uXXXX sequence (and a following \uYYYY low
// surrogate for a high surrogate) in the streaming raw scan, appending the raw
// hex bytes verbatim. The leading "\u" has already been appended.
func (l *L) rawUnicodeStreaming() error {
	var buf [4]byte
	if err := l.consumeN(buf[:]); err != nil {
		return codes.ErrUnicodeEscape
	}
	code, ok := scan.Hex4(buf[0], buf[1], buf[2], buf[3])
	if !ok {
		return codes.ErrUnicodeEscape
	}
	l.currentValue = append(l.currentValue, buf[:]...)

	r := rune(code)
	if utf16.IsSurrogate(r) {
		var nb [6]byte // "\uYYYY"
		if err := l.consumeN(nb[:]); err != nil {
			return codes.ErrSurrogateEscape
		}
		if nb[0] != escape || nb[1] != 'u' {
			return codes.ErrSurrogateEscape
		}
		code2, ok2 := scan.Hex4(nb[2], nb[3], nb[4], nb[5])
		if !ok2 {
			return codes.ErrUnicodeEscape
		}
		if utf16.DecodeRune(r, rune(code2)) == utf8.RuneError {
			return codes.ErrSurrogateEscape
		}
		l.currentValue = append(l.currentValue, nb[:]...)
	} else if !utf8.ValidRune(r) {
		return codes.ErrInvalidRune
	}

	return nil
}

// validateUnicodeWhole validates a \uXXXX sequence at pos (first hex digit, past
// the 'u') in a whole buffer, following one surrogate pair when present. It
// returns the index just past the validated sequence, or an error.
func validateUnicodeWhole(data []byte, pos, n int) (int, error) {
	if pos+4 > n {
		return pos, codes.ErrUnicodeEscape
	}
	code, ok := scan.Hex4(data[pos], data[pos+1], data[pos+2], data[pos+3])
	if !ok {
		return pos, codes.ErrUnicodeEscape
	}
	pos += 4

	r := rune(code)
	if utf16.IsSurrogate(r) {
		if pos+6 > n || data[pos] != escape || data[pos+1] != 'u' {
			return pos, codes.ErrSurrogateEscape
		}
		code2, ok2 := scan.Hex4(data[pos+2], data[pos+3], data[pos+4], data[pos+5])
		if !ok2 {
			return pos, codes.ErrUnicodeEscape
		}
		if utf16.DecodeRune(r, rune(code2)) == utf8.RuneError {
			return pos, codes.ErrSurrogateEscape
		}
		pos += 6
	} else if !utf8.ValidRune(r) {
		return pos, codes.ErrInvalidRune
	}

	return pos, nil
}
