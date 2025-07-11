package light

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	lexer "github.com/fredbi/core/json/lexers/default-lexer"
	store "github.com/fredbi/core/json/stores/default-store"
	writer "github.com/fredbi/core/json/writers/default-writer"
)

func TestNode(t *testing.T) {
	t.Run("with happy path", func(t *testing.T) {
		const jazon = `{
			"test":[
				null,1,2,"a","x\n\t\r",
				{
					"z":true,
					"x":null,
					"a":[12,13,14],
					"b":[],
					"c":{}
				}
			]
		}`

		// Prerequisites to building and rendering a node:
		//
		//  - a JSON lexer (on top of a io.Reader)
		//  - a values store
		//  - a JSON writer (on top of an io.Writer)

		r := bytes.NewBufferString(jazon)
		w := new(bytes.Buffer)
		s := store.New()

		// All this is passed via the ParentContext, as it is reused in the entire hierarchy of nodes.
		ctx := &ParentContext{
			L: lexer.New(r),
			W: writer.New(w),
			S: s,
		}

		n := Node{}

		t.Run("should decode JSON stream", func(t *testing.T) {
			n.Decode(ctx)
			require.NoError(t, ctx.L.Err())

			t.Run("should resolve key", func(t *testing.T) {
				v, ok := n.AtKey("test")
				require.True(t, ok)

				t.Run("should resolve elements in array", func(t *testing.T) {
					i := 0
					for e := range v.Elems() {
						ev := e.value.Resolve(s)
						t.Logf("elem: %v", ev.Kind())
						i++
					}

					j := 0
					for k, e := range v.IndexedElems() {
						ev := e.value.Resolve(s)
						t.Logf("elem: %v", ev.Kind())
						j = k + 1
					}
					require.Equal(t, i, j)
				})
			})

			t.Run("should encode", func(t *testing.T) {
				n.Encode(ctx)
				require.NoError(t, ctx.W.Err())
				require.Positive(t, ctx.W.Size())
				t.Logf("written: %d", ctx.W.Size())

				t.Run("output should be equivalent to input", func(t *testing.T) {
					assert.JSONEq(t, jazon, w.String())
					t.Logf("output: %s", w.String())
				})
			})
		})
	})
}
