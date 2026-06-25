package lab_test

// IndentLevel equivalence gate: the other equivalence tests cover the token
// stream (kind/value/blanks/position) but not IndentLevel. Folding VL's
// look-ahead out onto the unified core removed the lastStack mechanism VL used
// to report indent depth, so this gate locks in that (1) lab L still reports the
// reference L's IndentLevel on every fixture, and (2) lab VL's IndentLevel
// tracks lab L's (non-eliding) exactly — i.e. depth() alone is correct for both.

import (
	"testing"

	"github.com/stretchr/testify/require"

	lexer "github.com/fredbi/core/json/lexers/default-lexer"
	lab "github.com/fredbi/core/json/lexers/default-lexer/lab"
)

type indentLexer interface {
	Ok() bool
	IndentLevel() int
}

func semLevels(next func() (isEOF bool), l indentLexer) []int {
	var out []int
	for {
		eof := next()
		if !l.Ok() || eof {
			return out
		}
		out = append(out, l.IndentLevel())
	}
}

func TestIndentLevelEquivalence(t *testing.T) {
	fixtures := conformanceFixtures(t)
	for name, data := range fixtures {
		// L core unchanged: lab L must match reference L on every fixture.
		refL := lexer.NewWithBytes(data)
		labL := lab.NewWithBytes(data)
		wantL := semLevels(func() bool { return refL.NextToken().IsEOF() }, refL)
		gotL := semLevels(func() bool { return labL.NextToken().IsEOF() }, labL)
		require.Equalf(t, wantL, gotL, "L IndentLevel mismatch on %s", name)

		// VL now shares L's stack logic (no look-ahead, lastStack vestigial), so
		// lab VL's IndentLevel must track lab L's exactly on every fixture —
		// including i_/n_ where reference VL legitimately diverges.
		// VL never elides separators, so the L oracle must not either, else VL has
		// extra separator tokens (each with its own level) and the streams differ.
		labL2 := lab.NewWithBytes(data, lab.WithElideSeparator(false))
		labV := lab.NewVerbatimWithBytes(data)
		wantV := semLevels(func() bool { return labL2.NextToken().IsEOF() }, labL2)
		gotV := semLevels(func() bool { return labV.NextToken().IsEOF() }, labV)
		require.Equalf(t, wantV, gotV, "VL vs L IndentLevel mismatch on %s", name)
	}
}
