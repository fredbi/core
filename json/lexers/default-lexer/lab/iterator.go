package lab

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
		// whole-buffer fast path: a native push scan loop that keeps the cursor
		// in a local across the whole scan (no per-byte struct writes). Streaming
		// and value-capped modes keep the proven NextToken loop.
		if l.wholeBuffer && l.maxValueBytes == 0 {
			// generics spike: route the whole-buffer push path through the
			// policy-parameterized core via a non-generic wrapper (keeps Tokens
			// inlinable; see scanPushSemantic).
			l.scanPushSemantic(yield)

			return
		}

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

// Tokens returns an iterator over the verbatim JSON tokens. In whole-buffer mode
// it takes the native push path (the generic core with the verbatim policy),
// which gives VL all of L's fast paths; streaming/value-capped modes keep the
// proven NextToken loop. See [L.Tokens] for the semantics.
func (l *VL) Tokens() iter.Seq[token.VT] {
	return func(yield func(token.VT) bool) {
		if l.wholeBuffer && l.maxValueBytes == 0 {
			l.scanPushVerbatim(yield)

			return
		}

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
