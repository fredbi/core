package writer

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	lexer "github.com/fredbi/core/json/lexers/default-lexer"
)

// TestVerbatimTokenRoundTrip proves that lexing with the verbatim lexer and
// writing each token back with VerbatimToken reproduces the source byte-for-byte,
// including insignificant whitespace and string escapes (the verbatim lexer keeps
// string values raw; the writer must NOT re-escape them).
func TestVerbatimTokenRoundTrip(t *testing.T) {
	sources := []string{
		`{"a": 1, "b": [true, null, 2.5e3]}`,
		"{\n  \"greet\": \"line1\\nline2\\ttab\",\n  \"q\": \"a\\\"b\",\n  \"u\": \"\\u0041\\ud83d\\ude00\"\n}",
		`[  "café",   "back\\slash",  "slash\/x"  ]`,
		"   [1,\r\n 2,\r\n 3]   ",
		`"éèê"`,
	}

	for _, src := range sources {
		var tw bytes.Buffer
		jw := NewUnbuffered(&tw)

		vl := lexer.NewVerbatimWithBytes([]byte(src))
		for tok := range vk.Tokens() {
			jw.VerbatimToken(vl.LeadingSpace(), tok)
			if tok.IsEOF() {
				break
			}
		}
		require.NoErrorf(t, vl.Err(), "lex error on %q", src)
		require.NoErrorf(t, jw.Err(), "write error on %q", src)

		assert.Equalf(t, src, tw.String(), "verbatim round-trip must be byte-exact for %q", src)
	}
}
