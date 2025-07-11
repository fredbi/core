package json

import (
	"bytes"
	"testing"

	store "github.com/fredbi/core/json/stores/default-store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	s := store.New()
	b := NewBuilder(s)

	t.Run("should build a Document from scratch", func(t *testing.T) {
		b.Object().AppendKey("test",
			NewBuilder(s).Array().AppendElems(
				b.MakeNull(),
				b.MakeBool(true),
				b.MakeString("abc"),
				b.MakeNumber(123.45),
			).Document(),
		)

		require.True(t, b.Ok())

		doc := b.Document()

		w := new(bytes.Buffer)
		require.NoError(t, doc.Encode(w))

		t.Logf("output: %s", w.String())
		assert.Equal(t, `{"test":[null,true,"abc",123.45]}`, w.String())
	})
}
