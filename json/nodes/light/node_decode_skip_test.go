package light

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/lexers"
	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/nodes"
	"github.com/fredbi/core/json/stores/values"
)

func decodeDump(t *testing.T, jazon string, do DecodeOptions) string {
	t.Helper()

	ctx, n := newDecodeCtx(jazon, do)
	n.Decode(ctx)
	require.NoError(t, ctx.L.Err())

	return n.Dump(ctx.S)
}

// TestDecodeSkip verifies the hook "skip" semantics: a skipped value is discarded from the result but
// its tokens are still consumed (drained for a composite), so the parse stays in sync. This is the
// CTX-4 fix — previously a Before*/NodeHook skip of a composite desynced the lexer.
func TestDecodeSkip(t *testing.T) {
	skipKey := func(name string) HookKeyFunc {
		return func(_ *ParentContext, _ lexers.Lexer, key values.InternedKey) (bool, error) {
			return key.String() == name, nil
		}
	}

	t.Run("BeforeKey", func(t *testing.T) {
		t.Run("skips a scalar member", func(t *testing.T) {
			var do DecodeOptions
			do.BeforeKey = skipKey("b")
			assert.Equal(t, `{"a":1,"c":3}`, decodeDump(t, `{"a":1,"b":2,"c":3}`, do))
		})

		t.Run("skips an object member (drains the subtree)", func(t *testing.T) {
			var do DecodeOptions
			do.BeforeKey = skipKey("b")
			assert.Equal(t, `{"a":1,"c":3}`, decodeDump(t, `{"a":1,"b":{"x":1,"y":[2,3]},"c":3}`, do))
		})

		t.Run("skips an array member (drains the subtree)", func(t *testing.T) {
			var do DecodeOptions
			do.BeforeKey = skipKey("a")
			assert.Equal(t, `{"b":4}`, decodeDump(t, `{"a":[1,2,3],"b":4}`, do))
		})

		t.Run("skips a deeply nested member", func(t *testing.T) {
			var do DecodeOptions
			do.BeforeKey = skipKey("a")
			assert.Equal(t, `{"e":5}`, decodeDump(t, `{"a":{"b":{"c":[1,2,{"d":3}]}},"e":5}`, do))
		})
	})

	t.Run("BeforeElem", func(t *testing.T) {
		t.Run("skips scalar elements (survivors renumber)", func(t *testing.T) {
			var do DecodeOptions
			do.BeforeElem = func(_ *ParentContext, _ lexers.Lexer, tok token.T) (bool, error) {
				return tok.IsNull(), nil
			}
			assert.Equal(t, `[1,2,3]`, decodeDump(t, `[1,null,2,null,3]`, do))
		})

		t.Run("skips composite elements (drains the subtree)", func(t *testing.T) {
			var do DecodeOptions
			do.BeforeElem = func(_ *ParentContext, _ lexers.Lexer, tok token.T) (bool, error) {
				return tok.IsStartObject() || tok.IsStartArray(), nil
			}
			assert.Equal(t, `[1,2,3]`, decodeDump(t, `[1,{"x":1},2,[9,9],3]`, do))
		})
	})

	t.Run("NodeHook", func(t *testing.T) {
		t.Run("skip drops the member consistently (not an empty placeholder)", func(t *testing.T) {
			var do DecodeOptions
			do.NodeHook = func(_ *ParentContext, _ lexers.Lexer, tok token.T) (bool, error) {
				return tok.IsStartArray(), nil
			}
			// root is an object (kept); the array value of "b" is skipped and drained.
			assert.Equal(t, `{"a":1,"c":4}`, decodeDump(t, `{"a":1,"b":[2,3],"c":4}`, do))
		})
	})

	t.Run("AfterKey", func(t *testing.T) {
		t.Run("skip drops an already-decoded member", func(t *testing.T) {
			var do DecodeOptions
			do.AfterKey = func(_ *ParentContext, _ lexers.Lexer, key values.InternedKey, _ Node) (bool, error) {
				return key.String() == "b", nil
			}
			assert.Equal(t, `{"a":1,"c":3}`, decodeDump(t, `{"a":1,"b":{"deep":1},"c":3}`, do))
		})
	})

	t.Run("AfterElem", func(t *testing.T) {
		t.Run("skip drops an already-decoded element", func(t *testing.T) {
			var do DecodeOptions
			do.AfterElem = func(_ *ParentContext, _ lexers.Lexer, elem Node) (bool, error) {
				return elem.Kind() == nodes.KindNull, nil
			}
			assert.Equal(t, `[1,2]`, decodeDump(t, `[1,null,2]`, do))
		})
	})
}

// TestDecodeSkipDoesNotMaskErrors checks that an error returned from a before-value hook still aborts
// decoding (skip and err are independent paths).
func TestDecodeSkipDoesNotMaskErrors(t *testing.T) {
	var do DecodeOptions
	do.BeforeElem = func(_ *ParentContext, _ lexers.Lexer, tok token.T) (bool, error) {
		if tok.IsStartObject() {
			return false, assert.AnError
		}

		return false, nil
	}

	ctx, n := newDecodeCtx(`[1,{"x":1},2]`, do)
	n.Decode(ctx)

	require.Error(t, ctx.L.Err())
	assert.ErrorIs(t, ctx.L.Err(), assert.AnError)
}
