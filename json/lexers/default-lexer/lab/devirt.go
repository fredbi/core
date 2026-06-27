package lab

import (
	"iter"

	"github.com/fredbi/core/json/lexers/token"
)

// Devirtualized entry points. These mirror NextToken / Tokens exactly but call
// the generated, monomorphized scan cores (scan_gen.go) instead of the generic
// ones, so the per-token policy calls are direct and inline rather than routed
// through the generics dictionary. Both paths coexist (no build tags, no dispatch
// layer) so the devirt gap can be measured in one binary — see devirt_bench_test.
// They are unexported: only the lab's own A/B test and benchmark use them.

// nextTokenDevirt is the devirtualized counterpart of [L.NextToken].
func (l *L) nextTokenDevirt() token.T { return scanTokenSemantic(l, semanticPolicy{}) }

// nextTokenDevirt is the devirtualized counterpart of [VL.NextToken].
func (l *VL) nextTokenDevirt() token.VT { return scanTokenVerbatim(l.L, verbatimPolicy{}) }

// scanPushSemanticDevirt is the devirt counterpart of scanPushSemantic: same
// //go:noinline shim discipline (so the range-over-func yield closure stays on
// the stack), calling the generated concrete push core.
//
//go:noinline
func (l *L) scanPushSemanticDevirt(yield func(token.T) bool) {
	scanPushSemanticCore(l, semanticPolicy{}, yield)
}

//go:noinline
func (l *L) scanPushVerbatimDevirt(yield func(token.VT) bool) {
	scanPushVerbatimCore(l, verbatimPolicy{}, yield)
}

// tokensDevirt mirrors [L.Tokens] but routes the whole-buffer push path through
// the devirtualized core.
func (l *L) tokensDevirt() iter.Seq[token.T] {
	return func(yield func(token.T) bool) {
		if l.wholeBuffer && l.maxValueBytes == 0 {
			l.scanPushSemanticDevirt(yield)

			return
		}

		for {
			tok := l.nextTokenDevirt()
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

// tokensDevirt mirrors [VL.Tokens] but routes the whole-buffer push path through
// the devirtualized verbatim core.
func (l *VL) tokensDevirt() iter.Seq[token.VT] {
	return func(yield func(token.VT) bool) {
		if l.wholeBuffer && l.maxValueBytes == 0 {
			l.scanPushVerbatimDevirt(yield)

			return
		}

		for {
			tok := l.nextTokenDevirt()
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
