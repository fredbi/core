package lab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	lexer "github.com/fredbi/core/json/lexers/default-lexer"
)

// TestLabPositionEquivalence pins that the whole-buffer batched whitespace skip
// (skipWhitespaceWhole) keeps L.Line()/L.Column() identical to the reference, which
// tracks line/column per byte. The inputs deliberately include multi-space
// indentation and blank lines (the runs that trigger the batched skip and exercise
// its newline counting + lineStart update).
func TestLabPositionEquivalence(t *testing.T) {
	inputs := []string{
		"{\"a\": 12,\n \"b\": true}",                       // single newline + single space
		"{\n    \"a\": 1,\n    \"b\": 2\n}",                  // 4-space indentation (multi-ws runs)
		"[\n\n\n  1,\n\n  2\n]",                              // blank lines (consecutive newlines)
		"{\n\t\t\"x\": [\n\t\t\t1,\n\t\t\t2\n\t\t]\n}",      // tab indentation, nesting
		"   \n  \t  123   \n  ",                              // leading/trailing whitespace runs
		"[1,\r\n 2,\r\n 3]",                                 // CRLF line endings
	}

	type pos struct{ line, col int }
	collect := func(nt func() (kind int, line, col int, ok, eof bool)) []pos {
		var got []pos
		for {
			_, line, col, ok, eof := nt()
			if !ok || eof {
				return got
			}
			got = append(got, pos{line, col})
		}
	}

	for _, in := range inputs {
		ref := lexer.NewWithBytes([]byte(in), lexer.WithElideSeparator(false))
		want := collect(func() (int, int, int, bool, bool) {
			tok := ref.NextToken()
			return int(tok.Kind()), ref.Line(), ref.Column(), ref.Ok(), tok.IsEOF()
		})
		require.NoErrorf(t, ref.Err(), "reference error on %q", in)

		lab := NewWithBytes([]byte(in), WithElideSeparator(false))
		got := collect(func() (int, int, int, bool, bool) {
			tok := lab.NextToken()
			return int(tok.Kind()), lab.Line(), lab.Column(), lab.Ok(), tok.IsEOF()
		})
		require.NoErrorf(t, lab.Err(), "lab error on %q", in)

		assert.Equalf(t, want, got, "line/column mismatch on %q", in)
	}
}
