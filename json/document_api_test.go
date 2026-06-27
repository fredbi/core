package json

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/stores/values"
)

func mustDoc(t *testing.T, jazon string) Document {
	t.Helper()

	d := Make()
	require.NoError(t, d.Decode(bytes.NewBufferString(jazon)))

	return d
}

// TestDocumentKindPredicates covers the Is* family newly surfaced on Document (API-3).
func TestDocumentKindPredicates(t *testing.T) {
	obj := mustDoc(t, `{"a":1}`)
	assert.True(t, obj.IsObject())
	assert.False(t, obj.IsArray())

	assert.True(t, mustDoc(t, `[1,2]`).IsArray())
	assert.True(t, mustDoc(t, `"x"`).IsString())
	assert.True(t, mustDoc(t, `42`).IsNumber())
	assert.True(t, mustDoc(t, `true`).IsBool())

	null := mustDoc(t, `null`)
	assert.True(t, null.IsNull())
	assert.True(t, null.IsEmpty()) // IsEmpty and IsNull coincide for a null root
}

// TestDocumentValueAndHandleOnNull pins that the API-2 null-is-a-value contract carries through Document.
func TestDocumentValueAndHandleOnNull(t *testing.T) {
	null := mustDoc(t, `null`)

	v, ok := null.Value()
	require.True(t, ok)
	assert.Equal(t, values.NullValue, v)

	h, ok := null.Handle()
	require.True(t, ok)
	assert.False(t, h.IsZero())
}

// TestDocumentKeyAndAtInternedKey covers Key and the interned-key fast path on Document (API-3).
func TestDocumentKeyAndAtInternedKey(t *testing.T) {
	obj := mustDoc(t, `{"a":1,"b":2}`)

	a, ok := obj.AtKey("a")
	require.True(t, ok)
	k, has := a.Key()
	assert.True(t, has)
	assert.Equal(t, "a", k)

	b, ok := obj.AtInternedKey(values.MakeInternedKey("b"))
	require.True(t, ok)
	assert.True(t, b.IsNumber())

	_, ok = obj.AtInternedKey(values.MakeInternedKey("zzz"))
	assert.False(t, ok)

	// the root object itself has no key
	_, has = obj.Key()
	assert.False(t, has)
}

// TestDocumentIndexedElems covers IndexedElems newly surfaced on Document (API-3).
func TestDocumentIndexedElems(t *testing.T) {
	arr := mustDoc(t, `["x","y","z"]`)

	var idxs []int
	var vals []string
	for i, e := range arr.IndexedElems() {
		idxs = append(idxs, i)
		v, _ := e.Value()
		vals = append(vals, v.String())
	}

	assert.Equal(t, []int{0, 1, 2}, idxs)
	assert.Equal(t, []string{"x", "y", "z"}, vals)
}
