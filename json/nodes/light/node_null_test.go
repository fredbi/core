package light

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	store "github.com/fredbi/core/json/stores/default-store"
	"github.com/fredbi/core/json/stores/values"
)

// TestNullNodeIsAValue pins API-2: a JSON null is a defined value carrying a non-zero null handle, and
// a builder-built null is consistent with a decoded null (neither surfaces the zero handle). The
// builder path used to leave the zero handle (b.n = nullNode), which Value/Handle would have exposed.
func TestNullNodeIsAValue(t *testing.T) {
	t.Run("builder-built null", func(t *testing.T) {
		s := store.New()
		n := NewBuilder(s).Null().Node()

		require.True(t, n.IsNull())
		h, ok := n.Handle()
		require.True(t, ok)
		assert.False(t, h.IsZero(), "a builder null must carry a non-zero null handle")

		v, ok := n.Value(s)
		require.True(t, ok)
		assert.Equal(t, values.NullValue, v)
		assert.True(t, v.IsDefined())
	})

	t.Run("decoded null", func(t *testing.T) {
		n, ctx := decodeNode(t, `null`)

		require.True(t, n.IsNull())
		h, ok := n.Handle()
		require.True(t, ok)
		assert.False(t, h.IsZero())

		v, ok := n.Value(ctx.S)
		require.True(t, ok)
		assert.Equal(t, values.NullValue, v)
	})

	t.Run("not-found sentinel reports no value despite looking like null", func(t *testing.T) {
		obj, ctx := decodeNode(t, `{"a":1}`)
		missing, ok := obj.AtKey("zzz")
		require.False(t, ok)

		// the sentinel is KindNull by kind but carries the zero handle, so the zero-handle guard makes
		// Value/Handle report false — it is "no value", not a JSON null.
		assert.True(t, missing.IsNull()) // kind-only predicate still says null
		_, ok = missing.Value(ctx.S)
		assert.False(t, ok)
		_, ok = missing.Handle()
		assert.False(t, ok)
	})
}

// TestNodeKey pins API-4: Key returns (key, true) for an object member and ("", false) otherwise,
// distinguishing a missing key from a legitimately empty-string key.
func TestNodeKey(t *testing.T) {
	obj, _ := decodeNode(t, `{"a":1}`)

	t.Run("object member has a key", func(t *testing.T) {
		member, ok := obj.AtKey("a")
		require.True(t, ok)
		k, has := member.Key()
		assert.True(t, has)
		assert.Equal(t, "a", k)
	})

	t.Run("array element has no key", func(t *testing.T) {
		arr, _ := decodeNode(t, `[1]`)
		elem, ok := arr.Elem(0)
		require.True(t, ok)
		_, has := elem.Key()
		assert.False(t, has)
	})

	t.Run("root has no key", func(t *testing.T) {
		_, has := obj.Key()
		assert.False(t, has)
	})

	t.Run("empty-string key is present (distinct from absent)", func(t *testing.T) {
		o, _ := decodeNode(t, `{"":1}`)
		member, ok := o.AtKey("")
		require.True(t, ok)
		k, has := member.Key()
		assert.True(t, has)
		assert.Equal(t, "", k)
	})
}
