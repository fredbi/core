package light

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/nodes"
	nodecodes "github.com/fredbi/core/json/nodes/error-codes"
	store "github.com/fredbi/core/json/stores/default-store"
	"github.com/fredbi/core/json/types"
)

// TestBuilderNumberConstructors covers the typed numeric builders (previously ~0% covered).
func TestBuilderNumberConstructors(t *testing.T) {
	s := store.New()

	for name, tc := range map[string]struct {
		build func(*Builder) *Builder
		want  string
	}{
		"Float64Value":  {func(b *Builder) *Builder { return b.Float64Value(1.5) }, "1.5"},
		"Float32Value":  {func(b *Builder) *Builder { return b.Float32Value(2.5) }, "2.5"},
		"IntegerValue":  {func(b *Builder) *Builder { return b.IntegerValue(-42) }, "-42"},
		"UintegerValue": {func(b *Builder) *Builder { return b.UintegerValue(42) }, "42"},
		"NumberValue":   {func(b *Builder) *Builder { return b.NumberValue(types.Number{Value: []byte("3.14")}) }, "3.14"},
	} {
		t.Run(name, func(t *testing.T) {
			n := tc.build(NewBuilder(s)).Node()
			assert.Equal(t, nodes.KindScalar, n.Kind())
			assert.True(t, n.IsNumber(s))
			assert.Equal(t, tc.want, n.Dump(s))
		})
	}
}

// TestBuilderNumericalValue covers the any-dispatching NumericalValue across every supported type
// (previously 11% covered — the big type switch was almost entirely untested).
func TestBuilderNumericalValue(t *testing.T) {
	s := store.New()
	num := func(v any) Node { return NewBuilder(s).NumericalValue(v).Node() }

	for name, tc := range map[string]struct {
		in   any
		want string
	}{
		"int":        {int(7), "7"},
		"int64":      {int64(-7), "-7"},
		"int32":      {int32(-8), "-8"},
		"int16":      {int16(9), "9"},
		"int8":       {int8(-10), "-10"},
		"uint":       {uint(8), "8"},
		"uint64":     {uint64(11), "11"},
		"uint32":     {uint32(12), "12"},
		"uint16":     {uint16(13), "13"},
		"uint8":      {uint8(14), "14"},
		"float64":    {float64(0.5), "0.5"},
		"float32":    {float32(0.25), "0.25"},
		"*big.Int":   {big.NewInt(123456789), "123456789"},
		"big.Int":    {*big.NewInt(42), "42"},
		"*big.Float": {big.NewFloat(2.75), "2.75"},
		"big.Float":  {*big.NewFloat(3.5), "3.5"},
		"*big.Rat":   {big.NewRat(1, 4), "0.25"},
		"big.Rat":    {*big.NewRat(1, 2), "0.5"},
		"string":     {"9.99", "9.99"},
		"bytes":      {[]byte("123"), "123"},
	} {
		t.Run(name, func(t *testing.T) {
			n := num(tc.in)
			assert.True(t, n.IsNumber(s), "expected a number node")
			assert.Equal(t, tc.want, n.Dump(s))
		})
	}
}

// TestBuilderNumericalValueErrorsAndNoops covers the error and no-op branches of NumericalValue.
func TestBuilderNumericalValueErrorsAndNoops(t *testing.T) {
	s := store.New()

	t.Run("errors", func(t *testing.T) {
		for name, in := range map[string]any{
			"unsupported type": struct{}{},
			"bad string":       "not-a-number",
			"bad bytes":        []byte("xyz"),
		} {
			t.Run(name, func(t *testing.T) {
				b := NewBuilder(s).NumericalValue(in)
				require.Error(t, b.Err())
				assert.ErrorIs(t, b.Err(), nodecodes.ErrBuilder)
			})
		}
	})

	t.Run("no-ops leave a null node without error", func(t *testing.T) {
		var nilInt *big.Int
		var nilFloat *big.Float
		var nilRat *big.Rat
		for name, in := range map[string]any{
			"nil *big.Int":   nilInt,
			"nil *big.Float": nilFloat,
			"nil *big.Rat":   nilRat,
			"empty string":   "",
			"empty bytes":    []byte{},
		} {
			t.Run(name, func(t *testing.T) {
				b := NewBuilder(s).NumericalValue(in)
				require.NoError(t, b.Err())
				assert.Equal(t, nodes.KindNull, b.Node().Kind())
			})
		}
	})
}

// TestBuilderMiscPublicMethods covers public builder/accessor methods that were at 0%.
func TestBuilderMiscPublicMethods(t *testing.T) {
	s := store.New()

	t.Run("WithStore sets the store after construction", func(t *testing.T) {
		n := NewBuilder(nil).WithStore(s).IntegerValue(5).Node()
		assert.Equal(t, "5", n.Dump(s))
	})

	t.Run("WithContext sets the node context, readable via Context().Offset()", func(t *testing.T) {
		n := NewBuilder(s).IntegerValue(1).WithContext(Context{offset: 99}).Node()
		assert.Equal(t, uint64(99), n.Context().Offset())
	})

	t.Run("SetErr marks the builder not-ok", func(t *testing.T) {
		b := NewBuilder(s)
		b.SetErr(assert.AnError)
		assert.False(t, b.Ok())
		assert.ErrorIs(t, b.Err(), assert.AnError)
	})

	t.Run("ClearChildren empties a container", func(t *testing.T) {
		b := NewBuilder(s).Object().AppendKey("a", NewBuilder(s).IntegerValue(1).Node())
		b.ClearChildren()
		assert.Equal(t, 0, b.Node().Len())
	})

	t.Run("ClearChildren on a non-container errors", func(t *testing.T) {
		b := NewBuilder(s).IntegerValue(1)
		b.ClearChildren()
		require.Error(t, b.Err())
		assert.ErrorIs(t, b.Err(), nodecodes.ErrBuilder)
	})

	t.Run("IsBool true on a bool node", func(t *testing.T) {
		n := NewBuilder(s).BoolValue(true).Node()
		assert.True(t, n.IsBool(s))
		assert.False(t, n.IsNumber(s))
	})

	t.Run("Is* are false on a non-scalar node", func(t *testing.T) {
		obj := NewBuilder(s).Object().Node()
		assert.False(t, obj.IsNumber(s))
		assert.False(t, obj.IsBool(s))
		assert.False(t, obj.IsString(s))
	})
}

// TestBuilderShortCircuitsOnError verifies that once a builder holds an error every subsequent op is a
// no-op that preserves the error and never panics — covering the !b.Ok() guard of each method.
func TestBuilderShortCircuitsOnError(t *testing.T) {
	s := store.New()
	b := NewBuilder(s)
	b.SetErr(assert.AnError)

	assert.NotPanics(t, func() {
		b.Object().Array().Null().
			StringValue("x").BytesValue([]byte("y")).BoolValue(true).
			NumberValue(types.Number{Value: []byte("1")}).NumericalValue(5).
			Float64Value(1).Float32Value(2).IntegerValue(3).UintegerValue(4).
			AppendKey("k", nullNode).AppendElem(nullNode).AppendElems(nullNode, nullNode).
			PrependKey("p", nullNode).PrependElem(nullNode).
			InsertKey("i", 0, nullNode).InsertElem(0, nullNode).
			RemoveKey("k").RemoveElem(0).Swap(0, 1).ClearChildren()
	})

	require.ErrorIs(t, b.Err(), assert.AnError)
	assert.Equal(t, nodes.KindNull, b.Node().Kind())
}
