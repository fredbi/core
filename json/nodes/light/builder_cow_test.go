package light

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	store "github.com/fredbi/core/json/stores/default-store"
)

// snapshotObject renders an object node's keys (in order) plus its index, so a test can assert the
// original is byte-for-byte unaltered after a clone is mutated.
func snapshotObject(n Node) ([]string, map[string]int) {
	keys := make([]string, 0, len(n.children))
	for _, c := range n.children {
		keys = append(keys, c.key.String())
	}
	idx := make(map[string]int, len(n.keysIndex))
	for k, v := range n.keysIndex {
		idx[k.String()] = v
	}

	return keys, idx
}

func snapshotArray(t *testing.T, s *store.Store, n Node) []string {
	t.Helper()

	out := make([]string, 0, n.Len())
	for e := range n.Elems() {
		v, ok := e.Value(s)
		require.True(t, ok)
		out = append(out, string(v.StringValue().Value))
	}

	return out
}

// TestBuilderCopyOnWrite is the immutability guarantee: cloning a Node via From and mutating the clone
// must never alter the original, across every mutation path.
func TestBuilderCopyOnWrite(t *testing.T) {
	s := store.New()
	scalar := func(v string) Node { return NewBuilder(s).StringValue(v).Node() }

	makeObject := func() Node {
		return NewBuilder(s).Object().
			AppendKey("a", scalar("A")).
			AppendKey("b", scalar("B")).
			AppendKey("c", scalar("C")).Node()
	}
	makeArray := func() Node {
		return NewBuilder(s).Array().
			AppendElems(scalar("e0"), scalar("e1"), scalar("e2")).Node()
	}

	t.Run("object mutations leave the original unaltered", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			op   func(*Builder) *Builder
		}{
			{"AppendKey", func(b *Builder) *Builder { return b.AppendKey("z", scalar("Z")) }},
			{"PrependKey", func(b *Builder) *Builder { return b.PrependKey("z", scalar("Z")) }},
			{"InsertKey", func(b *Builder) *Builder { return b.InsertKey("z", 1, scalar("Z")) }},
			{"RemoveKey", func(b *Builder) *Builder { return b.RemoveKey("a") }},
			{"Swap", func(b *Builder) *Builder { return b.Swap(0, 2) }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				orig := makeObject()
				wantKeys, wantIdx := snapshotObject(orig)

				clone := tc.op(NewBuilder(s).From(orig))
				require.True(t, clone.Ok())
				_ = clone.Node()

				// the clone really changed (sanity), and the original did not.
				gotKeys, gotIdx := snapshotObject(orig)
				assert.Equalf(t, wantKeys, gotKeys, "%s altered original children", tc.name)
				assert.Equalf(t, wantIdx, gotIdx, "%s altered original index", tc.name)
				assertObjectIndexConsistent(t, orig)
			})
		}
	})

	t.Run("array mutations leave the original unaltered", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			op   func(*Builder) *Builder
		}{
			{"AppendElem", func(b *Builder) *Builder { return b.AppendElem(scalar("X")) }},
			{"PrependElem", func(b *Builder) *Builder { return b.PrependElem(scalar("X")) }},
			{"InsertElem", func(b *Builder) *Builder { return b.InsertElem(1, scalar("X")) }},
			{"RemoveElem", func(b *Builder) *Builder { return b.RemoveElem(0) }},
			{"Swap", func(b *Builder) *Builder { return b.Swap(0, 2) }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				orig := makeArray()
				want := snapshotArray(t, s, orig)

				clone := tc.op(NewBuilder(s).From(orig))
				require.True(t, clone.Ok())
				_ = clone.Node()

				assert.Equalf(t, want, snapshotArray(t, s, orig),
					"%s altered the original array", tc.name)
			})
		}
	})

	t.Run("the clone reflects the mutation (not a no-op)", func(t *testing.T) {
		orig := makeObject()
		clone := NewBuilder(s).From(orig).AppendKey("z", scalar("Z")).Node()

		_, ok := clone.AtKey("z")
		assert.True(t, ok, "clone should contain the new key")
		_, ok = orig.AtKey("z")
		assert.False(t, ok, "original must not contain the clone's new key")
		assert.Equal(t, 3, orig.Len())
		assert.Equal(t, 4, clone.Len())
	})

	t.Run("a long mutation chain still protects the original", func(t *testing.T) {
		orig := makeObject()
		wantKeys, wantIdx := snapshotObject(orig)

		clone := NewBuilder(s).From(orig).
			AppendKey("d", scalar("D")).
			PrependKey("z", scalar("Z")).
			InsertKey("m", 2, scalar("M")).
			RemoveKey("b").
			Swap(0, 1).
			Node()
		require.Equal(t, "object", clone.Kind().String())

		gotKeys, gotIdx := snapshotObject(orig)
		assert.Equal(t, wantKeys, gotKeys, "chain altered original children")
		assert.Equal(t, wantIdx, gotIdx, "chain altered original index")
		assertObjectIndexConsistent(t, clone)
	})

	t.Run("two snapshots from one builder diverge independently", func(t *testing.T) {
		b := NewBuilder(s).Object().AppendKey("a", scalar("A"))
		n1 := b.Node()
		b.AppendKey("b", scalar("B"))
		n2 := b.Node()

		assert.Equal(t, 1, n1.Len(), "first snapshot must not see later mutations")
		assert.Equal(t, 2, n2.Len())
		_, ok := n1.AtKey("b")
		assert.False(t, ok, "n1 must not contain the key added after it was extracted")
		assertObjectIndexConsistent(t, n1)
		assertObjectIndexConsistent(t, n2)
	})
}

// TestBuilderCopyOnWriteAllocs documents the copy-on-write allocation profile: cloning is free, a chain
// copies the shared slice/index at most once (not per mutation).
func TestBuilderCopyOnWriteAllocs(t *testing.T) {
	s := store.New()
	scalar := func(v string) Node { return NewBuilder(s).StringValue(v).Node() }

	obj := NewBuilder(s).Object().
		AppendKey("a", scalar("A")).
		AppendKey("b", scalar("B")).
		AppendKey("c", scalar("C")).Node()
	elem := scalar("X")

	// reuse a single builder so the NewBuilder allocation is excluded and we measure only the
	// structural copy-on-write cost.
	rb := NewBuilder(s)

	cloneOnly := int(testing.AllocsPerRun(100, func() {
		rb.Reset()
		rb.From(obj).Node()
	}))
	assert.Equalf(t, 0, cloneOnly, "cloning without mutation must not allocate, got %d", cloneOnly)

	oneMutation := int(testing.AllocsPerRun(100, func() {
		rb.Reset()
		rb.From(obj).AppendKey("x", elem).Node()
	}))

	fiveMutations := int(testing.AllocsPerRun(100, func() {
		rb.Reset()
		rb.From(obj).
			AppendKey("v", elem).
			AppendKey("w", elem).
			AppendKey("x", elem).
			AppendKey("y", elem).
			AppendKey("z", elem).Node()
	}))

	t.Logf("allocs — cloneOnly=%d oneMutation=%d fiveMutations=%d",
		cloneOnly, oneMutation, fiveMutations)

	// The slice+index are cloned once; the extra four appends only grow the owned map/slice. A 5-op
	// chain must therefore cost far less than 5x a single mutation (no per-step re-copy of the index).
	assert.Lessf(t, fiveMutations, 3*oneMutation,
		"a 5-op chain (%d) should not cost ~5x one mutation (%d): index copied once",
		fiveMutations, oneMutation)
}
