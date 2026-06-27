package lab

import (
	"github.com/fredbi/core/json/lexers/token"
)

// Devirtualized entry points calling the generated, monomorphized scan cores
// (scan_gen.go) instead of the generic ones, so the per-token policy calls are
// direct and inline rather than routed through the generics dictionary.
//
// The push shims are ADOPTED: Tokens() (L and VL) routes through them (see
// iterator.go) — measured +7..+18% over the generic core. The generic push shims
// (scanPushSemantic/scanPushVerbatim in generic.go) are retained as the A/B
// baseline, exercised by devirt_bench_test via the tokensGeneric test helpers.
//
// Pull (nextTokenDevirt) is NOT adopted: NextToken stays generic pending the ints
// pull regression (plan §5.1). It is kept here for the pull A/B measurement.

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

