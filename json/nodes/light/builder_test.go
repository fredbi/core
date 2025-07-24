package light

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	store "github.com/fredbi/core/json/stores/default-store"
	writer "github.com/fredbi/core/json/writers/default-writer"
)

func TestBuilder(t *testing.T) {
	s := store.New()
	b := NewBuilder(s)

	t.Run("should build a node from scratch", func(t *testing.T) {
		// this is an expression to build a node
		b.Object().AppendKey("test",
			NewBuilder(s).Array().AppendElems(
				NewBuilder(s).BoolValue(true).Node(),
				NewBuilder(s).StringValue("value").Node(),
				NewBuilder(s).NumericalValue(12.45).Node(),
			).Node(),
		).Node()
		require.True(t, b.Ok())

		t.Run("newly built node should decode to JSON", func(t *testing.T) {
			n := b.Node()

			w := new(bytes.Buffer)
			ctx := &ParentContext{
				W: writer.NewUnbuffered(w),
				S: s,
			}
			n.Encode(ctx)
			require.True(t, ctx.W.Ok())
			require.Nil(t, ctx.C)

			const original = `{"test":[true,"value",12.45]}`
			t.Run("produced JSON should match expectations", func(t *testing.T) {
				t.Logf("output: %s", w.String())
				assert.JSONEq(t, original, w.String())
			})

			t.Run("should mutate node", func(t *testing.T) {
				// this is an expression to clone a node and make some mutations
				c := NewBuilder(s).From(n).
					PrependKey("new_key",
						NewBuilder(s).Array().AppendElem(
							NewBuilder(s).BoolValue(true).Node(),
						).Node(),
					)
				require.True(t, c.Ok())

				nn := c.Node()

				t.Run("the cloned mutated node should decode to JSON", func(t *testing.T) {
					w.Reset()
					ctx.W.Reset()

					nn.Encode(ctx)
					require.True(t, ctx.W.Ok())
					require.Nil(t, ctx.C)

					t.Logf("output: %s", w.String())
					const mutated = `{"new_key":[true],"test":[true,"value",12.45]}`
					assert.JSONEq(t, mutated, w.String())
				})

				t.Run("the original node should not have changed", func(t *testing.T) {
					w.Reset()
					ctx.W.Reset()

					n.Encode(ctx)
					t.Logf("output: %s", w.String())
					assert.JSONEq(t, original, w.String())
				})
			})
		})
	})
}
