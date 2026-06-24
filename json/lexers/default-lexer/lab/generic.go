package lab

// Generics spike (roadmap 2.1): a policy-parameterized push core, to measure
// whether Go's monomorphization penalizes the per-token emission path when the
// token type and construction are abstracted behind a concrete policy type.
//
// Design (see the design discussion in the session log / roadmap 2.1):
//   - The per-byte hot loop is policy-free: it operates on []byte + ints exactly
//     as the hand-written scanPush does. No generics cost there by construction.
//   - l.current (a token.T) stays the grammar-state memory: the loop reads its
//     Kind/Delimiter to validate the next token, unchanged.
//   - Emission is the ONLY thing routed through the policy, once per token. For
//     the semantic lexer the policy is identity (it emits the token.T already
//     built for grammar state), so construction happens exactly once — the fair
//     measurement of the per-token generic-call overhead.
//
// The policy takes *L so a future verbatim policy can read the blanks/position
// the core records; the semantic policy ignores it.

import (
	"errors"
	"io"

	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/lexers/token"
)

// emitPolicy converts the grammar/value token the core just built (a token.T)
// into the emitted token type T, attaching any policy-specific extra
// information: the preceding blanks (the whitespace run since the previous
// token, sliced zero-copy from the input) and the token's 1-based line/column.
// The semantic lexer ignores both; the verbatim lexer bakes them into token.VT.
type emitPolicy[T any] interface {
	emit(t token.T, blanks []byte, line, col int) T
	// none is the zero/error token (token.None / token.VNone), returned when the
	// lexer enters an error state.
	none() T
	// eof is the end-of-input token; the verbatim policy attaches the trailing
	// blanks, the semantic policy ignores them.
	eof(blanks []byte) T
}

// semanticPolicy is the policy for the semantic lexer L: the emitted token IS
// the grammar-state token.T, so emission is the identity (blanks/position are
// dropped — the core computes them anyway, but the slice is just a header).
type semanticPolicy struct{}

func (semanticPolicy) emit(t token.T, _ []byte, _, _ int) token.T { return t }
func (semanticPolicy) none() token.T                              { return token.None }
func (semanticPolicy) eof(_ []byte) token.T                       { return token.EOFToken }

// verbatimPolicy is the policy for the verbatim lexer VL: it wraps the
// grammar/value token.T into a token.VT, attaching the preceding blanks and the
// position (zero-cost wrap — VT embeds T).
type verbatimPolicy struct{}

func (verbatimPolicy) emit(t token.T, blanks []byte, line, col int) token.VT {
	return t.AsVerbatim(blanks).WithPosition(line, col)
}
func (verbatimPolicy) none() token.VT            { return token.VNone }
func (verbatimPolicy) eof(blanks []byte) token.VT { return token.MakeVerbatimEOF(blanks) }

// scanPushG is the generic, policy-parameterized counterpart of scanPush. It is
// a near-verbatim copy: every `yield(l.current)` becomes `yield(p.emit(...))`,
// and the token type is the type parameter T. Instantiated with a concrete
// policy (e.g. scanPushG[token.T, semanticPolicy]) so the policy call is a
// statically-known, devirtualizable call rather than an interface dispatch.
//
// scanPushSemantic is a non-generic wrapper around the generic core for the
// semantic lexer. It must stay a real (non-inlined) call in the body of the
// iter.Seq returned by Tokens: range-over-func keeps the yield closure on the
// stack only when the Seq body calls a concrete function whose "yield does not
// escape" summary crosses the package boundary (as the reference's scanPush
// does). If this wrapper inlines, the generic scanPushG call resurfaces in the
// Seq body and the range-over-func desugaring heap-allocates the yield closure
// in external callers (+2 allocs/call). Keep it opaque.
//
//go:noinline
func (l *L) scanPushSemantic(yield func(token.T) bool) {
	scanPushG[token.T, semanticPolicy](l, semanticPolicy{}, yield)
}

// scanPushVerbatim is the verbatim counterpart of scanPushSemantic: same
// non-inlined-shim discipline, but instantiates the core with verbatimPolicy so
// the emitted tokens are token.VT (blanks + position baked in). This gives VL a
// native push path with all of L's fast paths.
//
//go:noinline
func (l *L) scanPushVerbatim(yield func(token.VT) bool) {
	scanPushG[token.VT, verbatimPolicy](l, verbatimPolicy{}, yield)
}

//nolint:gocognit,gocyclo
func scanPushG[T any, P emitPolicy[T]](l *L, p P, yield func(T) bool) {
	if l.err != nil {
		return
	}

	data := l.buffer[:l.bufferized]
	i := l.consumed
	// blankStart is the index right after the previous token: the whitespace run
	// [blankStart:tokenStart] is the preceding blanks the verbatim policy bakes
	// into the token (zero-copy slice of the input). The semantic policy ignores
	// it. It is reset to i after each emitted token.
	blankStart := i

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

		l.tokLine = l.line
		l.tokCol = int(uint64(i+1) - l.lineStart)
		// the whitespace run since the previous token (zero-copy); i is the index
		// of the first significant byte (the token start).
		blanks := data[blankStart:i:i]

		if l.afterKey {
			l.afterKey = false
			if b != colon {
				l.err = codes.ErrKeyColon
				writeback(i + 1)

				return
			}

			i++
			l.current = token.MakeDelimiter(token.Colon)
			blankStart = i
			if l.elideSeparator {
				continue
			}
			if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
				writeback(i)

				return
			}

			continue
		}

		switch b {
		case colon:
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
			if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
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
			if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
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
				continue
			}
			if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
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
			if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
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
			if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
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
			if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
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
			if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
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
					blankStart = i // this path continues, bypassing the loop-bottom update
					if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
						return
					}

					continue
				}
			}

			writeback(i + 1)
			l.current = l.consumeNumberWhole(b)
			if l.err != nil {
				return
			}
			i = l.consumed
			if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
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
			if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
				return
			}

		default:
			l.err = codes.ErrInvalidToken
			writeback(i + 1)

			return
		}

		// the token just processed ends at i: the next blanks run starts here.
		// (the afterKey colon path updates blankStart itself before its continue.)
		blankStart = i
	}

	writeback(i)
	l.errCheck(io.EOF)
}

// errCheckG is the generic counterpart of (*L).errCheck: shared EOF/error
// classification, returning the policy's eof token (with trailing blanks for the
// verbatim policy) on clean EOF, or the none token otherwise.
func errCheckG[T any, P emitPolicy[T]](l *L, p P, err error) T {
	hadToken := l.current.IsKnown()
	l.current = token.None

	if errors.Is(err, io.EOF) {
		switch {
		case l.isInContainer():
			if l.isInObject() {
				l.err = codes.ErrNotInObject
			} else {
				l.err = codes.ErrNotInArray
			}
		case l.isAtEOF:
			l.err = io.EOF
		case !hadToken:
			l.err = codes.ErrNoData
		}

		l.isAtEOF = true

		return p.eof(l.blanks)
	}

	l.err = err

	return p.none()
}

// scanTokenG is the generic, policy-parameterized pull core: it scans and
// returns exactly one token, shared by L.NextToken and VL.NextToken. It mirrors
// (*L).scanToken exactly (cursor in the struct, per-byte advance, readMore for
// streaming, deferred-error semantics) and emits via the policy. When
// l.trackBlanks is set (verbatim) it accumulates the preceding whitespace run
// into l.blanks (byte-by-byte, so it survives streaming refills); the semantic
// policy ignores blanks.
//
//nolint:gocognit,gocyclo
func scanTokenG[T any, P emitPolicy[T]](l *L, p P) T {
	if l.err != nil {
		return p.none()
	}

	if l.trackBlanks {
		l.blanks = l.blanks[:0]
	}

	for {
		if err := l.readMore(); err != nil {
			return errCheckG(l, p, err)
		}

		for l.consumed < l.bufferized {
			b := l.buffer[l.consumed]
			l.offset++
			l.consumed++

			switch b {
			case lineFeed:
				l.line++
				l.lineStart = l.offset
				if l.trackBlanks {
					l.blanks = append(l.blanks, b)
				}

				continue

			case blank, tab, carriageReturn:
				if l.trackBlanks {
					l.blanks = append(l.blanks, b)
				}

				continue
			}

			// a significant byte starts a token: snapshot its position
			l.tokLine = l.line
			l.tokCol = int(l.offset - l.lineStart)

			if l.afterKey {
				l.afterKey = false
				if b != colon {
					l.err = codes.ErrKeyColon

					return p.none()
				}

				l.current = token.MakeDelimiter(token.Colon)
				if l.elideSeparator {
					continue
				}

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)
			}

			switch b {
			case colon:
				if l.current.Kind() == token.String {
					l.err = codes.ErrMissingObject
				} else {
					l.err = codes.ErrMissingKey
				}

				return p.none()

			case closingBracket:
				if l.current.IsComma() {
					l.err = codes.ErrTrailingComma

					return p.none()
				}
				if !l.isInObject() {
					l.err = codes.ErrNotInObject

					return p.none()
				}

				l.expectKey = false
				l.popContainer()
				l.current = token.MakeDelimiter(token.ClosingBracket)

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			case closingSquareBracket:
				if l.current.IsComma() {
					l.err = codes.ErrTrailingComma

					return p.none()
				}
				if !l.isInArray() {
					l.err = codes.ErrNotInArray

					return p.none()
				}

				l.popContainer()
				l.current = token.MakeDelimiter(token.ClosingSquareBracket)

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			case comma:
				if l.current.IsComma() {
					l.err = codes.ErrRepeatedComma

					return p.none()
				}
				if l.expectKey {
					l.err = codes.ErrMissingKey

					return p.none()
				}
				if !l.isInContainer() {
					l.err = codes.ErrCommaInContainer

					return p.none()
				}
				if l.current.IsStartObject() || l.current.IsStartArray() || l.current.IsColon() {
					l.err = codes.ErrMissingValue

					return p.none()
				}

				if l.isInObject() {
					l.expectKey = true
				}

				l.current = token.MakeDelimiter(token.Comma)
				if l.elideSeparator {
					continue
				}

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			case openingBracket:
				if l.current.IsKnown() {
					if l.current.Kind() != token.Delimiter {
						l.err = codes.ErrInvalidToken

						return p.none()
					}
					if l.current.Delimiter().IsClosing() {
						l.err = codes.ErrMissingComma

						return p.none()
					}
					if l.isInArray() {
						if l.current.Delimiter() != token.OpeningSquareBracket &&
							l.current.Delimiter() != token.Comma {
							l.err = codes.ErrMissingComma

							return p.none()
						}
					} else if !l.current.IsColon() {
						l.err = codes.ErrMissingKey

						return p.none()
					}
				}
				if l.expectKey {
					l.err = codes.ErrMissingKey

					return p.none()
				}

				l.expectKey = true
				l.pushObject()
				if l.err != nil {
					return p.none()
				}
				l.current = token.MakeDelimiter(token.OpeningBracket)

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			case openingSquareBracket:
				if l.current.IsKnown() {
					if l.current.Kind() != token.Delimiter {
						l.err = codes.ErrInvalidToken

						return p.none()
					}
					if l.current.Delimiter().IsClosing() {
						l.err = codes.ErrMissingComma

						return p.none()
					}
				}
				if l.expectKey {
					l.err = codes.ErrMissingKey

					return p.none()
				}

				l.pushArray()
				if l.err != nil {
					return p.none()
				}
				l.current = token.MakeDelimiter(token.OpeningSquareBracket)

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			case doubleQuote:
				if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
					l.err = codes.ErrDelimitedValue
					l.current = token.None

					return p.none()
				}

				l.current = l.consumeString()
				if l.err != nil {
					return p.none()
				}

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			case startOfTrue, startOfFalse:
				if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
					l.err = codes.ErrDelimitedValue
					l.current = token.None

					return p.none()
				}
				if l.expectKey {
					l.err = codes.ErrMissingKey

					return p.none()
				}

				l.current = l.consumeBoolean(b)
				if l.err != nil {
					return p.none()
				}

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			case minusSign, decimalPoint, '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
					l.err = codes.ErrDelimitedValue
					l.current = token.None

					return p.none()
				}
				if l.expectKey {
					l.err = codes.ErrMissingKey

					return p.none()
				}

				if l.wholeBuffer && l.maxValueBytes == 0 {
					buf := l.buffer[:l.bufferized]
					numStart := l.consumed - 1
					runFrom := l.consumed
					var firstDigit byte
					ok := true

					switch {
					case b >= '0' && b <= '9':
						firstDigit = b
					case b == minusSign:
						if uint(l.consumed) < uint(len(buf)) && buf[l.consumed] >= '0' && buf[l.consumed] <= '9' {
							firstDigit = buf[l.consumed]
							runFrom = l.consumed + 1
						} else {
							ok = false
						}
					default:
						ok = false
					}

					if ok {
						n := runFrom
						for uint(n) < uint(len(buf)) && '0' <= buf[n] && buf[n] <= '9' {
							n++
						}

						leadingZero := firstDigit == '0' && n > runFrom
						var term byte
						if uint(n) < uint(len(buf)) {
							term = buf[n]
						}

						if !leadingZero && term != decimalPoint && term != 'e' && term != 'E' {
							l.offset += uint64(n - l.consumed)
							l.consumed = n
							l.current = token.MakeWithValue(token.Number, l.buffer[numStart:n:n])

							return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)
						}
					}

					l.current = l.consumeNumberWhole(b)
					if l.err != nil {
						return p.none()
					}

					return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)
				}

				l.current = l.consumeNumberStreaming(b)
				if l.err != nil {
					return p.none()
				}

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			case startOfNull:
				if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
					l.err = codes.ErrDelimitedValue
					l.current = token.None

					return p.none()
				}
				if l.expectKey {
					l.err = codes.ErrMissingKey

					return p.none()
				}

				l.current = l.consumeNull(b)
				if l.err != nil {
					return p.none()
				}

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			default:
				l.err = codes.ErrInvalidToken

				return p.none()
			}
		}
	}
}
