package json

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/nodes"
)

func TestCollection(t *testing.T) {
	jazonDocs := []string{
		`[1,2]`,
		`{"a": 1,"b":null}`,
		`{}`,
		`null`,
		`true`,
	}

	c := NewCollection()

	t.Run("Collection should DecodeAppend from readers", func(t *testing.T) {
		for _, input := range jazonDocs {
			r := bytes.NewBufferString(input)

			require.NoError(t, c.DecodeAppend(r))
		}

		require.Equal(t, len(jazonDocs), c.Len())
	})

	t.Run("individual docs with same Store should Append", func(t *testing.T) {
		doc1 := Make(WithStore(c.Store()))
		require.NoError(t, doc1.UnmarshalJSON([]byte("72")))

		doc2 := Make(WithStore(c.Store()))
		require.NoError(t, doc2.UnmarshalJSON([]byte("true")))

		c.Append(doc1, doc2)
		require.Equal(t, len(jazonDocs)+2, c.Len())
	})

	const expected = `[[1,2],{"a":1,"b":null},{},null,true,72,true]`
	t.Run("collection should MarshalJSON", func(t *testing.T) {
		data, err := c.MarshalJSON()
		require.NoError(t, err)

		t.Logf("output: %s", string(data))
		assert.JSONEq(t, expected, string(data))
	})

	t.Run("collection should Encode JSON", func(t *testing.T) {
		w := new(bytes.Buffer)
		require.NoError(t, c.Encode(w))

		t.Logf("output: %s", w.String())
		assert.JSONEq(t, expected, w.String())
	})

	t.Run("collection should AppendText", func(t *testing.T) {
		data := make([]byte, 0, 100)
		const prefix = `prefix:`
		data = append(data, []byte(prefix)...)
		data, err := c.AppendText(data)
		require.NoError(t, err)

		t.Logf("output: %s", string(data))
		cut, found := bytes.CutPrefix(data, []byte(prefix))
		require.True(t, found)
		assert.JSONEq(t, expected, string(cut))
	})

	t.Run("collection can be iterated over", func(t *testing.T) {
		i := 0
		for doc := range c.Documents() {
			t.Logf("kind[%d]: %v", i, doc.Kind())
			i++
		}
		require.Equal(t, c.Len(), i)

		d := c.Document(0)
		assert.Equal(t, nodes.KindArray, d.Kind())
	})
}
