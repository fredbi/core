package light

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/nodes"
	"github.com/fredbi/core/json/stores/values"
)

// decodeNode decodes a single JSON value and returns the node plus its store.
func decodeNode(t *testing.T, jazon string) (Node, *ParentContext) {
	t.Helper()

	ctx, n := newDecodeCtx(jazon, DecodeOptions{})
	n.Decode(ctx)
	require.NoError(t, ctx.L.Err())

	return *n, ctx
}

// TestAccessorIteratorsNeverPanic is the key regression: the public iterators returned nil on a
// kind-mismatch, so ranging them panicked. They must yield zero iterations instead.
func TestAccessorIteratorsNeverPanic(t *testing.T) {
	for name, jazon := range map[string]string{
		"scalar": `42`,
		"string": `"x"`,
		"bool":   `true`,
		"null":   `null`,
		"object": `{"a":1}`,
		"array":  `[1,2]`,
	} {
		t.Run(name, func(t *testing.T) {
			n, _ := decodeNode(t, jazon)

			assert.NotPanics(t, func() {
				count := 0
				for range n.Pairs() {
					count++
				}
				for range n.Elems() {
					count++
				}
				for range n.IndexedElems() {
					count++
				}
				_ = count
			})
		})
	}
}

// TestAccessorObject covers the object accessors on an object and on the wrong kind.
func TestAccessorObject(t *testing.T) {
	obj, ctx := decodeNode(t, `{"a":1,"b":2}`)

	t.Run("AtKey / KeyIndex hit and miss", func(t *testing.T) {
		got, ok := obj.AtKey("a")
		require.True(t, ok)
		_, isScalar := got.Value(ctx.S)
		require.True(t, isScalar)
		assert.Equal(t, "1", got.Dump(ctx.S))

		_, ok = obj.AtKey("zzz")
		assert.False(t, ok)

		idx, ok := obj.KeyIndex("b")
		assert.True(t, ok)
		assert.Equal(t, 1, idx)

		_, ok = obj.KeyIndex("zzz")
		assert.False(t, ok)
	})

	t.Run("Pairs yields members in order", func(t *testing.T) {
		var keys []string
		for k := range obj.Pairs() {
			keys = append(keys, k.String())
		}
		assert.Equal(t, []string{"a", "b"}, keys)
	})

	t.Run("Len and Is*", func(t *testing.T) {
		assert.Equal(t, 2, obj.Len())
		assert.True(t, obj.IsObject())
		assert.False(t, obj.IsArray())
	})

	t.Run("object accessors are safe on the wrong kind", func(t *testing.T) {
		scalar, _ := decodeNode(t, `42`)
		_, ok := scalar.AtKey("a")
		assert.False(t, ok)
		_, ok = scalar.KeyIndex("a")
		assert.False(t, ok)
		assert.Equal(t, 0, scalar.Len())
	})
}

// TestAccessorArray covers the array accessors on an array and on the wrong kind.
func TestAccessorArray(t *testing.T) {
	arr, ctx := decodeNode(t, `[10,20,30]`)

	t.Run("Elem in range and out of range", func(t *testing.T) {
		got, ok := arr.Elem(1)
		require.True(t, ok)
		assert.Equal(t, "20", got.Dump(ctx.S))

		_, ok = arr.Elem(3)
		assert.False(t, ok)
		_, ok = arr.Elem(-1)
		assert.False(t, ok)
	})

	t.Run("Elems / IndexedElems", func(t *testing.T) {
		var vals []string
		for v := range arr.Elems() {
			vals = append(vals, v.Dump(ctx.S))
		}
		assert.Equal(t, []string{"10", "20", "30"}, vals)

		last := -1
		for i := range arr.IndexedElems() {
			last = i
		}
		assert.Equal(t, 2, last)
		assert.Equal(t, 3, arr.Len())
	})

	t.Run("array accessors are safe on the wrong kind", func(t *testing.T) {
		obj, _ := decodeNode(t, `{"a":1}`)
		_, ok := obj.Elem(0)
		assert.False(t, ok)
	})
}

// TestAccessorScalarKindGuards covers Value/Handle/Is* across kinds, including the null-vs-scalar split.
func TestAccessorScalarKindGuards(t *testing.T) {
	t.Run("scalar", func(t *testing.T) {
		n, ctx := decodeNode(t, `"hello"`)
		assert.Equal(t, nodes.KindScalar, n.Kind())
		assert.True(t, n.IsString(ctx.S))
		assert.False(t, n.IsNumber(ctx.S))
		assert.False(t, n.IsBool(ctx.S))
		assert.False(t, n.IsNull())

		_, ok := n.Value(ctx.S)
		assert.True(t, ok)
		_, ok = n.Handle()
		assert.True(t, ok)
	})

	t.Run("null is a defined value, not a scalar subtype", func(t *testing.T) {
		n, ctx := decodeNode(t, `null`)
		assert.Equal(t, nodes.KindNull, n.Kind())
		assert.True(t, n.IsNull())
		assert.False(t, n.IsString(ctx.S))

		// null is a defined value: Value yields NullValue, Handle yields the non-zero null handle.
		v, ok := n.Value(ctx.S)
		assert.True(t, ok)
		assert.Equal(t, values.NullValue, v)
		h, ok := n.Handle()
		assert.True(t, ok)
		assert.False(t, h.IsZero())
	})

	t.Run("Value/Handle false on containers", func(t *testing.T) {
		obj, ctx := decodeNode(t, `{"a":1}`)
		_, ok := obj.Value(ctx.S)
		assert.False(t, ok)
		_, ok = obj.Handle()
		assert.False(t, ok)
	})
}
