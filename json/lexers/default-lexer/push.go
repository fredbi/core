package lexer

import (
	"iter"
	"unicode/utf8"

	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/lexers/token"
)

// P is a PROTOTYPE push-style lexer over a whole []byte buffer.
//
// It exists to validate the phase 2 "push core" design against the pull-based L
// and against mailru/easyjson. The scan loop drives itself and yields tokens via
// a callback (range-over-func), keeping the cursor and scan state in locals so
// the hot loop performs no per-byte writes to the struct, folds terminator
// detection into the loop (no separate look-ahead pass), aliases number and
// unescaped-string values into the input, and copies only escaped strings.
//
// Scope/caveats (prototype): bytes mode only; structural error detection is a
// subset of L's (it tokenizes valid JSON correctly and rejects the common
// errors, but full JSONTestSuite conformance is not a goal yet); separators are
// elided like L's default; line/column tracking is intentionally omitted to
// measure the raw scan ceiling.
type P struct {
	data  []byte
	err   error
	value []byte // reused buffer for escaped string values
}

// NewPush builds a prototype push lexer over data.
func NewPush(data []byte) *P {
	return &P{data: data}
}

// Err returns the error state after iterating.
func (p *P) Err() error { return p.err }

// Ok reports whether no error has occurred.
func (p *P) Ok() bool { return p.err == nil }

// Tokens iterates over the JSON tokens (separators elided), up to EOF or error.
func (p *P) Tokens() iter.Seq[token.T] {
	return func(yield func(token.T) bool) {
		p.scan(yield)
	}
}

//nolint:gocognit,gocyclo
func (p *P) scan(yield func(token.T) bool) {
	data := p.data
	n := len(data)
	i := 0

	// bit-packed container stack (same scheme as stack.go); kept local.
	stack := []uint64{1}
	last := 0 // index of the top word == len(stack)-1
	expectKey := false

	fail := func(e error) {
		p.err = e
	}

	for i < n {
		c := data[i]

		// skip insignificant whitespace
		if c == blank || c == tab || c == carriageReturn || c == lineFeed {
			i++
			continue
		}

		switch c {
		case openingBracket: // {
			i++
			// move up the stack with an even (object) bit
			top := stack[last]
			if top >= maxStack {
				stack = append(stack, 0b10)
				last++
			} else {
				stack[last] = top << 1
			}
			expectKey = true
			if !yield(token.MakeDelimiter(token.OpeningBracket)) {
				return
			}

		case openingSquareBracket: // [
			i++
			top := stack[last]
			if top >= maxStack {
				stack = append(stack, 0b11)
				last++
			} else {
				stack[last] = top<<1 | 1
			}
			expectKey = false
			if !yield(token.MakeDelimiter(token.OpeningSquareBracket)) {
				return
			}

		case closingBracket: // }
			top := stack[last]
			if !(top > 1 && top&1 == 0) { // not in object
				fail(codes.ErrNotInObject)
				return
			}
			i++
			expectKey = false
			if top > 1 {
				stack[last] = top >> 1
			}
			if stack[last] <= 1 && last > 0 {
				stack = stack[:last]
				last--
			}
			if !yield(token.MakeDelimiter(token.ClosingBracket)) {
				return
			}

		case closingSquareBracket: // ]
			top := stack[last]
			if !(top > 1 && top&1 == 1) { // not in array
				fail(codes.ErrNotInArray)
				return
			}
			i++
			stack[last] = top >> 1
			if stack[last] <= 1 && last > 0 {
				stack = stack[:last]
				last--
			}
			if !yield(token.MakeDelimiter(token.ClosingSquareBracket)) {
				return
			}

		case comma: // elided, but validated
			top := stack[last]
			if !(last > 0 || top > 1) { // not in a container
				fail(codes.ErrCommaInContainer)
				return
			}
			if top > 1 && top&1 == 0 { // in object
				expectKey = true
			}
			i++

		case colon: // elided
			i++

		case doubleQuote: // string or key
			end, val, err := p.scanString(data, i+1)
			if err != nil {
				fail(err)
				return
			}
			i = end
			top := stack[last]
			inObject := top > 1 && top&1 == 0
			kind := token.String
			if expectKey && inObject {
				kind = token.Key
				expectKey = false
			}
			if !yield(token.MakeWithValue(kind, val)) {
				return
			}

		case minusSign, decimalPoint, '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			end, ok := scanNumber(data, i)
			if !ok {
				fail(codes.ErrInvalidToken)
				return
			}
			tok := token.MakeWithValue(token.Number, data[i:end:end])
			i = end
			if !yield(tok) {
				return
			}

		case startOfTrue:
			if i+4 > n || data[i+1] != 'r' || data[i+2] != 'u' || data[i+3] != 'e' {
				fail(codes.ErrInvalidToken)
				return
			}
			i += 4
			if !yield(token.MakeBoolean(true)) {
				return
			}

		case startOfFalse:
			if i+5 > n || data[i+1] != 'a' || data[i+2] != 'l' || data[i+3] != 's' || data[i+4] != 'e' {
				fail(codes.ErrInvalidToken)
				return
			}
			i += 5
			if !yield(token.MakeBoolean(false)) {
				return
			}

		case startOfNull:
			if i+4 > n || data[i+1] != 'u' || data[i+2] != 'l' || data[i+3] != 'l' {
				fail(codes.ErrInvalidToken)
				return
			}
			i += 4
			if !yield(token.NullToken) {
				return
			}

		default:
			fail(codes.ErrInvalidToken)
			return
		}
	}

	// end of input: report unterminated containers
	if last > 0 || stack[last] > 1 {
		if stack[last] > 1 && stack[last]&1 == 1 {
			fail(codes.ErrNotInArray)
		} else {
			fail(codes.ErrNotInObject)
		}
	}
}

// scanNumber validates and finds the extent of a JSON number starting at start.
// It returns the index just past the number and whether it is well-formed.
// Position is a pure local; no struct writes.
func scanNumber(data []byte, start int) (int, bool) {
	n := len(data)
	i := start

	var (
		isExponent     bool
		exponentSign   bool
		hasLeadingZero bool
		hasFractional  bool
		isFractional   bool
		integerPart    int
		fractionalPart int
		exponentPart   int
	)

	switch data[i] {
	case decimalPoint:
		hasFractional = true
		isFractional = true
	case '0':
		hasLeadingZero = true
		integerPart++
	case minusSign:
		// leading sign: nothing yet
	default: // 1..9
		integerPart++
	}
	i++

	for i < n {
		c := data[i]
		switch {
		case c == decimalPoint:
			if hasFractional || isExponent {
				return i, false
			}
			hasFractional = true
			isFractional = true
		case c == '+' || c == '-':
			if !isExponent || exponentPart > 0 || exponentSign {
				return i, false
			}
			exponentSign = true
		case c == 'e' || c == 'E':
			if isExponent {
				return i, false
			}
			isExponent = true
			isFractional = false
		case c == '0':
			if hasLeadingZero && !isFractional && !isExponent {
				return i, false
			}
			switch {
			case isFractional:
				fractionalPart++
			case isExponent:
				exponentPart++
			default:
				integerPart++
				if integerPart == 1 {
					hasLeadingZero = true
				}
			}
		case c >= '1' && c <= '9':
			if hasLeadingZero && !isFractional && !isExponent {
				return i, false
			}
			switch {
			case isFractional:
				fractionalPart++
			case isExponent:
				exponentPart++
			default:
				integerPart++
			}
		default:
			goto done
		}
		i++
	}

done:
	if hasFractional && fractionalPart == 0 {
		return i, false
	}
	if isExponent && exponentPart == 0 {
		return i, false
	}
	if hasLeadingZero && integerPart > 1 {
		return i, false
	}
	if integerPart == 0 {
		return i, false
	}

	return i, true
}

// scanString scans a string body starting at start (just past the opening
// quote). It aliases the input when there are no escapes, and falls back to
// copying into p.value (unescaping) on the first escape. Returns the index just
// past the closing quote and the value bytes.
func (p *P) scanString(data []byte, start int) (int, []byte, error) {
	n := len(data)
	i := start

	// fast path: scan for the closing quote or the first escape / control char
	for i < n {
		c := data[i]
		if c == doubleQuote {
			// no escapes seen: alias the input (cap == len)
			return i + 1, data[start:i:i], nil
		}
		if c == escape {
			break
		}
		if c < 0x20 {
			return i, nil, codes.ErrControlChar
		}
		i++
	}
	if i >= n {
		return i, nil, codes.ErrUnterminatedString
	}

	// slow path: copy the clean prefix, then unescape the remainder
	p.value = append(p.value[:0], data[start:i]...)

	for i < n {
		c := data[i]
		switch {
		case c == doubleQuote:
			return i + 1, p.value, nil
		case c == escape:
			i++
			if i >= n {
				return i, nil, codes.ErrUnterminatedString
			}
			switch data[i] {
			case doubleQuote:
				p.value = append(p.value, '"')
			case escape:
				p.value = append(p.value, '\\')
			case slash:
				p.value = append(p.value, '/')
			case 'b':
				p.value = append(p.value, '\b')
			case 'f':
				p.value = append(p.value, '\f')
			case 'n':
				p.value = append(p.value, '\n')
			case 't':
				p.value = append(p.value, '\t')
			case 'r':
				p.value = append(p.value, '\r')
			case 'u':
				if i+4 >= n {
					return i, nil, codes.ErrUnicodeEscape
				}
				r, ok := unhex4(data[i+1 : i+5])
				if !ok {
					return i, nil, codes.ErrUnicodeEscape
				}
				i += 4
				p.value = utf8.AppendRune(p.value, r)
			default:
				return i, nil, codes.ErrUnknownEscape
			}
			i++
		case c < 0x20:
			return i, nil, codes.ErrControlChar
		default:
			p.value = append(p.value, c)
			i++
		}
	}

	return i, nil, codes.ErrUnterminatedString
}

// unhex4 decodes 4 hex digits to a rune (no surrogate-pair handling in the
// prototype). Returns false on a non-hex digit.
func unhex4(b []byte) (rune, bool) {
	h0, ok0 := unhex(b[0])
	h1, ok1 := unhex(b[1])
	h2, ok2 := unhex(b[2])
	h3, ok3 := unhex(b[3])
	if !ok0 || !ok1 || !ok2 || !ok3 {
		return utf8.RuneError, false
	}

	return rune(uint32(h0)<<12 | uint32(h1)<<8 | uint32(h2)<<4 | uint32(h3)), true
}
