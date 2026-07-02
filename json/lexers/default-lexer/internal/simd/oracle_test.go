package simd

import (
	"math/rand"
	"testing"
)

// scalarStop is the reference: first index of a byte < 0x20 || == '"' || == '\\'.
func scalarStop(data []byte) int {
	for i, b := range data {
		if b < 0x20 || b == '"' || b == '\\' {
			return i
		}
	}
	return len(data)
}

func TestStringStopAVX2Oracle(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	lengths := []int{0, 1, 5, 7, 8, 15, 16, 31, 32, 33, 40, 63, 64, 65, 96, 127, 128, 200, 257}
	stops := []byte{'"', '\\', 0x00, 0x1f, 0x0a, 0x09, 0x1e}
	for _, n := range lengths {
		for trial := 0; trial < 400; trial++ {
			data := make([]byte, n)
			for i := range data {
				b := byte(0x20 + rng.Intn(0xE0)) // 0x20..0xff (incl high bytes)
				if b == '"' || b == '\\' {
					b = 'x'
				}
				data[i] = b
			}
			for k := 0; k < rng.Intn(4); k++ {
				if n == 0 {
					break
				}
				data[rng.Intn(n)] = stops[rng.Intn(len(stops))]
			}
			if got, want := stringStopIndexAVX2(data), scalarStop(data); got != want {
				t.Fatalf("n=%d trial=%d: AVX2=%d want=%d", n, trial, got, want)
			}
		}
	}
}

func TestStringStopAVX2HighBytesSafe(t *testing.T) {
	data := make([]byte, 64)
	for i := range data {
		data[i] = 0x80 | byte(i)
	}
	if got := stringStopIndexAVX2(data); got != 64 {
		t.Fatalf("high bytes flagged as stop: got %d, want 64", got)
	}
	data[50] = 0x1f
	if got := stringStopIndexAVX2(data); got != 50 {
		t.Fatalf("got %d, want 50", got)
	}
}
