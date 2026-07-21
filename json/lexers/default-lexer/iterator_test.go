package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/lexers/token"
)

func TestTokensIterator(t *testing.T) {
	const doc = `{"a":[1,2.5e3,"x",true,null],"b":{"c":false}}`

	t.Run("yields the same tokens as the NextToken loop", func(t *testing.T) {
		// reference: manual loop
		var want []token.T
		ref := NewWithBytes([]byte(doc))
		for {
			tok := ref.NextToken()
			if !ref.Ok() || tok.IsEOF() {
				break
			}
			want = append(want, tok.Clone())
		}
		require.NoError(t, ref.Err())

		// iterator
		var got []token.T
		lex := NewWithBytes([]byte(doc))
		for tok := range lex.Tokens() {
			got = append(got, tok.Clone())
		}
		require.NoError(t, lex.Err())
		require.True(t, lex.Ok())

		require.Len(t, got, len(want))
		for i := range want {
			assert.Equalf(t, want[i].Kind(), got[i].Kind(), "token %d kind", i)
			assert.Equalf(t, want[i].Value(), got[i].Value(), "token %d value", i)
		}
	})

	t.Run("does not yield EOF", func(t *testing.T) {
		lex := NewWithBytes([]byte(`[1]`))
		for tok := range lex.Tokens() {
			require.False(t, tok.IsEOF(), "EOF must not be yielded")
		}
		require.True(t, lex.Ok())
	})

	t.Run("early break stops cleanly", func(t *testing.T) {
		lex := NewWithBytes([]byte(`[1,2,3,4,5]`))
		n := 0
		for range lex.Tokens() {
			n++
			if n == 2 {
				break
			}
		}
		assert.Equal(t, 2, n)
	})

	t.Run("stops on error, recorded in state", func(t *testing.T) {
		lex := NewWithBytes([]byte(`[1,,2]`)) // repeated comma
		n := 0
		for range lex.Tokens() {
			n++
		}
		assert.False(t, lex.Ok())
		require.Error(t, lex.Err())
	})
}

func TestVerbatimTokensIterator(t *testing.T) {
	const doc = ` { "a" : [ 1 , true ] } `

	var got []token.T
	var firstBlanks string
	vl := NewVerbatimWithBytes([]byte(doc))
	for tok := range vl.Tokens() {
		if len(got) == 0 {
			firstBlanks = string(vl.LeadingSpace()) // blanks are lexer state, read per token
		}
		got = append(got, tok.Clone())
	}
	require.NoError(t, vl.Err())
	require.NotEmpty(t, got)

	// the verbatim lexer preserves leading blanks (the doc starts with a space)
	assert.NotEmpty(t, firstBlanks, "first verbatim token should carry leading blanks")
}
