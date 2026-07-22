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

// swarProbe is the scalar look-ahead (bytes) the unescape slow path scans before
// switching a clean run to SWAR. Runs shorter than this (the escape-dense case)
// resolve scalar with no SWAR overhead; longer runs (sparse escapes + clean tail)
// switch to the word-at-a-time scan. One word keeps dense strings cheap.
const swarProbe = 8

// guessLong is the number of clean leading bytes after which the whole-buffer
// string scan stops probing inline (8-byte SWAR words) and hands the rest to the
// AVX2-gated strscan.ScanStop — Fred's "guess long strings" heuristic. It is the
// real long-string signal: strscan.ScanStop receives the buffer remainder (huge
// mid-document), so its own length guard cannot tell a short value from a long
// one; only the count of clean bytes already seen can. Short/medium values (object
// keys, most citm strings) resolve entirely inline and never pay the (non-inlinable)
// call; only genuinely long values, where AVX2's 32-bytes/iter pays off, delegate.
//
// 16 is the measured sweet spot across the full corpus (plan §9.3): it maximises
// geometric-mean throughput (+5.8%) while keeping the short-string workloads that
// gain nothing from AVX2 (citm, instruments, apache) at or above baseline — the
// larger win at 8 came with a real regression there. Must be a multiple of 8.
const guessLong = 16

// consumeString scans a string value (the opening quote is already consumed).
//
// In whole-buffer mode it takes the fast path: a local-cursor scan that aliases
// the input for unescaped strings (zero copy) and falls back to copying only on
// the first escape. Streaming uses the buffer-refilling path.
//
// The verbatim lexer (flagged by trackBlanks, set only for VL) keeps strings RAW
// — escapes intact for faithful round-tripping, decoded on demand via
// token.VT.Unescaped — so it routes to the validate-but-don't-decode scanners.
// The choice lives here, not in the shared scan core: that keeps both cores
// calling l.consumeString() directly, so the semantic core's codegen is
// unchanged (plan §9.1 — routing this through the policy or an inline branch in
// the core perturbed the semantic path's escape analysis, costing an alloc).
func (l *L) consumeString() token.T {
	if l.trackBlanks {
		if l.in.wholeBuffer {
			return l.consumeStringRawWhole()
		}

		return l.consumeStringRawStreamFast()
	}

	if l.in.wholeBuffer {
		return l.consumeStringWhole()
	}

	return l.consumeStringStreamFast()
}

// consumeStringStreamFast is the streaming string fast path (§10.3 Phase 1). It
// treats the CURRENT buffer window l.in.buffer[:l.in.bufferized] like whole-buffer mode:
// it scans for the closing quote with the shared SWAR/AVX2 string-stop scanner and,
// when a clean string completes inside the window, ALIASES the buffer zero-copy —
// the common case (token << window). It hands off to the byte-by-byte
// consumeStringStreaming (which refills and unescapes) only when the value actually
// needs the slow path: an escape, or the scan reaches the window end (the string may
// span a refill). Aliasing is valid until the next refill — the token's contractual
// lifespan (Fred: within the reuse contract).
//
// Unlike consumeStringWhole, reaching l.in.bufferized is NOT end-of-input, so it must
// delegate there rather than report ErrUnterminatedString; and because streaming
// keeps l.in.offset as the absolute stream offset (l.in.consumed is the window index),
// advances are RELATIVE deltas, not absolute assignments.
func (l *L) consumeStringStreamFast() token.T {
	data := l.in.buffer
	n := l.in.bufferized
	start := l.in.consumed // first content byte (opening quote already consumed)

	// jump to the first stop byte (closing quote, escape, or control), 8 bytes at a
	// time; delegate to the AVX2 scan once a run stays clean past guessLong. Identical
	// probe to consumeStringWhole, but bounded by the window end n = l.in.bufferized.
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
		// window end reached without a stop byte: the string may continue past a
		// refill boundary → hand off to the refilling byte-by-byte path (l.in.consumed
		// is still start, so it re-scans the clean prefix and continues correctly).
		return l.consumeStringStreaming()
	}

	switch c := data[i]; {
	case c == doubleQuote:
		if l.maxValueBytes > 0 && i-start > l.maxValueBytes {
			l.in.offset += uint64(i - start)
			l.in.consumed = i
			l.in.err = codes.ErrMaxValueBytes

			return token.None
		}
		value := data[start:i:i] // alias the window (valid until next refill)
		end := i + 1             // past the closing quote
		l.in.offset += uint64(end - start)
		l.in.consumed = end

		return l.finishStringValue(value)

	case c < 0x20:
		l.in.offset += uint64(i - start)
		l.in.consumed = i
		l.in.err = codes.ErrControlChar

		return token.None
	}

	// an escape was found inside the window: delegate to the streaming unescape path
	// (re-scans from l.in.consumed == start; handles escapes + any refill).
	return l.consumeStringStreaming()
}

// consumeStringWhole scans a string when the whole input is in l.in.buffer. The
// cursor is a pure local; in whole-buffer mode l.in.offset always equals the buffer
// index, so it (and l.in.consumed) are written back only at exit points.
func (l *L) consumeStringWhole() token.T {
	data := l.in.buffer
	n := l.in.bufferized
	start := l.in.consumed // first content byte

	// fast path: jump to the first byte that needs attention — the closing
	// quote, an escape, or a control char — scanning 8 bytes at a time with the
	// shared SWAR string-stop mask (swar.StringStopMask inlines, so there is no
	// call per word; see internal/swar). FirstByte locates the exact stop within
	// the matching word; the scalar tail handles the final < 8 bytes. The
	// overwhelmingly common case (no escapes, no control chars) aliases the input
	// with zero copy.
	i := start
	// guard is where the inline probe stops and delegates to the AVX2 scan. With
	// WithoutAVX2 it is pushed past the buffer so the loop never breaks to delegate
	// — the string is scanned entirely by the inline SWAR word loop (the pre-AVX2
	// baseline), no vector call at all.
	guard := start + guessLong
	if l.noAVX2 {
		guard = n + 1
	}
	for i+8 <= n {
		if m := swar.StringStopMask(binary.LittleEndian.Uint64(data[i:])); m != 0 {
			i += swar.FirstByte(m) // exact stop lane; skips the scalar re-scan

			break
		}
		i += 8
		if i >= guard {
			break // guessLong clean bytes in — leave the loop to delegate below
		}
	}
	// If the run stayed clean past guessLong (not stopped) and the buffer holds
	// more, guess this is a long value and hand the rest to the AVX2 scan. The call
	// lives OUTSIDE the word loop on purpose: a call in the loop body pessimizes its
	// register allocation for every short string that never reaches it (plan §9.1),
	// so short-string workloads must keep the tight, call-free loop above. i lands
	// on the stop byte or on n.
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
		l.in.consumed, l.in.offset = i, uint64(i)
		l.in.err = codes.ErrUnterminatedString

		return token.None
	}

	switch c := data[i]; {
	case c == doubleQuote:
		if l.maxValueBytes > 0 && i-start > l.maxValueBytes {
			l.in.consumed, l.in.offset = i, uint64(i)
			l.in.err = codes.ErrMaxValueBytes

			return token.None
		}
		value := data[start:i:i] // alias the input (cap == len)
		i++                      // past the closing quote
		l.in.consumed, l.in.offset = i, uint64(i)

		return l.finishStringValue(value)

	case c < 0x20:
		l.in.consumed, l.in.offset = i, uint64(i)
		l.in.err = codes.ErrControlChar

		return token.None
	}

	// an escape was found at i: hand off to the unescape slow path. It is a
	// separate function on purpose — keeping the byte-by-byte escape machinery out
	// of this frame insulates the fast path's codegen from it (and vice versa);
	// they were previously one function, where a fast-path change could regress
	// the slow path by ~12% and vice versa (plan §4.2).
	return l.consumeStringEscaped(start, i)
}

// consumeStringEscaped is the unescape slow path, split out of consumeStringWhole.
// It is entered with data[i] == escape and start..i the clean prefix already
// scanned. It copies that prefix then unescapes the rest; the loop invariant is
// that data[i] is the next "stop" byte (quote, escape, or control) — clean runs
// between stops are copied in bulk rather than byte-by-byte.
func (l *L) consumeStringEscaped(start, i int) token.T {
	data := l.in.buffer
	n := l.in.bufferized

	l.in.currentValue = append(l.in.currentValue[:0], data[start:i]...)

	for i < n {
		switch c := data[i]; {
		case c == doubleQuote:
			i++
			l.in.consumed, l.in.offset = i, uint64(i)

			return l.finishStringValue(l.in.currentValue)

		case c == escape:
			i++
			if i >= n {
				l.in.consumed, l.in.offset = i, uint64(i)
				l.in.err = codes.ErrUnterminatedString

				return token.None
			}
			switch data[i] {
			case doubleQuote:
				l.in.currentValue = append(l.in.currentValue, '"')
				i++
			case escape:
				l.in.currentValue = append(l.in.currentValue, '\\')
				i++
			case slash:
				l.in.currentValue = append(l.in.currentValue, '/')
				i++
			case 'b':
				l.in.currentValue = append(l.in.currentValue, '\b')
				i++
			case 'f':
				l.in.currentValue = append(l.in.currentValue, '\f')
				i++
			case 'n':
				l.in.currentValue = append(l.in.currentValue, '\n')
				i++
			case 't':
				l.in.currentValue = append(l.in.currentValue, '\t')
				i++
			case 'r':
				l.in.currentValue = append(l.in.currentValue, '\r')
				i++
			case 'u':
				// hand off to the surrogate-aware decoder, which reads from
				// l.in.consumed; offset==index lets us sync trivially
				l.in.consumed = i + 1 // past 'u'
				l.in.offset = uint64(l.in.consumed)
				r, err := l.unescapeUnicodeSequence()
				if err != nil {
					l.in.err = err

					return token.None
				}
				l.in.currentValue = utf8.AppendRune(l.in.currentValue, r)
				i = l.in.consumed
			default:
				l.in.consumed, l.in.offset = i, uint64(i)
				l.in.err = codes.ErrUnknownEscape

				return token.None
			}

		case c < 0x20:
			l.in.consumed, l.in.offset = i, uint64(i)
			l.in.err = codes.ErrControlChar

			return token.None
		}

		// Scan the clean run after the escape to the next stop byte, then bulk-append
		// it. Adaptive scan (escapes are usually sparse): start scalar — in
		// escape-dense strings the runs are tiny and a SWAR word-load would cost more
		// than it saves — but once a run proves longer than a word, bet the rest of
		// the string is mostly clean and finish the run with SWAR. This keeps the
		// dense case cheap while making the long-clean-tail case (sparse escapes + a
		// long unescaped trail) fast. The bound is checked against len + the run
		// width *before* the append so an over-long value is rejected without copying
		// a huge clean run, and escape-only expansion (zero-width run) is still caught.
		stop := i
		probe := min(i+swarProbe, n)
		for ; stop < probe; stop++ {
			if c := data[stop]; c == doubleQuote || c == escape || c < 0x20 {
				break
			}
		}
		if stop == probe && stop < n { // run outran the scalar probe → SWAR, guess long past guessLong
			for stop+8 <= n {
				if m := swar.StringStopMask(binary.LittleEndian.Uint64(data[stop:])); m != 0 {
					stop += swar.FirstByte(m)

					break
				}
				stop += 8
				if stop-i >= guessLong && !l.noAVX2 {
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
		if l.maxValueBytes > 0 && len(l.in.currentValue)+(stop-i) > l.maxValueBytes {
			l.in.consumed, l.in.offset = i, uint64(i)
			l.in.err = codes.ErrMaxValueBytes

			return token.None
		}
		l.in.currentValue = append(l.in.currentValue, data[i:stop]...)
		i = stop
	}

	l.in.consumed, l.in.offset = i, uint64(i)
	l.in.err = codes.ErrUnterminatedString

	return token.None
}

// finishStringValue turns a scanned string body into a Key (in object key
// position) or String token, handling the trailing colon for keys.
func (l *L) finishStringValue(value []byte) token.T {
	if l.in.expectKey {
		// the following colon is validated on the next scan (see l.in.afterKey)
		l.in.expectKey = false
		l.in.afterKey = true

		return token.MakeWithValue(token.Key, value)
	}

	return token.MakeWithValue(token.String, value)
}

func (l *L) consumeStringStreaming() token.T {
	var escapeSequence bool
	l.in.currentValue = l.in.currentValue[:0]

	for {
		if err := l.in.readMore(); err != nil {
			if errors.Is(err, io.EOF) {
				l.in.err = codes.ErrUnterminatedString
			} else {
				l.in.err = err
			}

			return token.None
		}

		for l.in.consumed < l.in.bufferized {

			if l.maxValueBytes > 0 && len(l.in.currentValue) > l.maxValueBytes {
				l.in.err = codes.ErrMaxValueBytes

				return token.None
			}

			b := l.in.buffer[l.in.consumed]
			l.in.offset++
			l.in.consumed++

			switch b {
			case escape:
				if escapeSequence {
					//  "\\"
					l.in.currentValue = append(l.in.currentValue, b)
					escapeSequence = false

					continue
				}

				escapeSequence = true

			case doubleQuote:
				if escapeSequence {
					//  "\""
					escapeSequence = false
					l.in.currentValue = append(l.in.currentValue, b)

					continue
				}

				return l.finishStringValue(l.in.currentValue)

			case slash:
				if escapeSequence {
					// "\/"
					escapeSequence = false
				}

				l.in.currentValue = append(l.in.currentValue, b)

			case 'b', 'f', 'n', 't', 'r':
				if !escapeSequence {
					l.in.currentValue = append(l.in.currentValue, b)

					continue
				}
				// shorthand escaped representations of popular characters
				// https://www.rfc-editor.org/rfc/rfc8259#page-9
				escapeSequence = false

				switch b {
				case 'b':
					l.in.currentValue = append(l.in.currentValue, '\b')
				case 'f':
					l.in.currentValue = append(l.in.currentValue, '\f')
				case 'n':
					l.in.currentValue = append(l.in.currentValue, '\n')
				case 't':
					l.in.currentValue = append(l.in.currentValue, '\t')
				case 'r':
					l.in.currentValue = append(l.in.currentValue, '\r')
				}

			case 'u':
				if !escapeSequence {
					l.in.currentValue = append(l.in.currentValue, b)

					continue
				}

				escapeSequence = false
				r, err := l.unescapeUnicodeSequence()
				if err != nil {
					l.in.err = err

					return token.None
				}

				l.in.currentValue = utf8.AppendRune(l.in.currentValue, r)

			default:
				if escapeSequence {
					l.in.err = codes.ErrUnknownEscape

					return token.None
				}

				if b < 0x20 {
					// RFC 8259: control characters U+0000..U+001F must be escaped
					l.in.err = codes.ErrControlChar

					return token.None
				}

				l.in.currentValue = append(l.in.currentValue, b)

				// bulk-scan the rest of this clean run within the current window
				// (§10.3 Phase 1c): a long clean stretch (e.g. between two escapes, or a
				// clean string that spilled past the fast path's window) is copied in one
				// shot — adaptive scalar-probe → SWAR → AVX2, the streaming analogue of
				// consumeStringEscaped's clean-run copy — instead of one byte per loop
				// turn. A run that reaches the window end just stops at bufferized; the
				// outer loop refills and re-enters here for the continuation.
				data := l.in.buffer
				n := l.in.bufferized
				runStart := l.in.consumed
				stop := runStart
				probe := min(stop+swarProbe, n)
				for ; stop < probe; stop++ {
					if c := data[stop]; c == doubleQuote || c == escape || c < 0x20 {
						break
					}
				}
				if stop == probe && stop < n { // run outran the scalar probe → SWAR
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
					if l.maxValueBytes > 0 && len(l.in.currentValue)+(stop-runStart) > l.maxValueBytes {
						l.in.err = codes.ErrMaxValueBytes

						return token.None
					}
					l.in.currentValue = append(l.in.currentValue, data[runStart:stop]...)
					l.in.offset += uint64(stop - runStart)
					l.in.consumed = stop
				}
			}
		}
	}
}

func (l *L) unescapeUnicodeSequence() (rune, error) {
	var buf [4]byte
	if err := l.in.consumeN(buf[:]); err != nil {
		return utf8.RuneError, codes.ErrUnicodeEscape
	}

	high1, highOK1 := scan.Unhex(buf[0])
	low1, lowOK1 := scan.Unhex(buf[1])
	high2, highOK2 := scan.Unhex(buf[2])
	low2, lowOK2 := scan.Unhex(buf[3])
	if !lowOK1 || !highOK1 || !lowOK2 || !highOK2 {
		return utf8.RuneError, codes.ErrUnicodeEscape
	}

	unicodeEscape := uint32(high1)<<12 + uint32(low1)<<8 + uint32(high2)<<4 + uint32(low2)
	r := rune(unicodeEscape)
	if utf16.IsSurrogate(r) {
		// this is a surrogate pair to encode a UTF-16 codepoint in 2 pairs
		// expect this to follow: \uXXXX
		var nextBuf [6]byte
		if err := l.in.consumeN(nextBuf[:]); err != nil {
			return utf8.RuneError, codes.ErrSurrogateEscape
		}

		if nextBuf[0] != escape || nextBuf[1] != 'u' {
			return utf8.RuneError, codes.ErrSurrogateEscape
		}

		high1, highOK1 = scan.Unhex(nextBuf[2])
		low1, lowOK1 = scan.Unhex(nextBuf[3])
		high2, highOK2 = scan.Unhex(nextBuf[4])
		low2, lowOK2 = scan.Unhex(nextBuf[5])
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
