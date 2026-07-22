package lexer

//go:generate go run ./internal/lexgen

// Unified lexer core (roadmap 2.1): two policy-parameterized generic cores —
// scanPushG (push) and scanTokenG (pull) — are the single source of truth for
// both the semantic lexer L and the verbatim lexer VL. A concrete zero-size
// policy per lexer (semanticPolicy / verbatimPolicy) selects the emitted token
// type and how each token is built, replacing the four hand-written loops the
// two lexers used to carry.
//
// Design:
//   - The per-byte hot loop is policy-free: it operates on []byte + ints. No
//     generics cost there by construction.
//   - l.current (a token.T) stays the grammar-state memory: the loop reads its
//     Kind/Delimiter to validate the next token.
//   - Emission is the ONLY thing routed through the policy, once per token. For
//     the semantic lexer the policy is identity (it emits the token.T already
//     built for grammar state); for the verbatim lexer it wraps that token.T
//     into a token.VT with the preceding blanks + position.
//   - Accepted cost: the per-token policy call routes through the generics
//     dictionary (Go does not devirtualize type-param method calls), ~5% on L.
//     See roadmap 2.1 for the measurement and the rationale for accepting it.

import (
	"errors"
	"io"

	codes "github.com/fredbi/core/json/lexers/error-codes"
	scan "github.com/fredbi/core/json/lexers/internal/scan"
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
	// tracksPosition reports whether the core must maintain line/column accounting.
	// Only the verbatim lexer [VL] needs it (it exposes line/col as lexer state); the
	// semantic lexer drops it. Returning a constant lets the devirtualized cores
	// (scan_gen.go) constant-fold and dead-code-eliminate the accounting entirely in
	// the semantic core, so its hot loop does no per-newline / per-token position
	// bookkeeping — matching jsontext's offset-only model.
	tracksPosition() bool
	// storesBlanks reports whether the core must stash the preceding-blanks slice into
	// lexer state (l.blanks) at each token boundary. True only for the verbatim lexer
	// [VL], which emits a light token.T and exposes the blanks via [VL.LeadingSpace]
	// instead of baking them into the token. Constant-folds away where false (semantic).
	storesBlanks() bool
}

// semanticPolicy is the policy for the semantic lexer L: the emitted token IS
// the grammar-state token.T, so emission is the identity (blanks/position are
// dropped — the core computes them anyway, but the slice is just a header).
type semanticPolicy struct{}

func (semanticPolicy) emit(t token.T, _ []byte, _, _ int) token.T { return t }
func (semanticPolicy) none() token.T                              { return token.None }
func (semanticPolicy) eof(_ []byte) token.T                       { return token.EOFToken }
func (semanticPolicy) tracksPosition() bool                       { return false }
func (semanticPolicy) storesBlanks() bool                         { return false }

// verbatimPolicy is the policy for the verbatim lexer [VL]: the "token-vs-state
// arbitrage". It emits the LIGHT token.T (identity, like the semantic policy) and
// keeps the verbatim feature as LEXER STATE — the preceding-blanks slice is stashed
// in l.blanks (via storesBlanks, read back through [VL.LeadingSpace]) and the
// position stays in l.tokLine / l.tokCol (the core writes it since tracksPosition is
// true, read back through [VL.Line] / [VL.Column]). This replaced the original
// token.VT-based verbatim lexer (§10.5a sizing: the per-token 72B VT
// construct-and-return-by-value was ~85% of the verbatim throughput tax; the
// state-based lexer runs at ~77–84% of the semantic L across all modes vs VT's ~27%).
type verbatimPolicy struct{}

func (verbatimPolicy) emit(t token.T, _ []byte, _, _ int) token.T { return t }
func (verbatimPolicy) none() token.T                              { return token.None }
func (verbatimPolicy) eof(_ []byte) token.T                       { return token.EOFToken }
func (verbatimPolicy) tracksPosition() bool                       { return true }
func (verbatimPolicy) storesBlanks() bool                         { return true }

// scanPushG is the generic, policy-parameterized push core backing Tokens() for
// both L and VL in whole-buffer mode. The per-byte hot loop keeps the cursor in
// a local; each token is emitted via `yield(p.emit(...))`, with the token type
// the type parameter T. Instantiated with a concrete policy (e.g.
// scanPushG[token.T, semanticPolicy]) so the policy is a statically-known type.
//
// scanPushSemantic is a non-generic wrapper around the generic core for the
// semantic lexer. It must stay a real (non-inlined) call in the body of the
// iter.Seq returned by Tokens: range-over-func keeps the yield closure on the
// stack only when the Seq body calls a concrete function whose "yield does not
// escape" summary crosses the package boundary. If this wrapper inlines, the
// generic scanPushG call resurfaces in the Seq body and the range-over-func
// desugaring heap-allocates the yield closure in external callers (+2
// allocs/call). Keep it opaque.
//
//go:noinline
func (l *L) scanPushSemantic(yield func(token.T) bool) {
	scanPushG[token.T, semanticPolicy](l, semanticPolicy{}, yield)
}

// scanPushVerbatim is the verbatim counterpart of scanPushSemantic: same
// non-inlined-shim discipline, instantiating the core with verbatimPolicy — which
// emits the light token.T and stashes blanks/position in lexer state. This gives VL
// a native push path with all of L's fast paths.
//
//go:noinline
func (l *L) scanPushVerbatim(yield func(token.T) bool) {
	scanPushG[token.T, verbatimPolicy](l, verbatimPolicy{}, yield)
}

//nolint:gocognit,gocyclo
func scanPushG[T any, P emitPolicy[T]](l *L, p P, yield func(T) bool) {
	if l.in.Err != nil {
		return
	}

	data := l.in.Buffer[:l.in.Bufferized]
	i := l.in.Consumed
	// blankStart is the index right after the previous token: the whitespace run
	// [blankStart:tokenStart] is the preceding blanks the verbatim policy bakes
	// into the token (zero-copy slice of the input). The semantic policy ignores
	// it. It is reset to i after each emitted token.
	blankStart := i

	writeback := func(pos int) {
		l.in.Consumed = pos
		l.in.Offset = uint64(pos)
	}

	for i < len(data) {
		b := data[i]

		switch b {
		case lineFeed:
			if !p.tracksPosition() {
				// semantic: batch-skip the whole whitespace run in one shot (the
				// citm/indentation bottleneck), mirroring the pull core. Gated by
				// the compile-time tracksPosition() constant so the verbatim core,
				// which must walk each blank to accumulate the preceding-blanks
				// slice and count lines, keeps its per-byte path below. Register-
				// safe: the semantic core has headroom since it dropped line/col.
				i += scan.ConsumeWhitespace(data[i:])

				continue
			}
			i++
			l.line++
			l.lineStart = uint64(i)

			continue
		case blank, tab, carriageReturn:
			if !p.tracksPosition() {
				i += scan.ConsumeWhitespace(data[i:]) // semantic batch-skip; see lineFeed

				continue
			}
			i++

			continue
		}

		if p.tracksPosition() {
			l.tokLine = l.line
			l.tokCol = int(uint64(i+1) - l.lineStart)
		}
		// the whitespace run since the previous token (zero-copy); i is the index
		// of the first significant byte (the token start).
		blanks := data[blankStart:i:i]
		if p.storesBlanks() {
			l.blanks = blanks // state-based VL: stash the leading blanks in lexer state
		}

		if l.in.AfterKey {
			l.in.AfterKey = false
			if b != colon {
				l.in.Err = codes.ErrKeyColon
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
				l.in.Err = codes.ErrMissingObject
			} else {
				l.in.Err = codes.ErrMissingKey
			}
			writeback(i + 1)

			return

		case closingBracket:
			if l.current.IsComma() {
				l.in.Err = codes.ErrTrailingComma
				writeback(i + 1)

				return
			}
			if !l.isInObject() {
				l.in.Err = codes.ErrNotInObject
				writeback(i + 1)

				return
			}

			i++
			l.in.ExpectKey = false
			l.popContainer()
			l.current = token.MakeDelimiter(token.ClosingBracket)
			if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
				writeback(i)

				return
			}

		case closingSquareBracket:
			if l.current.IsComma() {
				l.in.Err = codes.ErrTrailingComma
				writeback(i + 1)

				return
			}
			if !l.isInArray() {
				l.in.Err = codes.ErrNotInArray
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
				l.in.Err = codes.ErrRepeatedComma
				writeback(i + 1)

				return
			}
			if l.in.ExpectKey {
				l.in.Err = codes.ErrMissingKey
				writeback(i + 1)

				return
			}
			if !l.isInContainer() {
				l.in.Err = codes.ErrCommaInContainer
				writeback(i + 1)

				return
			}
			if l.current.IsStartObject() || l.current.IsStartArray() || l.current.IsColon() {
				l.in.Err = codes.ErrMissingValue
				writeback(i + 1)

				return
			}

			if l.isInObject() {
				l.in.ExpectKey = true
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
					l.in.Err = codes.ErrInvalidToken
					writeback(i + 1)

					return
				}
				if l.current.Delimiter().IsClosing() {
					l.in.Err = codes.ErrMissingComma
					writeback(i + 1)

					return
				}
				if l.isInArray() {
					if l.current.Delimiter() != token.OpeningSquareBracket &&
						l.current.Delimiter() != token.Comma {
						l.in.Err = codes.ErrMissingComma
						writeback(i + 1)

						return
					}
				} else if !l.current.IsColon() {
					l.in.Err = codes.ErrMissingKey
					writeback(i + 1)

					return
				}
			}
			if l.in.ExpectKey {
				l.in.Err = codes.ErrMissingKey
				writeback(i + 1)

				return
			}

			i++
			l.in.ExpectKey = true
			l.pushObject()
			if l.in.Err != nil {
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
					l.in.Err = codes.ErrInvalidToken
					writeback(i + 1)

					return
				}
				if l.current.Delimiter().IsClosing() {
					l.in.Err = codes.ErrMissingComma
					writeback(i + 1)

					return
				}
			}
			if l.in.ExpectKey {
				l.in.Err = codes.ErrMissingKey
				writeback(i + 1)

				return
			}

			i++
			l.pushArray()
			if l.in.Err != nil {
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
				l.in.Err = codes.ErrDelimitedValue
				l.current = token.None
				writeback(i + 1)

				return
			}

			writeback(i + 1)
			l.current = l.in.ConsumeString()
			if l.in.Err != nil {
				return
			}
			i = l.in.Consumed
			if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
				return
			}

		case startOfTrue, startOfFalse:
			if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
				l.in.Err = codes.ErrDelimitedValue
				l.current = token.None
				writeback(i + 1)

				return
			}
			if l.in.ExpectKey {
				l.in.Err = codes.ErrMissingKey
				writeback(i + 1)

				return
			}

			writeback(i + 1)
			l.current = l.in.ConsumeBoolean(b)
			if l.in.Err != nil {
				return
			}
			i = l.in.Consumed
			if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
				return
			}

		case minusSign, decimalPoint, '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
				l.in.Err = codes.ErrDelimitedValue
				l.current = token.None
				writeback(i + 1)

				return
			}
			if l.in.ExpectKey {
				l.in.Err = codes.ErrMissingKey
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
				end := n
				var term byte
				if uint(end) < uint(len(data)) {
					term = data[end]
				}
				// extend the fast path over a fractional part ('.' 1*DIGIT). A
				// trailing dot (no fraction digit) leaves term==decimalPoint, which the
				// final guard routes to consumeNumberWhole for the right error.
				if !leadingZero && term == decimalPoint {
					m := end + 1
					for uint(m) < uint(len(data)) && '0' <= data[m] && data[m] <= '9' {
						m++
					}
					if m > end+1 { // at least one fractional digit
						end = m
						term = 0
						if uint(end) < uint(len(data)) {
							term = data[end]
						}
					}
				}
				// extend over an exponent ((e|E) [+|-] 1*DIGIT). Completing it inline
				// is cheaper than bailing to consumeNumberWhole, which would re-scan
				// the int+frac we already consumed. A malformed exponent leaves
				// term=='e'/'E' and routes to the slow path for the right error.
				if !leadingZero && (term == 'e' || term == 'E') {
					m := end + 1
					if uint(m) < uint(len(data)) && (data[m] == '+' || data[m] == '-') {
						m++
					}
					expStart := m
					for uint(m) < uint(len(data)) && '0' <= data[m] && data[m] <= '9' {
						m++
					}
					if m > expStart { // at least one exponent digit
						end = m
						term = 0
						if uint(end) < uint(len(data)) {
							term = data[end]
						}
					}
				}

				if !leadingZero && term != decimalPoint && term != 'e' && term != 'E' {
					l.current = token.MakeWithValue(token.Number, data[numStart:end:end])
					i = end
					writeback(end)
					blankStart = i // this path continues, bypassing the loop-bottom update
					if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
						return
					}

					continue
				}
			}

			writeback(i + 1)
			l.current = l.in.ConsumeNumberWhole(b)
			if l.in.Err != nil {
				return
			}
			i = l.in.Consumed
			if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
				return
			}

		case startOfNull:
			if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
				l.in.Err = codes.ErrDelimitedValue
				l.current = token.None
				writeback(i + 1)

				return
			}
			if l.in.ExpectKey {
				l.in.Err = codes.ErrMissingKey
				writeback(i + 1)

				return
			}

			writeback(i + 1)
			l.current = l.in.ConsumeNull(b)
			if l.in.Err != nil {
				return
			}
			i = l.in.Consumed
			if !yield(p.emit(l.current, blanks, l.tokLine, l.tokCol)) {
				return
			}

		default:
			l.in.Err = codes.ErrInvalidToken
			writeback(i + 1)

			return
		}

		// the token just processed ends at i: the next blanks run starts here.
		// (the afterKey colon path updates blankStart itself before its continue.)
		blankStart = i
	}

	writeback(i)
	// classify the terminal EOF for its side effects on l.in.Err/l.isAtEOF; the
	// push core yields no token here, so the returned EOF token is discarded.
	_ = errCheckG(l, p, io.EOF)
}

// errCheckG performs the shared EOF/error classification for both cores,
// returning the policy's eof token (with trailing blanks for the verbatim
// policy) on clean EOF, or the none token otherwise.
func errCheckG[T any, P emitPolicy[T]](l *L, p P, err error) T {
	hadToken := l.current.IsKnown()
	l.current = token.None

	if errors.Is(err, io.EOF) {
		switch {
		case l.isInContainer():
			if l.isInObject() {
				l.in.Err = codes.ErrNotInObject
			} else {
				l.in.Err = codes.ErrNotInArray
			}
		case l.isAtEOF:
			l.in.Err = io.EOF
		case !hadToken:
			l.in.Err = codes.ErrNoData
		}

		l.isAtEOF = true

		return p.eof(l.blanks)
	}

	l.in.Err = err

	return p.none()
}

// whitespace scanning + hex/\u decoding are stateless primitives shared with the
// token package; they live in internal/scan (still inline into the hot cores).

// skipBlanksRestStream batch-skips the CONTINUATION of a whitespace run in the current
// window for the position-tracking stream cores (§10.5d) — the verbatim/state analogue
// of the semantic core's consumeWhitespace batch-skip. The caller has already consumed
// and captured the run's first byte and confirmed (a cheap inline peek) that l.in.Consumed
// points at another whitespace byte, so this scans the rest of the run in ONE step,
// updates line/lineStart from a single scan, and — when trackBlanks — BULK-appends the
// rest into l.blanks. Splitting it this way keeps SHORT runs (e.g. mesh's 73k
// single-byte runs) on the cheap inline path — no call — while LONG runs (pretty) pay
// one call and one memcpy instead of a per-byte walk. A run reaching the window end
// stops at bufferized; the outer loop refills and re-enters, so a run spanning refills
// accumulates across calls. The caller does the maxValueBytes check once afterwards.
func (l *L) skipBlanksRestStream() {
	base := l.in.Offset - uint64(l.in.Consumed) // absolute offset of buffer index 0 this window
	start := l.in.Consumed
	n, lines, afterNL := scan.ConsumeWhitespaceTracked(l.in.Buffer[start:l.in.Bufferized])

	if lines > 0 {
		l.line += lines
		l.lineStart = base + uint64(start+afterNL) // just past the last newline in the run
	}

	l.in.Consumed = start + n
	l.in.Offset = base + uint64(start+n)

	if l.trackBlanks {
		l.blanks = append(l.blanks, l.in.Buffer[start:l.in.Consumed]...)
	}
}

// scanTokenStreamG is the generic, policy-parameterized pull core for STREAMING
// input (io.Reader): it scans and returns exactly one token, dispatched from
// L.NextToken / VL.NextToken when l.in.WholeBuffer is false. The cursor lives in the
// struct (per-byte advance, readMore for refills, deferred-error semantics) and
// each token is emitted via the policy. When l.trackBlanks is set (verbatim) it
// accumulates the preceding whitespace run into l.blanks (byte-by-byte, so it
// survives streaming refills); the semantic policy ignores blanks.
//
// The whole-buffer lane is scanTokenBufferG (no readMore, local cursor, zero-copy
// blanks). Splitting the two (roadmap §10) lets us optimize the stream refill
// path without perturbing the register-delicate whole-buffer core (§9.1). Here
// l.in.WholeBuffer is always false; values take the streaming fast paths
// (consumeStringStreamFast / consumeNumberStreamFast, §10.3) — optimistic in-window
// scan + zero-copy alias, delegating to the byte-by-byte consumers only on an
// escape or a token that spans a refill.
//
//nolint:gocognit,gocyclo
func scanTokenStreamG[T any, P emitPolicy[T]](l *L, p P) T {
	if l.in.Err != nil {
		return p.none()
	}

	if l.trackBlanks {
		l.blanks = l.blanks[:0]
	}

	for {
		if err := l.in.ReadMore(); err != nil {
			return errCheckG(l, p, err)
		}

		for l.in.Consumed < l.in.Bufferized {
			b := l.in.Buffer[l.in.Consumed]
			l.in.Offset++
			l.in.Consumed++

			switch b {
			case lineFeed:
				if !p.tracksPosition() {
					// semantic: batch-skip the rest of the whitespace run with a local
					// cursor (no line/col, no blanks) — folds to the only path in the
					// devirtualized semantic core. This kills the per-byte struct-cursor
					// cost over whitespace (the citm bottleneck).
					ws := scan.ConsumeWhitespace(l.in.Buffer[l.in.Consumed:l.in.Bufferized])
					l.in.Consumed += ws
					l.in.Offset += uint64(ws)

					continue
				}
				// verbatim/state (§10.5d): handle the first byte inline (b is a newline),
				// then batch-skip the REST only if the run actually continues — so a
				// single-byte run stays as cheap as the old per-byte path (no call).
				l.line++
				l.lineStart = l.in.Offset // just past the newline b
				if l.trackBlanks {
					l.blanks = append(l.blanks, b)
				}
				if l.in.Consumed < l.in.Bufferized && scan.IsBlank(l.in.Buffer[l.in.Consumed]) {
					l.skipBlanksRestStream()
				}
				if l.trackBlanks && l.maxValueBytes > 0 && len(l.blanks) > l.maxValueBytes {
					l.in.Err = codes.ErrMaxValueBytes

					return p.none()
				}

				continue

			case blank, tab, carriageReturn:
				if !p.tracksPosition() {
					ws := scan.ConsumeWhitespace(l.in.Buffer[l.in.Consumed:l.in.Bufferized])
					l.in.Consumed += ws
					l.in.Offset += uint64(ws)

					continue
				}
				// verbatim/state (§10.5d): first byte inline; batch-skip the rest only
				// if the run continues (b is not a newline).
				if l.trackBlanks {
					l.blanks = append(l.blanks, b)
				}
				if l.in.Consumed < l.in.Bufferized && scan.IsBlank(l.in.Buffer[l.in.Consumed]) {
					l.skipBlanksRestStream()
				}
				if l.trackBlanks && l.maxValueBytes > 0 && len(l.blanks) > l.maxValueBytes {
					l.in.Err = codes.ErrMaxValueBytes

					return p.none()
				}

				continue
			}

			// a significant byte starts a token: snapshot its position (verbatim only)
			if p.tracksPosition() {
				l.tokLine = l.line
				l.tokCol = int(l.in.Offset - l.lineStart)
			}

			if l.in.AfterKey {
				l.in.AfterKey = false
				if b != colon {
					l.in.Err = codes.ErrKeyColon

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
					l.in.Err = codes.ErrMissingObject
				} else {
					l.in.Err = codes.ErrMissingKey
				}

				return p.none()

			case closingBracket:
				if l.current.IsComma() {
					l.in.Err = codes.ErrTrailingComma

					return p.none()
				}
				if !l.isInObject() {
					l.in.Err = codes.ErrNotInObject

					return p.none()
				}

				l.in.ExpectKey = false
				l.popContainer()
				l.current = token.MakeDelimiter(token.ClosingBracket)

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			case closingSquareBracket:
				if l.current.IsComma() {
					l.in.Err = codes.ErrTrailingComma

					return p.none()
				}
				if !l.isInArray() {
					l.in.Err = codes.ErrNotInArray

					return p.none()
				}

				l.popContainer()
				l.current = token.MakeDelimiter(token.ClosingSquareBracket)

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			case comma:
				if l.current.IsComma() {
					l.in.Err = codes.ErrRepeatedComma

					return p.none()
				}
				if l.in.ExpectKey {
					l.in.Err = codes.ErrMissingKey

					return p.none()
				}
				if !l.isInContainer() {
					l.in.Err = codes.ErrCommaInContainer

					return p.none()
				}
				if l.current.IsStartObject() || l.current.IsStartArray() || l.current.IsColon() {
					l.in.Err = codes.ErrMissingValue

					return p.none()
				}

				if l.isInObject() {
					l.in.ExpectKey = true
				}

				l.current = token.MakeDelimiter(token.Comma)
				if l.elideSeparator {
					continue
				}

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			case openingBracket:
				if l.current.IsKnown() {
					if l.current.Kind() != token.Delimiter {
						l.in.Err = codes.ErrInvalidToken

						return p.none()
					}
					if l.current.Delimiter().IsClosing() {
						l.in.Err = codes.ErrMissingComma

						return p.none()
					}
					if l.isInArray() {
						if l.current.Delimiter() != token.OpeningSquareBracket &&
							l.current.Delimiter() != token.Comma {
							l.in.Err = codes.ErrMissingComma

							return p.none()
						}
					} else if !l.current.IsColon() {
						l.in.Err = codes.ErrMissingKey

						return p.none()
					}
				}
				if l.in.ExpectKey {
					l.in.Err = codes.ErrMissingKey

					return p.none()
				}

				l.in.ExpectKey = true
				l.pushObject()
				if l.in.Err != nil {
					return p.none()
				}
				l.current = token.MakeDelimiter(token.OpeningBracket)

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			case openingSquareBracket:
				if l.current.IsKnown() {
					if l.current.Kind() != token.Delimiter {
						l.in.Err = codes.ErrInvalidToken

						return p.none()
					}
					if l.current.Delimiter().IsClosing() {
						l.in.Err = codes.ErrMissingComma

						return p.none()
					}
				}
				if l.in.ExpectKey {
					l.in.Err = codes.ErrMissingKey

					return p.none()
				}

				l.pushArray()
				if l.in.Err != nil {
					return p.none()
				}
				l.current = token.MakeDelimiter(token.OpeningSquareBracket)

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			case doubleQuote:
				if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
					l.in.Err = codes.ErrDelimitedValue
					l.current = token.None

					return p.none()
				}

				l.current = l.in.ConsumeString()
				if l.in.Err != nil {
					return p.none()
				}

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			case startOfTrue, startOfFalse:
				if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
					l.in.Err = codes.ErrDelimitedValue
					l.current = token.None

					return p.none()
				}
				if l.in.ExpectKey {
					l.in.Err = codes.ErrMissingKey

					return p.none()
				}

				l.current = l.in.ConsumeBoolean(b)
				if l.in.Err != nil {
					return p.none()
				}

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			case minusSign, decimalPoint, '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
					l.in.Err = codes.ErrDelimitedValue
					l.current = token.None

					return p.none()
				}
				if l.in.ExpectKey {
					l.in.Err = codes.ErrMissingKey

					return p.none()
				}

				l.current = l.in.ConsumeNumberStreamFast(b)
				if l.in.Err != nil {
					return p.none()
				}

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			case startOfNull:
				if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
					l.in.Err = codes.ErrDelimitedValue
					l.current = token.None

					return p.none()
				}
				if l.in.ExpectKey {
					l.in.Err = codes.ErrMissingKey

					return p.none()
				}

				l.current = l.in.ConsumeNull(b)
				if l.in.Err != nil {
					return p.none()
				}

				return p.emit(l.current, l.blanks, l.tokLine, l.tokCol)

			default:
				l.in.Err = codes.ErrInvalidToken

				return p.none()
			}
		}
	}
}

// scanPushStreamG is the generic, policy-parameterized PUSH core for STREAMING input
// (io.Reader): the yield→loop counterpart of scanTokenStreamG (§10.5g). It backs
// Tokens() over a reader, replacing the previous fallthrough to a NextToken loop
// wrapped in a range-over-func closure (iterator.go): that fallthrough paid per-token
// NextToken call overhead PLUS the closure, running BELOW the streaming pull core. Here
// the cursor and scan state stay put across the whole scan and each token is delivered
// inline via yield(...) — the same win the whole-buffer push core scanPushG has over
// its pull twin, now on the streaming lane.
//
// Structurally it is scanTokenStreamG with `return p.emit(...)` replaced by
// `if !yield(p.emit(...)) { return }` and, for the position-tracking policies, an
// l.blanks reset so the next token's preceding-blanks run starts fresh (gated on the
// compile-time tracksPosition() so it folds away in the semantic core). A grammar error
// stops the range (bare return, l.in.Err already set); readMore's EOF is classified by
// errCheckG (no EOF token is yielded, matching scanPushG). Must be reached only through
// the //go:noinline devirt shim so the yield closure stays on the stack.
//
//nolint:gocognit,gocyclo
func scanPushStreamG[T any, P emitPolicy[T]](l *L, p P, yield func(T) bool) {
	if l.in.Err != nil {
		return
	}

	if l.trackBlanks {
		l.blanks = l.blanks[:0]
	}

	for {
		if err := l.in.ReadMore(); err != nil {
			_ = errCheckG(l, p, err) // classify EOF/error; push yields no EOF token

			return
		}

		for l.in.Consumed < l.in.Bufferized {
			b := l.in.Buffer[l.in.Consumed]
			l.in.Offset++
			l.in.Consumed++

			switch b {
			case lineFeed:
				if !p.tracksPosition() {
					ws := scan.ConsumeWhitespace(l.in.Buffer[l.in.Consumed:l.in.Bufferized])
					l.in.Consumed += ws
					l.in.Offset += uint64(ws)

					continue
				}
				// verbatim/state (§10.5d): first byte inline (newline), batch-skip the rest.
				l.line++
				l.lineStart = l.in.Offset
				if l.trackBlanks {
					l.blanks = append(l.blanks, b)
				}
				if l.in.Consumed < l.in.Bufferized && scan.IsBlank(l.in.Buffer[l.in.Consumed]) {
					l.skipBlanksRestStream()
				}
				if l.trackBlanks && l.maxValueBytes > 0 && len(l.blanks) > l.maxValueBytes {
					l.in.Err = codes.ErrMaxValueBytes

					return
				}

				continue

			case blank, tab, carriageReturn:
				if !p.tracksPosition() {
					ws := scan.ConsumeWhitespace(l.in.Buffer[l.in.Consumed:l.in.Bufferized])
					l.in.Consumed += ws
					l.in.Offset += uint64(ws)

					continue
				}
				if l.trackBlanks {
					l.blanks = append(l.blanks, b)
				}
				if l.in.Consumed < l.in.Bufferized && scan.IsBlank(l.in.Buffer[l.in.Consumed]) {
					l.skipBlanksRestStream()
				}
				if l.trackBlanks && l.maxValueBytes > 0 && len(l.blanks) > l.maxValueBytes {
					l.in.Err = codes.ErrMaxValueBytes

					return
				}

				continue
			}

			if p.tracksPosition() {
				l.tokLine = l.line
				l.tokCol = int(l.in.Offset - l.lineStart)
			}

			if l.in.AfterKey {
				l.in.AfterKey = false
				if b != colon {
					l.in.Err = codes.ErrKeyColon

					return
				}

				l.current = token.MakeDelimiter(token.Colon)
				if l.elideSeparator {
					continue
				}
				if !yield(p.emit(l.current, l.blanks, l.tokLine, l.tokCol)) {
					return
				}
				if p.tracksPosition() {
					l.blanks = l.blanks[:0]
				}

				continue
			}

			switch b {
			case colon:
				if l.current.Kind() == token.String {
					l.in.Err = codes.ErrMissingObject
				} else {
					l.in.Err = codes.ErrMissingKey
				}

				return

			case closingBracket:
				if l.current.IsComma() {
					l.in.Err = codes.ErrTrailingComma

					return
				}
				if !l.isInObject() {
					l.in.Err = codes.ErrNotInObject

					return
				}

				l.in.ExpectKey = false
				l.popContainer()
				l.current = token.MakeDelimiter(token.ClosingBracket)
				if !yield(p.emit(l.current, l.blanks, l.tokLine, l.tokCol)) {
					return
				}
				if p.tracksPosition() {
					l.blanks = l.blanks[:0]
				}

			case closingSquareBracket:
				if l.current.IsComma() {
					l.in.Err = codes.ErrTrailingComma

					return
				}
				if !l.isInArray() {
					l.in.Err = codes.ErrNotInArray

					return
				}

				l.popContainer()
				l.current = token.MakeDelimiter(token.ClosingSquareBracket)
				if !yield(p.emit(l.current, l.blanks, l.tokLine, l.tokCol)) {
					return
				}
				if p.tracksPosition() {
					l.blanks = l.blanks[:0]
				}

			case comma:
				if l.current.IsComma() {
					l.in.Err = codes.ErrRepeatedComma

					return
				}
				if l.in.ExpectKey {
					l.in.Err = codes.ErrMissingKey

					return
				}
				if !l.isInContainer() {
					l.in.Err = codes.ErrCommaInContainer

					return
				}
				if l.current.IsStartObject() || l.current.IsStartArray() || l.current.IsColon() {
					l.in.Err = codes.ErrMissingValue

					return
				}

				if l.isInObject() {
					l.in.ExpectKey = true
				}

				l.current = token.MakeDelimiter(token.Comma)
				if l.elideSeparator {
					continue
				}
				if !yield(p.emit(l.current, l.blanks, l.tokLine, l.tokCol)) {
					return
				}
				if p.tracksPosition() {
					l.blanks = l.blanks[:0]
				}

			case openingBracket:
				if l.current.IsKnown() {
					if l.current.Kind() != token.Delimiter {
						l.in.Err = codes.ErrInvalidToken

						return
					}
					if l.current.Delimiter().IsClosing() {
						l.in.Err = codes.ErrMissingComma

						return
					}
					if l.isInArray() {
						if l.current.Delimiter() != token.OpeningSquareBracket &&
							l.current.Delimiter() != token.Comma {
							l.in.Err = codes.ErrMissingComma

							return
						}
					} else if !l.current.IsColon() {
						l.in.Err = codes.ErrMissingKey

						return
					}
				}
				if l.in.ExpectKey {
					l.in.Err = codes.ErrMissingKey

					return
				}

				l.in.ExpectKey = true
				l.pushObject()
				if l.in.Err != nil {
					return
				}
				l.current = token.MakeDelimiter(token.OpeningBracket)
				if !yield(p.emit(l.current, l.blanks, l.tokLine, l.tokCol)) {
					return
				}
				if p.tracksPosition() {
					l.blanks = l.blanks[:0]
				}

			case openingSquareBracket:
				if l.current.IsKnown() {
					if l.current.Kind() != token.Delimiter {
						l.in.Err = codes.ErrInvalidToken

						return
					}
					if l.current.Delimiter().IsClosing() {
						l.in.Err = codes.ErrMissingComma

						return
					}
				}
				if l.in.ExpectKey {
					l.in.Err = codes.ErrMissingKey

					return
				}

				l.pushArray()
				if l.in.Err != nil {
					return
				}
				l.current = token.MakeDelimiter(token.OpeningSquareBracket)
				if !yield(p.emit(l.current, l.blanks, l.tokLine, l.tokCol)) {
					return
				}
				if p.tracksPosition() {
					l.blanks = l.blanks[:0]
				}

			case doubleQuote:
				if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
					l.in.Err = codes.ErrDelimitedValue
					l.current = token.None

					return
				}

				l.current = l.in.ConsumeString()
				if l.in.Err != nil {
					return
				}
				if !yield(p.emit(l.current, l.blanks, l.tokLine, l.tokCol)) {
					return
				}
				if p.tracksPosition() {
					l.blanks = l.blanks[:0]
				}

			case startOfTrue, startOfFalse:
				if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
					l.in.Err = codes.ErrDelimitedValue
					l.current = token.None

					return
				}
				if l.in.ExpectKey {
					l.in.Err = codes.ErrMissingKey

					return
				}

				l.current = l.in.ConsumeBoolean(b)
				if l.in.Err != nil {
					return
				}
				if !yield(p.emit(l.current, l.blanks, l.tokLine, l.tokCol)) {
					return
				}
				if p.tracksPosition() {
					l.blanks = l.blanks[:0]
				}

			case minusSign, decimalPoint, '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
					l.in.Err = codes.ErrDelimitedValue
					l.current = token.None

					return
				}
				if l.in.ExpectKey {
					l.in.Err = codes.ErrMissingKey

					return
				}

				l.current = l.in.ConsumeNumberStreamFast(b)
				if l.in.Err != nil {
					return
				}
				if !yield(p.emit(l.current, l.blanks, l.tokLine, l.tokCol)) {
					return
				}
				if p.tracksPosition() {
					l.blanks = l.blanks[:0]
				}

			case startOfNull:
				if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
					l.in.Err = codes.ErrDelimitedValue
					l.current = token.None

					return
				}
				if l.in.ExpectKey {
					l.in.Err = codes.ErrMissingKey

					return
				}

				l.current = l.in.ConsumeNull(b)
				if l.in.Err != nil {
					return
				}
				if !yield(p.emit(l.current, l.blanks, l.tokLine, l.tokCol)) {
					return
				}
				if p.tracksPosition() {
					l.blanks = l.blanks[:0]
				}

			default:
				l.in.Err = codes.ErrInvalidToken

				return
			}
		}
	}
}

// scanTokenBufferG is the generic, policy-parameterized pull core for WHOLE-BUFFER
// input: it scans and returns exactly one token, dispatched from L.NextToken /
// VL.NextToken when l.in.WholeBuffer is true. It is the yield→return counterpart of
// scanPushG (roadmap §10) — the same proven whole-buffer shape: the cursor is a
// pure local i (no readMore, no per-byte struct write), and the preceding blanks
// are a zero-copy slice of the input (data[blankStart:i:i], as push does) rather
// than the byte-by-byte l.blanks append the streaming core needs. It resumes from
// l.in.Consumed on the next call and only continues its loop past elided separators
// and whitespace; every value/delimiter token returns immediately.
//
// Keeping this separate from scanTokenStreamG is the whole point of the split: the
// register-delicate buffer loop (§9.1, source of the 16/18 corpus wins) is frozen
// here, and stream-refill optimization happens next door without touching it.
//
//nolint:gocognit,gocyclo
func scanTokenBufferG[T any, P emitPolicy[T]](l *L, p P) T {
	if l.in.Err != nil {
		return p.none()
	}

	data := l.in.Buffer[:l.in.Bufferized]
	i := l.in.Consumed
	// blankStart is the index right after the previous token: [blankStart:i] is the
	// whitespace run the verbatim policy bakes into the token (zero-copy). It is
	// only advanced past elided separators; every emit resets it via the next call's
	// re-init from l.in.Consumed.
	blankStart := i

	writeback := func(pos int) {
		l.in.Consumed = pos
		l.in.Offset = uint64(pos)
	}

	for i < len(data) {
		b := data[i]

		switch b {
		case lineFeed:
			if !p.tracksPosition() {
				i += scan.ConsumeWhitespace(data[i:]) // semantic batch-skip (citm bottleneck)

				continue
			}
			i++
			l.line++
			l.lineStart = uint64(i)

			continue
		case blank, tab, carriageReturn:
			if !p.tracksPosition() {
				i += scan.ConsumeWhitespace(data[i:])

				continue
			}
			i++

			continue
		}

		if p.tracksPosition() {
			l.tokLine = l.line
			l.tokCol = int(uint64(i+1) - l.lineStart)
		}
		// the whitespace run since the previous token (zero-copy); i is the token start.
		blanks := data[blankStart:i:i]
		if p.storesBlanks() {
			l.blanks = blanks // state-based VL: stash the leading blanks in lexer state
		}

		// verbatim circuit breaker: bound the preceding-whitespace run. The stream
		// core checks this per-byte as it appends; here blanks is a zero-copy slice,
		// so one length check at the token boundary is equivalent (folded away in the
		// semantic core, where tracksPosition() is a false constant).
		if p.tracksPosition() && l.maxValueBytes > 0 && len(blanks) > l.maxValueBytes {
			l.in.Err = codes.ErrMaxValueBytes
			writeback(i)

			return p.none()
		}

		if l.in.AfterKey {
			l.in.AfterKey = false
			if b != colon {
				l.in.Err = codes.ErrKeyColon
				writeback(i + 1)

				return p.none()
			}

			i++
			l.current = token.MakeDelimiter(token.Colon)
			blankStart = i
			if l.elideSeparator {
				continue
			}
			writeback(i)

			return p.emit(l.current, blanks, l.tokLine, l.tokCol)
		}

		switch b {
		case colon:
			if l.current.Kind() == token.String {
				l.in.Err = codes.ErrMissingObject
			} else {
				l.in.Err = codes.ErrMissingKey
			}
			writeback(i + 1)

			return p.none()

		case closingBracket:
			if l.current.IsComma() {
				l.in.Err = codes.ErrTrailingComma
				writeback(i + 1)

				return p.none()
			}
			if !l.isInObject() {
				l.in.Err = codes.ErrNotInObject
				writeback(i + 1)

				return p.none()
			}

			i++
			l.in.ExpectKey = false
			l.popContainer()
			l.current = token.MakeDelimiter(token.ClosingBracket)
			writeback(i)

			return p.emit(l.current, blanks, l.tokLine, l.tokCol)

		case closingSquareBracket:
			if l.current.IsComma() {
				l.in.Err = codes.ErrTrailingComma
				writeback(i + 1)

				return p.none()
			}
			if !l.isInArray() {
				l.in.Err = codes.ErrNotInArray
				writeback(i + 1)

				return p.none()
			}

			i++
			l.popContainer()
			l.current = token.MakeDelimiter(token.ClosingSquareBracket)
			writeback(i)

			return p.emit(l.current, blanks, l.tokLine, l.tokCol)

		case comma:
			if l.current.IsComma() {
				l.in.Err = codes.ErrRepeatedComma
				writeback(i + 1)

				return p.none()
			}
			if l.in.ExpectKey {
				l.in.Err = codes.ErrMissingKey
				writeback(i + 1)

				return p.none()
			}
			if !l.isInContainer() {
				l.in.Err = codes.ErrCommaInContainer
				writeback(i + 1)

				return p.none()
			}
			if l.current.IsStartObject() || l.current.IsStartArray() || l.current.IsColon() {
				l.in.Err = codes.ErrMissingValue
				writeback(i + 1)

				return p.none()
			}

			if l.isInObject() {
				l.in.ExpectKey = true
			}

			i++
			l.current = token.MakeDelimiter(token.Comma)
			blankStart = i
			if l.elideSeparator {
				continue
			}
			writeback(i)

			return p.emit(l.current, blanks, l.tokLine, l.tokCol)

		case openingBracket:
			if l.current.IsKnown() {
				if l.current.Kind() != token.Delimiter {
					l.in.Err = codes.ErrInvalidToken
					writeback(i + 1)

					return p.none()
				}
				if l.current.Delimiter().IsClosing() {
					l.in.Err = codes.ErrMissingComma
					writeback(i + 1)

					return p.none()
				}
				if l.isInArray() {
					if l.current.Delimiter() != token.OpeningSquareBracket &&
						l.current.Delimiter() != token.Comma {
						l.in.Err = codes.ErrMissingComma
						writeback(i + 1)

						return p.none()
					}
				} else if !l.current.IsColon() {
					l.in.Err = codes.ErrMissingKey
					writeback(i + 1)

					return p.none()
				}
			}
			if l.in.ExpectKey {
				l.in.Err = codes.ErrMissingKey
				writeback(i + 1)

				return p.none()
			}

			i++
			l.in.ExpectKey = true
			l.pushObject()
			if l.in.Err != nil {
				writeback(i)

				return p.none()
			}
			l.current = token.MakeDelimiter(token.OpeningBracket)
			writeback(i)

			return p.emit(l.current, blanks, l.tokLine, l.tokCol)

		case openingSquareBracket:
			if l.current.IsKnown() {
				if l.current.Kind() != token.Delimiter {
					l.in.Err = codes.ErrInvalidToken
					writeback(i + 1)

					return p.none()
				}
				if l.current.Delimiter().IsClosing() {
					l.in.Err = codes.ErrMissingComma
					writeback(i + 1)

					return p.none()
				}
			}
			if l.in.ExpectKey {
				l.in.Err = codes.ErrMissingKey
				writeback(i + 1)

				return p.none()
			}

			i++
			l.pushArray()
			if l.in.Err != nil {
				writeback(i)

				return p.none()
			}
			l.current = token.MakeDelimiter(token.OpeningSquareBracket)
			writeback(i)

			return p.emit(l.current, blanks, l.tokLine, l.tokCol)

		case doubleQuote:
			if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
				l.in.Err = codes.ErrDelimitedValue
				l.current = token.None
				writeback(i + 1)

				return p.none()
			}

			writeback(i + 1)
			l.current = l.in.ConsumeString()
			if l.in.Err != nil {
				return p.none()
			}
			i = l.in.Consumed

			return p.emit(l.current, blanks, l.tokLine, l.tokCol)

		case startOfTrue, startOfFalse:
			if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
				l.in.Err = codes.ErrDelimitedValue
				l.current = token.None
				writeback(i + 1)

				return p.none()
			}
			if l.in.ExpectKey {
				l.in.Err = codes.ErrMissingKey
				writeback(i + 1)

				return p.none()
			}

			writeback(i + 1)
			l.current = l.in.ConsumeBoolean(b)
			if l.in.Err != nil {
				return p.none()
			}
			i = l.in.Consumed

			return p.emit(l.current, blanks, l.tokLine, l.tokCol)

		case minusSign, decimalPoint, '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
				l.in.Err = codes.ErrDelimitedValue
				l.current = token.None
				writeback(i + 1)

				return p.none()
			}
			if l.in.ExpectKey {
				l.in.Err = codes.ErrMissingKey
				writeback(i + 1)

				return p.none()
			}

			// Bounded values (maxValueBytes) route to the streaming number consumer,
			// which enforces the cap; the inline fast path + consumeNumberWhole below
			// do not check it. In whole-buffer mode this gate reduces to maxValueBytes,
			// so the unbounded common case keeps the full inline fast path (§10 keeps
			// the champion shape; the old combined core gated it the same way).
			if l.maxValueBytes > 0 {
				writeback(i + 1)
				l.current = l.in.ConsumeNumberStreaming(b)
				if l.in.Err != nil {
					return p.none()
				}
				i = l.in.Consumed

				return p.emit(l.current, blanks, l.tokLine, l.tokCol)
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
				end := n
				var term byte
				if uint(end) < uint(len(data)) {
					term = data[end]
				}
				// extend over a fractional part ('.' 1*DIGIT); a trailing dot leaves
				// term==decimalPoint for consumeNumberWhole's error.
				if !leadingZero && term == decimalPoint {
					m := end + 1
					for uint(m) < uint(len(data)) && '0' <= data[m] && data[m] <= '9' {
						m++
					}
					if m > end+1 { // at least one fractional digit
						end = m
						term = 0
						if uint(end) < uint(len(data)) {
							term = data[end]
						}
					}
				}
				// extend over an exponent ((e|E) [+|-] 1*DIGIT); cheaper than bailing to
				// consumeNumberWhole and re-scanning. A malformed exponent leaves
				// term=='e'/'E' for the slow-path error.
				if !leadingZero && (term == 'e' || term == 'E') {
					m := end + 1
					if uint(m) < uint(len(data)) && (data[m] == '+' || data[m] == '-') {
						m++
					}
					expStart := m
					for uint(m) < uint(len(data)) && '0' <= data[m] && data[m] <= '9' {
						m++
					}
					if m > expStart { // at least one exponent digit
						end = m
						term = 0
						if uint(end) < uint(len(data)) {
							term = data[end]
						}
					}
				}

				if !leadingZero && term != decimalPoint && term != 'e' && term != 'E' {
					l.current = token.MakeWithValue(token.Number, data[numStart:end:end])
					i = end
					writeback(end)

					return p.emit(l.current, blanks, l.tokLine, l.tokCol)
				}
			}

			writeback(i + 1)
			l.current = l.in.ConsumeNumberWhole(b)
			if l.in.Err != nil {
				return p.none()
			}
			i = l.in.Consumed

			return p.emit(l.current, blanks, l.tokLine, l.tokCol)

		case startOfNull:
			if l.current.IsKnown() && !l.current.Delimiter().AcceptValue() {
				l.in.Err = codes.ErrDelimitedValue
				l.current = token.None
				writeback(i + 1)

				return p.none()
			}
			if l.in.ExpectKey {
				l.in.Err = codes.ErrMissingKey
				writeback(i + 1)

				return p.none()
			}

			writeback(i + 1)
			l.current = l.in.ConsumeNull(b)
			if l.in.Err != nil {
				return p.none()
			}
			i = l.in.Consumed

			return p.emit(l.current, blanks, l.tokLine, l.tokCol)

		default:
			l.in.Err = codes.ErrInvalidToken
			writeback(i + 1)

			return p.none()
		}
	}

	// buffer exhausted: the trailing whitespace run [blankStart:i] is the verbatim
	// EOF token's preceding blanks (zero-copy); the semantic policy ignores it.
	l.blanks = data[blankStart:i:i]
	writeback(i)

	// verbatim circuit breaker on a trailing whitespace flood with no closing token
	// (parity with the stream core's per-byte check); folded away in the semantic core.
	if p.tracksPosition() && l.maxValueBytes > 0 && len(l.blanks) > l.maxValueBytes {
		l.in.Err = codes.ErrMaxValueBytes

		return p.none()
	}

	return errCheckG(l, p, io.EOF)
}
