//go:build !race

package store

import (
	"bytes"
	"math"
	"math/rand/v2"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fredbi/core/json/lexers/token"
)

func TestStoreAllocations(t *testing.T) {
	// not to be asserted with -race: when running with the race detector, some contention on the pools
	// may prevent memory from being perfectly recycled, as in real life.
	// This may alter the measurement of the number of allocations, which is no longer fully accurated.

	const (
		epsilon = 1e-6
		runs    = 10
	)

	t.Run("with expected allocations on get", func(t *testing.T) {
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

		t.Run(
			"should not require any allocation for not so large string values",
			func(t *testing.T) {
				input := token.MakeWithValue(token.String, []byte("abcdefghij"))
				allocs := testing.AllocsPerRun(runs, func() {
					h := s.PutToken(input) // 1 alloc, amortized
					_ = s.Get(h)           // 1 alloc (required)
				})
				assert.InDelta(t, 0.00, allocs, epsilon)
			},
		)

		t.Run(
			"should require 1 allocation for compressed string values (heuristic works)",
			func(t *testing.T) {
				input := token.MakeWithValue(token.String, bytes.Repeat([]byte("a"), 129))
				allocs := testing.AllocsPerRun(10000*runs, func() {
					h := s.PutToken(input) // 1 alloc, amortized
					_ = s.Get(h)           // 1 alloc (required)
				})
				// Assert how the special case is handled here, because of the super-high compression ratio with this data sample.
				//
				assert.InDelta(t, 1.00, allocs, epsilon)
			},
		)

		t.Run(
			"should require 1 allocation for compressed string values (heuristic works)",
			func(t *testing.T) {
				alphabet := []byte("0123456789abcdefghijklmnoprstuvwxyz")
				testString := bytes.Repeat(alphabet, 10)
				rand.Shuffle(len(testString), func(i, j int) {
					testString[i], testString[j] = testString[j], testString[i]
				})
				input := token.MakeWithValue(token.String, testString)
				allocs := testing.AllocsPerRun(10000*runs, func() {
					h := s.PutToken(input) // 1 alloc, amortized
					_ = s.Get(h)           // 1 alloc (required)
				})
				// Assert how the special case is handled here, because of the super-high compression ratio with this data sample.
				//
				assert.InDelta(t, 1.00, allocs, epsilon)
			},
		)

		t.Run(
			"should require 2 allocations for compressed string values (heuristic works)",
			func(t *testing.T) {
				alphabet := []byte(strconv.FormatFloat(math.Pi, 'f', -1, 64))
				testString := bytes.Repeat(alphabet, 50)
				rand.Shuffle(len(testString), func(i, j int) {
					testString[i], testString[j] = testString[j], testString[i]
				})
				input := token.MakeWithValue(token.String, testString)
				allocs := testing.AllocsPerRun(10000*runs, func() {
					h := s.PutToken(input) // 1 alloc, amortized
					_ = s.Get(h)           // 1 alloc (required)
				})
				// Assert how the special case is handled here, because of the super-high compression ratio with this data sample.
				//
				assert.InDelta(t, 1.00, allocs, epsilon)
			},
		)
	})

	t.Run("with provided bytes factory", func(t *testing.T) {
		var buffer [256]byte
		s := New(
			WithBytesFactory(func() []byte {
				return buffer[:0]
			}),
		)
		s.PutToken(token.MakeBoolean(true))
		s.PutToken(token.MakeWithValue(token.String, []byte("abcd")))
		s.PutToken(token.MakeWithValue(token.String, []byte("abcdefgh")))
		s.PutToken(token.MakeWithValue(token.String, bytes.Repeat([]byte("a"), 129)))

		t.Run("should not require any allocation for small integer values", func(t *testing.T) {
			input := token.MakeWithValue(token.Number, []byte("1234"))
			allocs := testing.AllocsPerRun(runs, func() {
				h := s.PutToken(input)
				_ = s.Get(h)
			})
			assert.InDelta(t, 0.00, allocs, epsilon)
		})

		t.Run("should not require any allocation for small string values", func(t *testing.T) {
			input := token.MakeWithValue(token.String, []byte("abcd"))
			allocs := testing.AllocsPerRun(runs, func() {
				h := s.PutToken(input)
				_ = s.Get(h)
			})
			assert.InDelta(t, 0.00, allocs, epsilon)
		})

		t.Run("should not require allocation for compressed string values", func(t *testing.T) {
			input := token.MakeWithValue(token.String, bytes.Repeat([]byte("a"), 129))
			allocs := testing.AllocsPerRun(runs, func() {
				h := s.PutToken(input) // 1 alloc, amortized
				_ = s.Get(h)           // 1 alloc (required)
			})
			assert.InDelta(
				t,
				0.00,
				allocs,
				epsilon,
			) // this works because the resize heuristics remain within the provided buffer
		})
	})
}
