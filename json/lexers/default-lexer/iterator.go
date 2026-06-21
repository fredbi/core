package lexer

import (
	"iter"

	"github.com/fredbi/core/json/lexers/token"
)

// Tokens returns an iterator over the JSON tokens, as a convenience over the
// [L.NextToken] loop.
//
// The range yields every token up to (but not including) EOF, then ends. It also
// ends on error. As with NextToken, errors are kept in the lexer's state: check
// [L.Ok] / [L.Err] after the loop.
//
//	for tok := range lex.Tokens() {
//	    // use tok
//	}
//	if !lex.Ok() {
//	    // handle lex.Err()
//	}
//
// The iterator is single-pass: it consumes from the same input as NextToken and
// does not rewind.
func (l *L) Tokens() iter.Seq[token.T] {
	return func(yield func(token.T) bool) {
		for {
			tok := l.NextToken()
			if l.err != nil {
				return
			}
			if tok.Kind() == token.EOF {
				return
			}
			if !yield(tok) {
				return
			}
		}
	}
}

// Tokens returns an iterator over the verbatim JSON tokens, as a convenience
// over the [VL.NextToken] loop. See [L.Tokens] for the semantics.
func (l *VL) Tokens() iter.Seq[token.VT] {
	return func(yield func(token.VT) bool) {
		for {
			tok := l.NextToken()
			if l.err != nil {
				return
			}
			if tok.Kind() == token.EOF {
				return
			}
			if !yield(tok) {
				return
			}
		}
	}
}
