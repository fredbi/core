package lab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	lexer "github.com/fredbi/core/json/lexers/default-lexer"
)

// TestLabVerbatimPositionEquivalence pins that the VERBATIM lexer's line/column
// (baked into token.VT) stays identical to the reference verbatim lexer. Position
// accounting now lives only in the verbatim path (the semantic lexer dropped it —
// see emitPolicy.tracksPosition); these inputs include multi-space indentation,
// blank lines, tabs and CRLF so the per-newline bookkeeping is exercised.
func TestLabVerbatimPositionEquivalence(t *testing.T) {
	inputs := []string{
		"{\"a\": 12,\n \"b\": true}",
		"{\n    \"a\": 1,\n    \"b\": 2\n}",
		"[\n\n\n  1,\n\n  2\n]",
		"{\n\t\t\"x\": [\n\t\t\t1,\n\t\t\t2\n\t\t]\n}",
		"   \n  \t  123   \n  ",
		"[1,\r\n 2,\r\n 3]",
	}

	type pos struct{ line, col int }

	for _, in := range inputs {
		var want []pos
		rv := lexer.NewVerbatimWithBytes([]byte(in))
		for {
			tok := rv.NextToken()
			if !rv.Ok() || tok.IsEOF() {
				break
			}
			want = append(want, pos{tok.Line(), tok.Col()})
		}
		require.NoErrorf(t, rv.Err(), "reference verbatim error on %q", in)

		var got []pos
		lv := NewVerbatimWithBytes([]byte(in))
		for {
			tok := lv.NextToken()
			if !lv.Ok() || tok.IsEOF() {
				break
			}
			// the lexer-level VL.Line()/Column() must agree with the token's position
			assert.Equalf(t, tok.Line(), lv.Line(), "VL.Line() vs token on %q", in)
			assert.Equalf(t, tok.Col(), lv.Column(), "VL.Column() vs token on %q", in)
			got = append(got, pos{tok.Line(), tok.Col()})
		}
		require.NoErrorf(t, lv.Err(), "lab verbatim error on %q", in)

		assert.Equalf(t, want, got, "verbatim line/column mismatch on %q", in)
	}
}

// Note: the semantic lexer L intentionally has NO Line()/Column() methods — the
// no-position-accounting contract is enforced by their absence (the package would
// not compile if anything called them). Position is verbatim-only; see [VL.Line].
