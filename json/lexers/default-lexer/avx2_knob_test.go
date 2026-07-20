package lexer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/lexers/token"
)

// avx2Doc builds a document that exercises the AVX2 long-string delegation: long
// clean string values (well past guessLong), a long value with an escape followed
// by a long clean tail, plus short keys/values and numbers so both the inline fast
// path and the delegated path run.
func avx2Doc() []byte {
	long := strings.Repeat("abcdefghij", 30)          // 300-byte clean value
	tail := strings.Repeat("x", 200)                  // long clean tail after an escape
	return []byte(fmt.Sprintf(
		`{"k":"v","desc":%q,"n":123,"escaped":"pre\t%s","arr":["short",%q],"u":"café %s"}`,
		long, tail, long, tail,
	))
}

// tokenLike is satisfied by both token.T (semantic) and token.VT (verbatim, which
// embeds T), so the collector drains either lexer.
type tokenLike interface {
	Kind() token.Kind
	Value() []byte
	IsEOF() bool
}

// collectKV drains a token stream into (kind, value) pairs.
func collectKV[TK tokenLike](next func() TK, ok func() bool, err func() error) ([][2]string, error) {
	var out [][2]string
	for {
		tok := next()
		if !ok() {
			return out, err()
		}
		if tok.IsEOF() {
			return out, nil
		}
		out = append(out, [2]string{tok.Kind().String(), string(tok.Value())})
	}
}

// TestWithoutAVX2Equivalence asserts the WithoutAVX2 knob is purely a performance
// switch: the token stream (kinds + values) is identical with the AVX2 gate on and
// forced off, for both the semantic lexer L and the verbatim lexer VL.
func TestWithoutAVX2Equivalence(t *testing.T) {
	doc := avx2Doc()

	t.Run("semantic L", func(t *testing.T) {
		on := NewWithBytes(doc)
		off := NewWithBytes(doc, WithoutAVX2(true))
		gotOn, errOn := collectKV(on.NextToken, on.Ok, on.Err)
		gotOff, errOff := collectKV(off.NextToken, off.Ok, off.Err)
		require.NoError(t, errOn)
		require.NoError(t, errOff)
		require.Equal(t, gotOn, gotOff)
		require.NotEmpty(t, gotOn)
	})

	t.Run("verbatim VL", func(t *testing.T) {
		on := NewVerbatimWithBytes(doc)
		off := NewVerbatimWithBytes(doc, WithoutAVX2(true))
		gotOn, errOn := collectKV(on.NextToken, on.Ok, on.Err)
		gotOff, errOff := collectKV(off.NextToken, off.Ok, off.Err)
		require.NoError(t, errOn)
		require.NoError(t, errOff)
		require.Equal(t, gotOn, gotOff)
		require.NotEmpty(t, gotOn)
	})
}
