package json

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/nodes"
)

func TestDocument(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		const jazon = `[
		1,
		2,
		3,
		"a", 
		{
		  "b": {},
		  "c": [12.3,"x",{},true],
		  "d": null
		}
	]`

		t.Run("with no option", func(t *testing.T) {
			t.Run("should decode Document from JSON stream", func(t *testing.T) {
				doc := Make()
				r := bytes.NewBufferString(jazon)

				require.NoError(t, doc.Decode(r))

				t.Run("should encode Document to JSON stream", func(t *testing.T) {
					w := new(bytes.Buffer)
					require.NoError(t, doc.Encode(w))

					t.Logf("output: %s", w.String())
					assert.JSONEq(t, jazon, w.String())
				})

				t.Run("Document should MarshalJSON", func(t *testing.T) {
					data, err := doc.MarshalJSON()
					require.NoError(t, err)
					assert.JSONEq(t, jazon, string(data))

					t.Run("Document should UnmarshalJSON", func(t *testing.T) {
						clone := Make()
						require.NoError(t, clone.UnmarshalJSON(data))
						cloneData, err := doc.MarshalJSON()
						require.NoError(t, err)
						assert.JSONEq(t, jazon, string(cloneData))
					})
				})

				t.Run("Document should AppendText", func(t *testing.T) {
					data := make([]byte, 0, 100)
					const prefix = `prefix:`
					data = append(data, []byte(prefix)...)
					data, err := doc.AppendText(data)
					require.NoError(t, err)
					cut, found := bytes.CutPrefix(data, []byte(prefix))
					require.True(t, found)
					assert.JSONEq(t, jazon, string(cut))
				})

				t.Run("Document should be navigable with Elems", func(t *testing.T) {
					for e := range doc.Elems() {
						t.Logf("elem: %v", e.Kind())
						if e.Kind() == nodes.KindObject {
							for k, v := range e.Pairs() {
								t.Logf("key: %v; value: %v", k, v.Kind())
							}
						}
					}
				})
			})
		})
	})

	t.Run("document with error", func(t *testing.T) {
		const jazon = `[
		1,
		2,
		3,
		"a", 
		{
		  "b": {},
		  "c": [12.3,"x",{INVALID_TOKEN},true],
		  "d": null
		}
	]`

		doc := Make()
		r := bytes.NewBufferString(jazon)

		err := doc.Decode(r)
		require.Error(t, err)

		e, ok := err.(*DecodeError)
		require.True(t, ok)

		require.ErrorContains(t, e, `at path "/4/c/2" (offset: 63): invalid JSON token`)
		t.Logf("%v", e)
	})
}
