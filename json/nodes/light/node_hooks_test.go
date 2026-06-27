package light

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/lexers"
	"github.com/fredbi/core/json/nodes"
)

// TestHookContainerEnd exercises the capability the redesign adds: OnExit fires for every container
// including the document root, enabling whole-container validation (required keys, $ref detection) that
// the old hook set could not express (the root had no "after" event).
func TestHookContainerEnd(t *testing.T) {
	t.Run("root object required-keys validation", func(t *testing.T) {
		requireKey := func(name string) DecodeOptions {
			var do DecodeOptions
			do.OnExit = func(_ *ParentContext, _ lexers.Lexer, ev HookEvent) (Action, error) {
				if ev.Depth == 0 && ev.Kind == nodes.KindObject {
					if _, ok := ev.Node.AtKey(name); !ok {
						return Continue, fmt.Errorf("missing required key %q", name)
					}
				}

				return Continue, nil
			}

			return do
		}

		ctx, n := newDecodeCtx(`{"a":1,"b":2}`, requireKey("b"))
		n.Decode(ctx)
		require.NoError(t, ctx.L.Err())

		// missing required key at the root — impossible to catch with the old hooks.
		ctx, n = newDecodeCtx(`{"a":1}`, requireKey("b"))
		n.Decode(ctx)
		require.Error(t, ctx.L.Err())
	})

	t.Run("detect a $ref object anywhere in the tree", func(t *testing.T) {
		var refs int
		var do DecodeOptions
		do.OnExit = func(_ *ParentContext, _ lexers.Lexer, ev HookEvent) (Action, error) {
			if ev.Kind == nodes.KindObject {
				if _, ok := ev.Node.AtKey("$ref"); ok {
					refs++
				}
			}

			return Continue, nil
		}

		ctx, n := newDecodeCtx(`{"items":{"$ref":"#/defs/x"},"also":{"$ref":"#/defs/y"}}`, do)
		n.Decode(ctx)
		require.NoError(t, ctx.L.Err())
		assert.Equal(t, 2, refs)
	})

	t.Run("OnExit fires bottom-up: children before their container, root last", func(t *testing.T) {
		var order []string
		var do DecodeOptions
		do.OnExit = func(_ *ParentContext, _ lexers.Lexer, ev HookEvent) (Action, error) {
			switch {
			case ev.HasKey():
				order = append(order, ev.Key.String())
			case ev.Depth == 0:
				order = append(order, "<root>")
			}

			return Continue, nil
		}

		ctx, n := newDecodeCtx(`{"a":{"b":1},"c":2}`, do)
		n.Decode(ctx)
		require.NoError(t, ctx.L.Err())
		assert.Equal(t, []string{"b", "a", "c", "<root>"}, order)
	})

	t.Run("OnEnter sees the key and opening token before decoding", func(t *testing.T) {
		var seen []string
		var do DecodeOptions
		do.OnEnter = func(_ *ParentContext, _ lexers.Lexer, ev HookEvent) (Action, error) {
			if ev.HasKey() {
				seen = append(seen, fmt.Sprintf("%s:%s", ev.Key.String(), ev.Kind))
			}

			return Continue, nil
		}

		ctx, n := newDecodeCtx(`{"a":1,"b":{"c":true},"d":[1]}`, do)
		n.Decode(ctx)
		require.NoError(t, ctx.L.Err())
		assert.Equal(t, []string{"a:scalar", "b:object", "c:scalar", "d:array"}, seen)
	})
}
