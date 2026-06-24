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
	"io"

	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/lexers/token"
)

// emitPolicy converts the grammar/value token the core just built (a token.T)
// into the emitted token type T, attaching any policy-specific extra
// information (e.g. the verbatim lexer's preceding blanks and position).
type emitPolicy[T any] interface {
	emit(t token.T) T
}

// semanticPolicy is the policy for the semantic lexer L: the emitted token IS
// the grammar-state token.T, so emission is the identity.
type semanticPolicy struct{}

func (semanticPolicy) emit(t token.T) token.T { return t }

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

//nolint:gocognit,gocyclo
func scanPushG[T any, P emitPolicy[T]](l *L, p P, yield func(T) bool) {
	if l.err != nil {
		return
	}

	data := l.buffer[:l.bufferized]
	i := l.consumed

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
				continue
			}
			if !yield(p.emit(l.current)) {
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
			if !yield(p.emit(l.current)) {
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
			if !yield(p.emit(l.current)) {
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
			if !yield(p.emit(l.current)) {
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
			if !yield(p.emit(l.current)) {
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
			if !yield(p.emit(l.current)) {
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
			if !yield(p.emit(l.current)) {
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
			if !yield(p.emit(l.current)) {
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
					if !yield(p.emit(l.current)) {
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
			if !yield(p.emit(l.current)) {
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
			if !yield(p.emit(l.current)) {
				return
			}

		default:
			l.err = codes.ErrInvalidToken
			writeback(i + 1)

			return
		}
	}

	writeback(i)
	l.errCheck(io.EOF)
}
