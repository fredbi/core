package lab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/lexers/token"
)

func collectKinds(lex *L) ([]token.Kind, error) {
	var kinds []token.Kind
	for {
		tok := lex.NextToken()
		if !lex.Ok() {
			return kinds, lex.Err()
		}
		if tok.IsEOF() {
			return kinds, nil
		}
		kinds = append(kinds, tok.Kind())
	}
}

func TestElideSeparator(t *testing.T) {
	const doc = `{"a": "x", "b": 123, "c": [1, 2]}`

	t.Run("default elides comma and colon", func(t *testing.T) {
		kinds, err := collectKinds(NewWithBytes([]byte(doc)))
		require.NoError(t, err)

		// { Key Str Key Num Key [ Num Num ] }  -- no Delimiter for "," or ":"
		want := []token.Kind{
			token.Delimiter, // {
			token.Key, token.String,
			token.Key, token.Number,
			token.Key,
			token.Delimiter, // [
			token.Number, token.Number,
			token.Delimiter, // ]
			token.Delimiter, // }
		}
		require.Equal(t, want, kinds)
	})

	t.Run("no comma or colon delimiter is emitted", func(t *testing.T) {
		lex := NewWithBytes([]byte(doc))
		for {
			tok := lex.NextToken()
			if tok.IsEOF() {
				break
			}
			require.False(t, tok.IsComma(), "comma must be elided")
			require.False(t, tok.IsColon(), "colon must be elided")
		}
		require.NoError(t, lex.Err())
	})

	t.Run("WithElideSeparator(false) keeps separators", func(t *testing.T) {
		kinds, err := collectKinds(NewWithBytes([]byte(doc), WithElideSeparator(false)))
		require.NoError(t, err)

		commas, colons := 0, 0
		lex := NewWithBytes([]byte(doc), WithElideSeparator(false))
		for {
			tok := lex.NextToken()
			if tok.IsEOF() {
				break
			}
			if tok.IsComma() {
				commas++
			}
			if tok.IsColon() {
				colons++
			}
		}
		assert.Equal(t, 3, commas)
		assert.Equal(t, 3, colons)
		assert.Greater(t, len(kinds), 11) // more tokens than the elided stream
	})

	t.Run("grammar errors still fire under elision", func(t *testing.T) {
		_, err := collectKinds(NewWithBytes([]byte(`[1,,2]`)))
		require.ErrorIs(t, err, codes.ErrRepeatedComma)

		_, err = collectKinds(NewWithBytes([]byte(`[1,2,]`)))
		require.ErrorIs(t, err, codes.ErrTrailingComma)

		_, err = collectKinds(NewWithBytes([]byte(`{"a" 1}`)))
		require.ErrorIs(t, err, codes.ErrKeyColon)
	})

	t.Run("iterator honors elision", func(t *testing.T) {
		lex := NewWithBytes([]byte(doc))
		for tok := range lex.Tokens() {
			require.False(t, tok.IsComma())
			require.False(t, tok.IsColon())
		}
		require.NoError(t, lex.Err())
	})
}

func TestVerbatimNeverElides(t *testing.T) {
	const doc = `{"a": 1, "b": 2}`

	// even if the option is requested, the verbatim lexer keeps every token
	vl := NewVerbatimWithBytes([]byte(doc), WithElideSeparator(true))

	commas, colons := 0, 0
	for {
		tok := vl.NextToken()
		if tok.IsEOF() {
			break
		}
		if tok.IsComma() {
			commas++
		}
		if tok.IsColon() {
			colons++
		}
	}
	require.NoError(t, vl.Err())
	assert.Equal(t, 1, commas)
	assert.Equal(t, 2, colons)
}
