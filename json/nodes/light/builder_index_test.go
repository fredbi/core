package light

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/nodes"
	store "github.com/fredbi/core/json/stores/default-store"
	writer "github.com/fredbi/core/json/writers/default-writer"
)

// assertObjectIndexConsistent verifies that an object node's keysIndex is in perfect sync with its
// children slice: every child is reachable by key at its real position, and the index has no stale or
// dangling entries. This is the invariant that C1 (AppendKey off-by-one), C2 (RemoveKey re-index) and
// C3 (InsertKey dup-check) all broke.
func assertObjectIndexConsistent(t *testing.T, n Node) {
	t.Helper()

	require.Lenf(t, n.keysIndex, len(n.children),
		"keysIndex size (%d) must match children count (%d)", len(n.keysIndex), len(n.children))

	for i, child := range n.children {
		idx, found := n.keysIndex[child.key]
		require.Truef(t, found, "child %d key %q missing from keysIndex", i, child.key.String())
		assert.Equalf(t, i, idx, "keysIndex[%q] should be %d, got %d", child.key.String(), i, idx)

		got, ok := n.AtInternedKey(child.key)
		require.Truef(t, ok, "AtInternedKey(%q) should resolve", child.key.String())
		assert.Equal(t, child.key, got.key, "AtKey resolved to the wrong child")
	}
}

// objectKeys returns the ordered list of child keys of an object node.
func objectKeys(n Node) []string {
	keys := make([]string, 0, len(n.children))
	for _, c := range n.children {
		keys = append(keys, c.key.String())
	}

	return keys
}

func TestBuilderObjectIndex(t *testing.T) {
	s := store.New()

	scalar := func(v string) Node { return NewBuilder(s).StringValue(v).Node() }

	t.Run("AppendKey keeps index in sync and AtKey resolves (C1)", func(t *testing.T) {
		b := NewBuilder(s).Object().
			AppendKey("a", scalar("AAA")).
			AppendKey("b", scalar("BBB")).
			AppendKey("c", scalar("CCC"))
		require.True(t, b.Ok())

		n := b.Node()
		assertObjectIndexConsistent(t, n)
		assert.Equal(t, []string{"a", "b", "c"}, objectKeys(n))

		// AtKey used to panic with an out-of-range index before the off-by-one fix.
		for _, k := range []string{"a", "b", "c"} {
			got, ok := n.AtKey(k)
			require.Truef(t, ok, "AtKey(%q)", k)
			assert.Equal(t, k, got.Key())
		}

		idx, ok := n.KeyIndex("c")
		require.True(t, ok)
		assert.Equal(t, 2, idx)
	})

	t.Run("PrependKey keeps index in sync", func(t *testing.T) {
		b := NewBuilder(s).Object().
			AppendKey("a", scalar("AAA")).
			PrependKey("z", scalar("ZZZ")).
			PrependKey("y", scalar("YYY"))
		require.True(t, b.Ok())

		n := b.Node()
		assertObjectIndexConsistent(t, n)
		assert.Equal(t, []string{"y", "z", "a"}, objectKeys(n))
	})

	t.Run("InsertKey keeps index in sync at every position", func(t *testing.T) {
		b := NewBuilder(s).Object().
			AppendKey("a", scalar("AAA")).
			AppendKey("d", scalar("DDD")).
			InsertKey("b", 1, scalar("BBB")). // middle
			InsertKey("c", 2, scalar("CCC"))  // middle again
		require.True(t, b.Ok())

		n := b.Node()
		assertObjectIndexConsistent(t, n)
		assert.Equal(t, []string{"a", "b", "c", "d"}, objectKeys(n))

		got, ok := n.AtKey("c")
		require.True(t, ok)
		assert.Equal(t, "CCC", mustScalar(t, s, got))
	})

	t.Run("InsertKey position clamps to prepend/append", func(t *testing.T) {
		b := NewBuilder(s).Object().
			AppendKey("m", scalar("MMM")).
			InsertKey("first", -5, scalar("F")). // <= 0 -> prepend
			InsertKey("last", 999, scalar("L"))  // >= len -> append
		require.True(t, b.Ok())

		n := b.Node()
		assertObjectIndexConsistent(t, n)
		assert.Equal(t, []string{"first", "m", "last"}, objectKeys(n))
	})

	t.Run("RemoveKey re-indexes trailing keys (C2)", func(t *testing.T) {
		b := NewBuilder(s).Object().
			AppendKey("a", scalar("AAA")).
			AppendKey("b", scalar("BBB")).
			AppendKey("c", scalar("CCC")).
			AppendKey("d", scalar("DDD"))
		require.True(t, b.Ok())

		// remove a middle key: c and d must shift left and stay resolvable.
		b.RemoveKey("b")
		require.True(t, b.Ok())

		n := b.Node()
		assertObjectIndexConsistent(t, n)
		assert.Equal(t, []string{"a", "c", "d"}, objectKeys(n))

		got, ok := n.AtKey("d")
		require.True(t, ok)
		assert.Equal(t, "DDD", mustScalar(t, s, got))

		// removing the head key too.
		b.RemoveKey("a")
		require.True(t, b.Ok())
		n = b.Node()
		assertObjectIndexConsistent(t, n)
		assert.Equal(t, []string{"c", "d"}, objectKeys(n))
	})

	t.Run("RemoveKey of an absent key is a no-op", func(t *testing.T) {
		b := NewBuilder(s).Object().AppendKey("a", scalar("AAA"))
		b.RemoveKey("nope")
		require.True(t, b.Ok())

		n := b.Node()
		assertObjectIndexConsistent(t, n)
		assert.Equal(t, []string{"a"}, objectKeys(n))
	})

	t.Run("Swap exchanges children and index for objects", func(t *testing.T) {
		b := NewBuilder(s).Object().
			AppendKey("a", scalar("AAA")).
			AppendKey("b", scalar("BBB")).
			AppendKey("c", scalar("CCC"))
		require.True(t, b.Ok())

		b.Swap(0, 2)
		require.True(t, b.Ok())

		n := b.Node()
		assertObjectIndexConsistent(t, n)
		assert.Equal(t, []string{"c", "b", "a"}, objectKeys(n))
	})

	t.Run("duplicate keys are rejected on every insertion path (C3)", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			op   func(*Builder) *Builder
		}{
			{"AppendKey", func(b *Builder) *Builder { return b.AppendKey("a", scalar("dup")) }},
			{"PrependKey", func(b *Builder) *Builder { return b.PrependKey("a", scalar("dup")) }},
			{"InsertKey", func(b *Builder) *Builder { return b.InsertKey("a", 1, scalar("dup")) }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				b := NewBuilder(s).Object().
					AppendKey("a", scalar("AAA")).
					AppendKey("b", scalar("BBB"))
				require.True(t, b.Ok())

				tc.op(b)
				require.Falsef(t, b.Ok(), "%s should reject the duplicate key", tc.name)
				require.ErrorContains(t, b.Err(), "already present")
			})
		}
	})
}

// mustScalar encodes a single scalar node and returns the JSON-rendered string value (unquoted).
func mustScalar(t *testing.T, s *store.Store, n Node) string {
	t.Helper()

	v, ok := n.Value(s)
	require.True(t, ok)

	return string(v.StringValue().Value)
}

func TestNodeIsNullAndEncodeHandles(t *testing.T) {
	s := store.New()

	t.Run("IsNull is true only for a null node (C4)", func(t *testing.T) {
		nullN := NewBuilder(s).Null().Node()
		assert.True(t, nullN.IsNull(s), "a null node must report IsNull")

		strN := NewBuilder(s).StringValue("x").Node()
		assert.False(t, strN.IsNull(s), "a string scalar must not report IsNull")

		objN := NewBuilder(s).Object().Node()
		assert.False(t, objN.IsNull(s), "an object must not report IsNull")
	})

	t.Run("a null node encodes to JSON null", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := &ParentContext{W: writer.NewUnbuffered(&buf), S: s}
		NewBuilder(s).Null().Node().Encode(ctx)
		require.NoError(t, ctx.W.Err())
		assert.Equal(t, "null", buf.String())
	})

	t.Run("a scalar with a zero (absent) handle breaks the encoder (C4)", func(t *testing.T) {
		// hand-craft a corrupted scalar node: kind scalar but value == HandleZero (the zero handle).
		corrupt := Node{kind: nodes.KindScalar}
		require.True(t, corrupt.value.IsZero())

		var buf bytes.Buffer
		ctx := &ParentContext{W: writer.NewUnbuffered(&buf), S: s}
		corrupt.Encode(ctx)

		require.Error(t, ctx.W.Err(), "encoding a zero-handle scalar must raise an error")
		require.NotNil(t, ctx.C)
		assert.ErrorContains(t, ctx.C.Err, "zero")
	})
}
