package light

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/lexers"
	"github.com/fredbi/core/json/nodes"
)

func decodeDump(t *testing.T, jazon string, do DecodeOptions) string {
	t.Helper()

	ctx, n := newDecodeCtx(jazon, do)
	n.Decode(ctx)
	require.NoError(t, ctx.L.Err())

	return n.Dump(ctx.S)
}

// TestDecodeSkip verifies the Skip action: a skipped value is discarded from the result but its tokens
// are still consumed (drained for a composite), so the parse stays in sync. Skip on OnEnter avoids
// materializing the value; Skip on OnExit drops the already-built node.
func TestDecodeSkip(t *testing.T) {
	t.Run("OnEnter", func(t *testing.T) {
		t.Run("skips a scalar member", func(t *testing.T) {
			var do DecodeOptions
			do.OnEnter = func(_ *ParentContext, _ lexers.Lexer, ev HookEvent) (Action, error) {
				if ev.HasKey() && ev.Key.String() == "b" {
					return Skip, nil
				}

				return Continue, nil
			}
			assert.Equal(t, `{"a":1,"c":3}`, decodeDump(t, `{"a":1,"b":2,"c":3}`, do))
		})

		t.Run("skips an object member and drains its subtree", func(t *testing.T) {
			var do DecodeOptions
			do.OnEnter = func(_ *ParentContext, _ lexers.Lexer, ev HookEvent) (Action, error) {
				if ev.HasKey() && ev.Key.String() == "b" {
					return Skip, nil
				}

				return Continue, nil
			}
			assert.Equal(t, `{"a":1,"c":3}`, decodeDump(t, `{"a":1,"b":{"x":1,"y":[2,3]},"c":3}`, do))
		})

		t.Run("skips a deeply nested member", func(t *testing.T) {
			var do DecodeOptions
			do.OnEnter = func(_ *ParentContext, _ lexers.Lexer, ev HookEvent) (Action, error) {
				if ev.HasKey() && ev.Key.String() == "a" {
					return Skip, nil
				}

				return Continue, nil
			}
			assert.Equal(t, `{"e":5}`, decodeDump(t, `{"a":{"b":{"c":[1,2,{"d":3}]}},"e":5}`, do))
		})

		t.Run("skips composite array elements and drains them", func(t *testing.T) {
			var do DecodeOptions
			do.OnEnter = func(_ *ParentContext, _ lexers.Lexer, ev HookEvent) (Action, error) {
				if !ev.HasKey() && ev.Depth == 1 &&
					(ev.Token.IsStartObject() || ev.Token.IsStartArray()) {
					return Skip, nil
				}

				return Continue, nil
			}
			assert.Equal(t, `[1,2,3]`, decodeDump(t, `[1,{"x":1},2,[9,9],3]`, do))
		})
	})

	t.Run("OnExit", func(t *testing.T) {
		t.Run("skip drops an already-decoded member", func(t *testing.T) {
			var do DecodeOptions
			do.OnExit = func(_ *ParentContext, _ lexers.Lexer, ev HookEvent) (Action, error) {
				if ev.HasKey() && ev.Key.String() == "b" {
					return Skip, nil
				}

				return Continue, nil
			}
			assert.Equal(t, `{"a":1,"c":3}`, decodeDump(t, `{"a":1,"b":{"deep":1},"c":3}`, do))
		})

		t.Run("skip drops null array elements", func(t *testing.T) {
			var do DecodeOptions
			do.OnExit = func(_ *ParentContext, _ lexers.Lexer, ev HookEvent) (Action, error) {
				if !ev.HasKey() && ev.Depth == 1 && ev.Node.Kind() == nodes.KindNull {
					return Skip, nil
				}

				return Continue, nil
			}
			assert.Equal(t, `[1,2]`, decodeDump(t, `[1,null,2]`, do))
		})
	})
}

// TestDecodeStop verifies the Stop action: decoding halts, keeping what was built so far, with no error.
func TestDecodeStop(t *testing.T) {
	t.Run("Stop on OnExit keeps the value that triggered it", func(t *testing.T) {
		var do DecodeOptions
		do.OnExit = func(_ *ParentContext, _ lexers.Lexer, ev HookEvent) (Action, error) {
			if ev.HasKey() && ev.Key.String() == "b" {
				return Stop, nil
			}

			return Continue, nil
		}
		// "c" is never decoded; "a" and "b" are kept.
		assert.Equal(t, `{"a":1,"b":2}`, decodeDump(t, `{"a":1,"b":2,"c":3}`, do))
	})

	t.Run("Stop on OnEnter drops the value and halts", func(t *testing.T) {
		var do DecodeOptions
		do.OnEnter = func(_ *ParentContext, _ lexers.Lexer, ev HookEvent) (Action, error) {
			if ev.HasKey() && ev.Key.String() == "b" {
				return Stop, nil
			}

			return Continue, nil
		}
		// "b" is not decoded, "c" never reached; only "a" survives.
		assert.Equal(t, `{"a":1}`, decodeDump(t, `{"a":1,"b":2,"c":3}`, do))
	})
}

// TestDecodeHookError checks that a hook error aborts decoding (independent of the Action).
func TestDecodeHookError(t *testing.T) {
	var do DecodeOptions
	do.OnEnter = func(_ *ParentContext, _ lexers.Lexer, ev HookEvent) (Action, error) {
		if ev.Token.IsStartObject() && ev.Depth > 0 {
			return Continue, assert.AnError
		}

		return Continue, nil
	}

	ctx, n := newDecodeCtx(`[1,{"x":1},2]`, do)
	n.Decode(ctx)

	require.Error(t, ctx.L.Err())
	assert.ErrorIs(t, ctx.L.Err(), assert.AnError)
}
