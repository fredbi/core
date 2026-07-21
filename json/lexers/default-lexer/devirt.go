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

//go:noinline
func (l *L) scanPushVerbatimDevirt(yield func(token.VT) bool) {
	scanPushVerbatimCore(l, verbatimPolicy{}, yield)
}

// scanPushStateDevirt is the push shim for the prototype state-based verbatim lexer
// [VS] (§10.5b): emits the light token.T while the core stashes blanks/position in
// lexer state. Same //go:noinline discipline as the others.
//
//go:noinline
func (l *L) scanPushStateDevirt(yield func(token.T) bool) {
	scanPushStateCore(l, statePolicy{}, yield)
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
func (l *L) scanPushStreamVerbatimDevirt(yield func(token.VT) bool) {
	scanPushStreamVerbatimCore(l, verbatimPolicy{}, yield)
}

//go:noinline
func (l *L) scanPushStreamStateDevirt(yield func(token.T) bool) {
	scanPushStreamStateCore(l, statePolicy{}, yield)
}

