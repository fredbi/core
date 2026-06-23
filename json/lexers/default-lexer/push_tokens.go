package lexer

import (
	"io"

	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/lexers/token"
)

// scanPush is the native push back-end for Tokens() in whole-buffer mode. It
// drives the scan loop itself and yields tokens through yield, keeping the byte
// cursor in a local (i) across the ENTIRE scan so the hot loop performs no
// per-byte writes to the struct cursor (l.consumed/l.offset) — the property that
// makes push faster than the pull NextToken loop on string/structure-heavy input.
//
// It mirrors scanToken's validation exactly (token streams and error codes are
// identical, verified by push-equivalence tests + conformance) and reuses the
// value scanners (consumeString / consumeNumberWhole / consumeBoolean /
// consumeNull) by syncing l.consumed/l.offset around each call. The inline
// integer fast path is duplicated here to avoid a call on the common case.
//
// Line/column tracking is preserved: l.line/l.lineStart are updated on line
// feeds and l.tokLine/l.tokCol are snapshotted at each token start, exactly as in
// scanToken (in whole-buffer mode l.offset == the buffer index).
//
//nolint:gocognit,gocyclo
func (l *L) scanPush(yield func(token.T) bool) {
	if l.err != nil {
		return
	}

	data := l.buffer[:l.bufferized]
	i := l.consumed

	// writeback records the final cursor position into the struct (offset ==
	// index in whole-buffer mode); call before every early return.
	writeback := func(pos int) {
		l.consumed = pos
		l.offset = uint64(pos)
	}

	for i < len(data) {
		b := data[i]

		switch b {
		case lineFeed:
			i++
			l.line++
			l.lineStart = uint64(i)

			continue
		case blank, tab, carriageReturn:
			i++

			continue
		}

		// a significant byte starts a token: snapshot its position (l.offset
		// would be i+1 once b is consumed, matching scanToken).
		l.tokLine = l.line
		l.tokCol = int(uint64(i+1) - l.lineStart)

		// an object key must be followed by the ':' name-separator
		if l.afterKey {
			l.afterKey = false
			if b != colon {
				l.err = codes.ErrKeyColon
				writeback(i + 1)

				return
			}

			i++
			l.current = token.MakeDelimiter(token.Colon)
			if l.elideSeparator {
				continue // skip the colon; context recorded in l.current
			}
			if !yield(l.current) {
				writeback(i)

				return
			}

			continue
		}

		switch b {
		case colon:
			// a stray colon: only an object key (handled above) may precede one
			if l.current.Kind() == token.String {
				l.err = codes.ErrMissingObject
			} else {
				l.err = codes.ErrMissingKey
			}
			writeback(i + 1)

			return

		case closingBracket:
			if l.current.IsComma() {
				l.err = codes.ErrTrailingComma
				writeback(i + 1)

				return
			}
			if !l.isInObject() {
				l.err = codes.ErrNotInObject
				writeback(i + 1)

				return
			}

			i++
			l.expectKey = false
			l.popContainer()
			l.current = token.MakeDelimiter(token.ClosingBracket)
			if !yield(l.current) {
				writeback(i)

				return
			}

		case closingSquareBracket:
			if l.current.IsComma() {
				l.err = codes.ErrTrailingComma
				writeback(i + 1)

				return
			}
			if !l.isInArray() {
				l.err = codes.ErrNotInArray
				writeback(i + 1)

				return
			}

			i++
			l.popContainer()
			l.current = token.MakeDelimiter(token.ClosingSquareBracket)
			if !yield(l.current) {
				writeback(i)

				return
			}

		case comma:
			if l.current.IsComma() {
				l.err = codes.ErrRepeatedComma
				writeback(i + 1)

				return
			}
			if l.expectKey {
				l.err = codes.ErrMissingKey
				writeback(i + 1)

				return
			}
			if !l.isInContainer() {
				l.err = codes.ErrCommaInContainer
				writeback(i + 1)

				return
			}
			if l.current.IsStartObject() || l.current.IsStartArray() || l.current.IsColon() {
				l.err = codes.ErrMissingValue
				writeback(i + 1)

				return
			}

			if l.isInObject() {
				l.expectKey = true
			}

			i++
			l.current = token.MakeDelimiter(token.Comma)
			if l.elideSeparator {
				continue // skip the comma; context recorded in l.current
			}
			if !yield(l.current) {
				writeback(i)

				return
			}

		case openingBracket:
			if l.current.IsKnown() {
				if l.current.Kind() != token.Delimiter {
					l.err = codes.ErrInvalidToken
					writeback(i + 1)

					return
				}
				if l.current.Delimiter().IsClosing() {
					l.err = codes.ErrMissingComma
					writeback(i + 1)

					return
				}
				if l.isInArray() {
					if l.current.Delimiter() != token.OpeningSquareBracket &&
						l.current.Delimiter() != token.Comma {
						l.err = codes.ErrMissingComma
						writeback(i + 1)

						return
					}
				} else if !l.current.IsColon() {
					l.err = codes.ErrMissingKey
					writeback(i + 1)

					return
				}
			}
			if l.expectKey {
				l.err = codes.ErrMissingKey
				writeback(i + 1)

				return
			}

			i++
			l.expectKey = true
			l.pushObject()
			if l.err != nil {
				writeback(i)

				return
			}
			l.current = token.MakeDelimiter(token.OpeningBracket)
			if !yield(l.current) {
				writeback(i)

				return
			}

		case openingSquareBracket:
			if l.current.IsKnown() {
				if l.current.Kind() != token.Delimiter {
					l.err = codes.ErrInvalidToken
					writeback(i + 1)

					return
				}
				if l.current.Delimiter().IsClosing() {
					l.err = codes.ErrMissingComma
					writeback(i + 1)

					return
				}
			}
			if l.expectKey {
				l.err = codes.ErrMissingKey
				writeback(i + 1)

				return
			}

			i++
			l.pushArray()
			if l.err != nil {
				writeback(i)

				return
			}
			l.current = token.MakeDelimiter(token.OpeningSquareBracket)
			if !yield(l.current) {
				writeback(i)

				return
			}

		case doubleQuote:
			if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
				l.err = codes.ErrDelimitedValue
				l.current = token.None
				writeback(i + 1)

				return
			}

			writeback(i + 1)
			l.current = l.consumeString()
			if l.err != nil {
				return
			}
			i = l.consumed
			if !yield(l.current) {
				return
			}

		case startOfTrue, startOfFalse:
			if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
				l.err = codes.ErrDelimitedValue
				l.current = token.None
				writeback(i + 1)

				return
			}
			if l.expectKey {
				l.err = codes.ErrMissingKey
				writeback(i + 1)

				return
			}

			writeback(i + 1)
			l.current = l.consumeBoolean(b)
			if l.err != nil {
				return
			}
			i = l.consumed
			if !yield(l.current) {
				return
			}

		case minusSign, decimalPoint, '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
				l.err = codes.ErrDelimitedValue
				l.current = token.None
				writeback(i + 1)

				return
			}
			if l.expectKey {
				l.err = codes.ErrMissingKey
				writeback(i + 1)

				return
			}

			// inline integer fast path (mirrors scanToken): a plain integer,
			// optionally negative, aliased with no call and no state machine.
			numStart := i
			runFrom := i + 1
			var firstDigit byte
			ok := true

			switch {
			case b >= '0' && b <= '9':
				firstDigit = b
			case b == minusSign:
				if uint(i+1) < uint(len(data)) && data[i+1] >= '0' && data[i+1] <= '9' {
					firstDigit = data[i+1]
					runFrom = i + 2
				} else {
					ok = false
				}
			default: // decimalPoint
				ok = false
			}

			if ok {
				n := runFrom
				for uint(n) < uint(len(data)) && '0' <= data[n] && data[n] <= '9' {
					n++
				}

				leadingZero := firstDigit == '0' && n > runFrom
				var term byte
				if uint(n) < uint(len(data)) {
					term = data[n]
				}

				if !leadingZero && term != decimalPoint && term != 'e' && term != 'E' {
					l.current = token.MakeWithValue(token.Number, data[numStart:n:n])
					i = n
					writeback(n)
					if !yield(l.current) {
						return
					}

					continue
				}
			}

			// complicated number: reuse the digit-run whole-buffer scanner
			writeback(i + 1)
			l.current = l.consumeNumberWhole(b)
			if l.err != nil {
				return
			}
			i = l.consumed
			if !yield(l.current) {
				return
			}

		case startOfNull:
			if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
				l.err = codes.ErrDelimitedValue
				l.current = token.None
				writeback(i + 1)

				return
			}
			if l.expectKey {
				l.err = codes.ErrMissingKey
				writeback(i + 1)

				return
			}

			writeback(i + 1)
			l.current = l.consumeNull(b)
			if l.err != nil {
				return
			}
			i = l.consumed
			if !yield(l.current) {
				return
			}

		default:
			l.err = codes.ErrInvalidToken
			writeback(i + 1)

			return
		}
	}

	// end of input: reproduce the pull EOF semantics (clean EOF, ErrNoData, or an
	// unterminated-container error), keeping l.err / l.isAtEOF consistent.
	writeback(i)
	l.errCheck(io.EOF)
}
