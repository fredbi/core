// Package easyjson drives mailru/easyjson's jlexer over a whole JSON document as
// a generic recursive walk, so it can be benchmarked as a comparison point for
// the default-lexer (both are pull-style, []byte-only lexers; easyjson is the
// original inspiration for the default-lexer).
//
// To compare apples to apples with the default-lexer — which emits raw token
// bytes and never converts numbers to native Go types — numbers here are taken
// via Raw() (the raw sub-slice, no numeric conversion). Strings are taken via
// String() (easyjson's typical, non-mutating path; UnsafeBytes would avoid the
// allocation but rewrites escapes in place, corrupting a reused fixture).
package easyjson

import "github.com/mailru/easyjson/jlexer"

// Sink prevents the compiler from eliminating the walk.
var Sink int

// Walk fully tokenizes data with jlexer, descending into every container, and
// returns the lexer's error state (nil on success).
func Walk(data []byte) error {
	l := &jlexer.Lexer{Data: data}
	walkValue(l)

	return l.Error()
}

func walkValue(l *jlexer.Lexer) {
	switch l.CurrentToken() {
	case jlexer.TokenString:
		Sink += len(l.String())

	case jlexer.TokenNumber:
		Sink += len(l.Raw()) // raw bytes, no numeric conversion

	case jlexer.TokenBool:
		if l.Bool() {
			Sink++
		}

	case jlexer.TokenNull:
		l.Null()

	case jlexer.TokenDelim:
		switch {
		case l.IsDelim('{'):
			l.Delim('{')
			for l.Ok() && !l.IsDelim('}') {
				Sink += len(l.String()) // key
				l.WantColon()
				walkValue(l)
				l.WantComma()
			}
			l.Delim('}')

		case l.IsDelim('['):
			l.Delim('[')
			for l.Ok() && !l.IsDelim(']') {
				walkValue(l)
				l.WantComma()
			}
			l.Delim(']')
		}

	default: // TokenUndef
	}
}
