//nolint:gosec
package store

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"testing"

	"github.com/fredbi/core/json/stores"
	"github.com/stretchr/testify/assert"
	"go.step.sm/crypto/randutil"
)

const (
	sampleShortStrings fixtureType = iota
	sampleShortNumbers
	sampleStrings
	sampleNumbers
	sampleBools
	maxTypesInSample // should come last
)

func TestConcurrentStore(t *testing.T) {
	const (
		parallel   = 10
		sampleSize = 10
		repeated   = 10
	)

	s := NewConcurrent()

	samples := generateValueSample(sampleSize)
	var handles testHandlesMap

	for n := range parallel {
		for range repeated {
			t.Run("put & get in random order", func(t *testing.T) {
				t.Parallel()

				t.Run(fmt.Sprintf("[%d] put value and keep handle", n), func(*testing.T) {
					i := rand.IntN(int(maxTypesInSample))
					j := rand.IntN(sampleSize)
					v := samples[i][j]
					h := s.PutValue(v)

					// keep the result for later verification
					handles.Put(h, v)
				})

				t.Run(fmt.Sprintf("[%d] get value from previously picked handle", n), func(*testing.T) {
					verify := handles.Get()
					v := s.Get(verify.h)

					// check the returned value against the original
					assert.Equal(t, verify.v, v)
				})
			})
		}
	}
}

type fixtureType uint8

type verifyHandle struct {
	h stores.Handle
	v stores.Value
}

type testHandlesMap struct {
	sync.Mutex
	handles []verifyHandle
}

func (m *testHandlesMap) Get() verifyHandle {
	m.Lock()
	defer m.Unlock()

	if len(m.handles) == 0 {
		return verifyHandle{stores.Handle(0), stores.NullValue}
	}

	i := rand.IntN(len(m.handles))

	return m.handles[i]
}

func (m *testHandlesMap) Put(h stores.Handle, v stores.Value) {
	m.Lock()
	defer m.Unlock()

	m.handles = append(m.handles, verifyHandle{h: h, v: v})
}

func generateValueSample(sampleSize int) [][]stores.Value {
	const (
		maxShortStringSize = 9
		maxStringSize      = 255
		maxShortNumberSize = uint64(1_000_000_000_000_000)
		floatMagnitude     = float64(1_000_000_000_000_000_000_000)
	)

	sample := make([][]stores.Value, maxTypesInSample)

	sample[sampleShortStrings] = randomStringValues(sampleSize, 1, maxShortStringSize)
	sample[sampleShortNumbers] = randomUintegerValues(sampleSize, maxShortNumberSize)
	sample[sampleStrings] = randomStringValues(sampleSize, maxShortStringSize, maxStringSize)
	sample[sampleNumbers] = randomNumberValues(sampleSize, floatMagnitude)
	sample[sampleBools] = randomBoolValues(sampleSize)

	return sample
}

func randomStringValues(n, minSize, maxSize int) []stores.Value {
	s := make([]stores.Value, 0, n)

	for range n {
		s = append(s, randomStringValue(minSize, maxSize))
	}

	return s
}

func randomStringValue(minSize, maxSize int) stores.Value {
	return stores.MakeStringValue(randomString(minSize, maxSize))
}

func randomString(minSize, maxSize int) string {
	size := rand.IntN(maxSize-minSize) + minSize

	s, err := randutil.Alphanumeric(size)
	if err != nil {
		panic(err)
	}

	return s
}

func randomNumberValues(n int, magnitude float64) []stores.Value {
	s := make([]stores.Value, 0, n)

	for range n {
		s = append(s, randomNumberValue(magnitude))
	}

	return s
}

func randomNumberValue(magnitude float64) stores.Value {
	return stores.MakeFloatValue(randomFloat(magnitude))
}

func randomFloat(magnitude float64) float64 {
	f := rand.Float64()

	return f * magnitude
}

func randomUintegerValues(n int, magnitude uint64) []stores.Value {
	s := make([]stores.Value, 0, n)

	for range n {
		s = append(s, randomUintegerValue(magnitude))
	}

	return s
}

func randomUintegerValue(magnitude uint64) stores.Value {
	return stores.MakeUintegerValue(randomUint(magnitude))
}

func randomUint(magnitude uint64) uint64 {
	return rand.Uint64N(magnitude)
}

func randomBoolValues(n int) []stores.Value {
	s := make([]stores.Value, 0, n)

	for range n {
		s = append(s, randomBoolValue())
	}

	return s
}

func randomBoolValue() stores.Value {
	return stores.MakeBoolValue(randomBool())
}

func randomBool() bool {
	const coinFlip = 0.5

	f := rand.Float64()

	return f < coinFlip
}
