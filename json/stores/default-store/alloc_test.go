//go:build !race

package store

import (
	"bytes"
	"testing"

	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/stores"
	"github.com/stretchr/testify/assert"
)

func TestStoreAllocations(t *testing.T) {
	// not to be asserted with -race: when running with the race detector, some contention on the pools
	// may prevent memory from being perfectly recycled, as in real life.
	// This may alter the measurement of the number of allocations, which is no longer fully accurated.

	const (
		epsilon = 1e-6
		runs    = 10
	)
	s := New()

	t.Run("should not require any allocation on bool values", func(t *testing.T) {
		input := token.MakeBoolean(true)
		allocs := testing.AllocsPerRun(10, func() {
			h := s.PutToken(input)
			_ = s.Get(h)
		})
		assert.InDelta(t, 0.00, allocs, epsilon)
	})

	t.Run("should require 1 allocation for small integer values", func(t *testing.T) {
		input := token.MakeWithValue(token.Number, []byte("1234"))
		allocs := testing.AllocsPerRun(runs, func() {
			h := s.PutToken(input) // 1 alloc, amortized
			_ = s.Get(h)           // 1 alloc (required)
		})
		assert.InDelta(t, 1.00, allocs, epsilon)
	})

	t.Run("should require 1 allocation for small string values", func(t *testing.T) {
		input := token.MakeWithValue(token.String, []byte("abcd"))
		allocs := testing.AllocsPerRun(runs, func() {
			h := s.PutToken(input) // 1 alloc, amortized
			_ = s.Get(h)           // 1 alloc (required)
		})
		assert.InDelta(t, 1.00, allocs, epsilon)
	})

	t.Run("should require 1 allocation for small ASCII string values", func(t *testing.T) {
		input := token.MakeWithValue(token.String, []byte("abcdefgh"))
		allocs := testing.AllocsPerRun(runs, func() {
			h := s.PutToken(input) // 1 alloc, amortized
			_ = s.Get(h)           // 1 alloc (required)
		})
		assert.InDelta(t, 1.00, allocs, epsilon)
	})

	t.Run("should not require any allocation for not so large string values", func(t *testing.T) {
		input := token.MakeWithValue(token.String, []byte("abcdefghij"))
		allocs := testing.AllocsPerRun(runs, func() {
			h := s.PutToken(input) // 1 alloc, amortized
			_ = s.Get(h)           // 1 alloc (required)
		})
		assert.InDelta(t, 0.00, allocs, epsilon)
	})

	t.Run("should require 1 allocation for compressed string values", func(t *testing.T) {
		input := token.MakeWithValue(token.String, bytes.Repeat([]byte("a"), 129))
		allocs := testing.AllocsPerRun(10000*runs, func() {
			h := s.PutToken(input) // 1 alloc, amortized
			_ = s.Get(h)           // 1 alloc (required)
		})
		assert.InDelta(t, 1.00, allocs, epsilon)
	})

	t.Run("should not require any allocation for not so large string values", func(t *testing.T) {
		input := token.MakeWithValue(token.String, []byte("abcdefghij"))
		allocs := testing.AllocsPerRun(runs, func() {
			h := s.PutToken(input) // 1 alloc, amortized
			_ = s.Get(h)           // 1 alloc (required)
		})
		assert.InDelta(t, 0.00, allocs, epsilon)
	})

	t.Run("with provided buffer", func(t *testing.T) {
		t.Run("should not require any allocation for small integer values", func(t *testing.T) {
			var buffer [4]byte
			input := token.MakeWithValue(token.Number, []byte("1234"))
			allocs := testing.AllocsPerRun(runs, func() {
				h := s.PutToken(input)
				_ = s.Get(h, stores.WithBuffer(buffer[:0]))
			})
			assert.InDelta(t, 0.00, allocs, epsilon)
		})

		t.Run("should not require any allocation for small string values", func(t *testing.T) {
			var buffer [4]byte
			input := token.MakeWithValue(token.String, []byte("abcd"))
			allocs := testing.AllocsPerRun(runs, func() {
				h := s.PutToken(input)
				_ = s.Get(h, stores.WithBuffer(buffer[:0]))
			})
			assert.InDelta(t, 0.00, allocs, epsilon)
		})

		t.Run("should not require allocation for compressed string values", func(t *testing.T) {
			var buffer [129]byte
			input := token.MakeWithValue(token.String, bytes.Repeat([]byte("a"), 129))
			allocs := testing.AllocsPerRun(runs, func() {
				h := s.PutToken(input)                      // 1 alloc, amortized
				_ = s.Get(h, stores.WithBuffer(buffer[:0])) // 1 alloc (required)
			})
			assert.InDelta(t, 0.00, allocs, epsilon)
		})
	})
}
