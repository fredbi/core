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
//
// Tokens and [L.NextToken] share the lexer's state, so the two may be
// interleaved: you can range over Tokens, break, and then continue with
// NextToken (or the reverse) — NextToken resumes exactly where the range
// stopped. Note the standard range-over-func semantics: the token delivered in
// the iteration where you break has already been consumed, so the next
// NextToken returns the following token, not a repeat.
func (l *L) Tokens() iter.Seq[token.T] {
	return func(yield func(token.T) bool) {
		l.primeStream() // §10.5f: resolve the whole-buffer short-circuit before choosing the core
		// whole-buffer fast path: a native push scan loop that keeps the cursor
		// in a local across the whole scan (no per-byte struct writes). Streaming
		// and value-capped modes keep the proven NextToken loop.
		if l.wholeBuffer && l.maxValueBytes == 0 {
			// route the whole-buffer push path through the DEVIRTUALIZED core (adopted
			// 2026-06-27: +7..+18% over the generic core on push, measured in-binary;
			// see devirt_bench_test + plan §5.1). The generic shim scanPushSemantic is
			// retained as the A/B baseline. Non-inlined wrapper keeps Tokens inlinable.
			l.scanPushSemanticDevirt(yield)

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
		l.primeStream() // §10.5f: resolve the whole-buffer short-circuit before choosing the core
		if l.wholeBuffer && l.maxValueBytes == 0 {
			l.scanPushVerbatimDevirt(yield)

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
