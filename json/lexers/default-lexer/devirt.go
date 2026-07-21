package lexer

import (
	"github.com/fredbi/core/json/lexers/token"
)

// Devirtualized push shims. Both pull and push are now ADOPTED:
//   - pull: NextToken (L and VL) calls scanTokenSemantic/scanTokenVerbatim directly
//     (see lexer.go / verbatim.go) — a plain return, no shim needed.
//   - push: Tokens (L and VL) routes through the //go:noinline shims below (the
//     range-over-func yield closure must not escape across the package boundary).
//
// The generated concrete cores (scan_gen.go) replace the generics-dictionary calls
// with direct, inlined policy calls. The generic cores in generic.go are retained
// as lexgen's source-of-truth and the A/B baseline (the *Generic test helpers in
// devirt_test.go drive them).

// scanPushSemanticDevirt is the devirt counterpart of scanPushSemantic: same
// //go:noinline shim discipline (so the range-over-func yield closure stays on
// the stack), calling the generated concrete push core.
//
//go:noinline
func (l *L) scanPushSemanticDevirt(yield func(token.T) bool) {
	scanPushSemanticCore(l, semanticPolicy{}, yield)
}

// scanPushVerbatimDevirt is the verbatim counterpart: the verbatim policy emits the
// light token.T and stashes blanks/position in lexer state.
//
//go:noinline
func (l *L) scanPushVerbatimDevirt(yield func(token.T) bool) {
	scanPushVerbatimCore(l, verbatimPolicy{}, yield)
}

// Streaming push shims (§10.5g): the io.Reader counterparts of the whole-buffer push
// shims above, calling the streaming push cores (scanPushStream*Core) that refill the
// buffer as they yield. Same //go:noinline discipline so the range-over-func yield
// closure stays on the stack. Tokens() over a reader routes here instead of the old
// NextToken-loop-in-a-closure fallthrough.

//go:noinline
func (l *L) scanPushStreamSemanticDevirt(yield func(token.T) bool) {
	scanPushStreamSemanticCore(l, semanticPolicy{}, yield)
}

//go:noinline
func (l *L) scanPushStreamVerbatimDevirt(yield func(token.T) bool) {
	scanPushStreamVerbatimCore(l, verbatimPolicy{}, yield)
}

