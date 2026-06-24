package store

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/stores/values"
)

// dictTestPayloads returns long, compressible strings that share substrings with the preset
// dictionary below, so a Store with WithCompressionDict actually exercises the dictionary on both
// the compress and the decompress side.
func dictTestPayloads() []string {
	return []string{
		strings.Repeat("the quick brown fox ", 12),
		strings.Repeat("lazy dog jumps over ", 10),
		"the quick brown fox " + strings.Repeat("and the lazy dog ", 8),
		strings.Repeat("0123456789abcdef", 20),
	}
}

func dictTestDict() []byte {
	return []byte("the quick brown fox lazy dog jumps over and the 0123456789abcdef")
}

// newDictStore builds a Store whose compression always kicks in (low threshold) and is seeded with a
// caller-provided preset dictionary.
func newDictStore(dict []byte) *Store {
	return New(WithCompressionOptions(
		WithCompressionDict(dict),
		WithCompressionThreshold(16),
	))
}

// putAndCheck stores each payload, asserts it round-trips through Get and through the allocation-free
// AppendValueBytes path, and returns the handles for later re-verification.
func putAndCheck(t *testing.T, s stores.Store, payloads []string) []stores.Handle {
	t.Helper()

	handles := make([]stores.Handle, 0, len(payloads))
	var scratch []byte

	for _, p := range payloads {
		h := s.PutValue(values.MakeStringValue(p))
		handles = append(handles, h)

		got := s.Get(h)
		require.Equal(t, p, got.String(), "Get must round-trip a dict-compressed string")

		var v values.Value
		v, scratch = s.AppendValueBytes(scratch[:0], h)
		assert.Equal(t, p, string(v.Bytes()), "AppendValueBytes must round-trip a dict-compressed string")
	}

	return handles
}

// TestCompressionDictRoundTrip proves the core invariant: a value compressed against an injected
// preset dictionary decodes back identically against that same dictionary, on both the Get and the
// AppendValueBytes paths.
func TestCompressionDictRoundTrip(t *testing.T) {
	s := newDictStore(dictTestDict())

	payloads := dictTestPayloads()
	handles := putAndCheck(t, s, payloads)

	// re-read every handle once more: the arena holds compressed bytes bound to the frozen dict.
	for i, h := range handles {
		assert.Equal(t, payloads[i], s.Get(h).String())
	}
}

// TestCompressionDictSurvivesGob proves the dictionary travels with the Store across a gob round-trip
// and that the reloaded Store rebuilds its compression writer (cw): old payloads still decode, and
// the reloaded Store can compress a NEW payload that itself round-trips.
func TestCompressionDictSurvivesGob(t *testing.T) {
	dict := dictTestDict()
	src := newDictStore(dict)

	payloads := dictTestPayloads()
	handles := make([]stores.Handle, 0, len(payloads))
	for _, p := range payloads {
		handles = append(handles, src.PutValue(values.MakeStringValue(p)))
	}

	blob, err := src.MarshalBinary()
	require.NoError(t, err)

	var loaded Store
	require.NoError(t, loaded.UnmarshalBinary(blob))

	// the dictionary travelled with the arena: previously-compressed payloads still decode.
	for i, h := range handles {
		assert.Equal(t, payloads[i], loaded.Get(h).String(), "reloaded Store must decode old payloads")
	}

	// cw was rebuilt on load: the reloaded Store can compress a fresh payload and read it back.
	fresh := "the quick brown fox " + strings.Repeat("writes again ", 12)
	h := loaded.PutValue(values.MakeStringValue(fresh))
	assert.Equal(t, fresh, loaded.Get(h).String(), "reloaded Store must be able to compress and decode new payloads")
}

// TestCompressionDictSurvivesRecycle proves Store.Reset is a no-op on the compression configuration:
// a recycled Store keeps its injected dictionary and stays able to compress/decode against it. Under
// the previous Reset (which truncated dict to [:0] but kept the dict-encoding writer), the writer and
// the reader would have disagreed and the second round-trip would corrupt.
func TestCompressionDictSurvivesRecycle(t *testing.T) {
	dict := dictTestDict()
	s := newDictStore(dict)
	payloads := dictTestPayloads()

	putAndCheck(t, s, payloads)

	s.Reset()
	require.Empty(t, s.arena, "Reset clears the arena (data) ...")
	require.NotNil(t, s.dict, "... but preserves the compression configuration (frozen dictionary)")

	// the dictionary is still in force after recycling: a fresh round-trip must still succeed.
	putAndCheck(t, s, payloads)
}

// TestCompressionDictAliasesCaller documents that the Store aliases (does not copy) the injected
// dictionary slice: the contract is that the caller must treat it as immutable for the Store's life.
func TestCompressionDictAliasesCaller(t *testing.T) {
	dict := dictTestDict()
	s := newDictStore(dict)

	assert.Equal(t, &dict[0], &s.dict[0], "WithCompressionDict aliases the caller slice; it must not be mutated while the Store is alive")
}
