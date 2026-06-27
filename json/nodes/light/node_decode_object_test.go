package light

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	lexcodes "github.com/fredbi/core/json/lexers/error-codes"
	nodecodes "github.com/fredbi/core/json/nodes/error-codes"
)

// TestDecodeDuplicateKey pins the duplicate-key contract: by default a duplicate key is a clean error
// pointing at the offending key; when tolerated, the last value wins and no sibling is lost.
//
// Regression: the default path used to `return false` without setting an error, which desynced the
// lexer — the object was silently truncated to the first member and a misleading "invalid JSON token"
// surfaced from the leftover tokens.
func TestDecodeDuplicateKey(t *testing.T) {
	t.Run("default mode errors with ErrDuplicateKey and the offending path", func(t *testing.T) {
		ctx, n := newDecodeCtx(`{"a":1,"a":2,"b":3}`, DecodeOptions{})
		n.Decode(ctx)

		require.Error(t, ctx.L.Err())
		assert.ErrorIs(t, ctx.L.Err(), nodecodes.ErrDuplicateKey)
		require.NotNil(t, ctx.C)
		assert.Equal(t, "/a", ctx.C.Path)
		// the misleading lexer error must NOT be what surfaces
		assert.NotErrorIs(t, ctx.L.Err(), lexcodes.ErrInvalidToken)
	})

	t.Run("nested duplicate reports the full path", func(t *testing.T) {
		ctx, n := newDecodeCtx(`{"x":{"a":1,"a":2}}`, DecodeOptions{})
		n.Decode(ctx)

		require.Error(t, ctx.L.Err())
		assert.ErrorIs(t, ctx.L.Err(), nodecodes.ErrDuplicateKey)
		require.NotNil(t, ctx.C)
		assert.Equal(t, "/x/a", ctx.C.Path)
	})

	t.Run("tolerate mode keeps the last value and all siblings", func(t *testing.T) {
		var do DecodeOptions
		do.tolerateDuplKey = true
		assert.Equal(t, `{"a":2,"b":3}`, decodeDump(t, `{"a":1,"a":2,"b":3}`, do))
	})

	t.Run("tolerate mode, duplicate is the last member", func(t *testing.T) {
		var do DecodeOptions
		do.tolerateDuplKey = true
		assert.Equal(t, `{"a":1,"b":9}`, decodeDump(t, `{"a":1,"b":2,"b":9}`, do))
	})
}

// TestDecodeEmptyContainers covers the empty-object/array paths through the same decode loops.
func TestDecodeEmptyContainers(t *testing.T) {
	for name, jazon := range map[string]string{
		"empty object":          `{}`,
		"empty array":           `[]`,
		"nested empties":        `{"a":{},"b":[]}`,
		"array of empties":      `[{},[],{}]`,
		"object with empty val": `{"a":{"b":{}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, jazon, decodeDump(t, jazon, DecodeOptions{}))
		})
	}
}
