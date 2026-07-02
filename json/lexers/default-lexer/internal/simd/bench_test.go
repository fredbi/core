package simd

import (
	"encoding/binary"
	"fmt"
	"math/bits"
	"testing"
)

const lo = 0x0101010101010101
const hi = lo * 0x80

// stringStopMask / stringStopIndexSWAR replicate the lexer's 8-byte SWAR scan, for
// the head-to-head against the AVX2 kernel.
func stringStopMask(w uint64) uint64 {
	m := (w - lo*0x20) &^ w & hi
	q := w ^ (lo * 0x22)
	m |= (q - lo) &^ q & hi
	s := w ^ (lo * 0x5c)
	m |= (s - lo) &^ s & hi
	return m
}

func stringStopIndexSWAR(data []byte) int {
	n, i := len(data), 0
	for i+8 <= n {
		if m := stringStopMask(binary.LittleEndian.Uint64(data[i:])); m != 0 {
			return i + bits.TrailingZeros64(m)>>3
		}
		i += 8
	}
	for ; i < n; i++ {
		if b := data[i]; b < 0x20 || b == '"' || b == '\\' {
			return i
		}
	}
	return n
}

var sink int

func BenchmarkStringStop(b *testing.B) {
	for _, n := range []int{8, 16, 32, 64, 128, 256, 512, 1024, 4096} {
		data := make([]byte, n)
		for i := range data {
			data[i] = 'x'
		}
		data[n-1] = '"' // stop only at the end: whole-buffer scan
		b.Run(fmt.Sprintf("len=%04d/SWAR", n), func(b *testing.B) {
			b.SetBytes(int64(n))
			for b.Loop() {
				sink += stringStopIndexSWAR(data)
			}
		})
		b.Run(fmt.Sprintf("len=%04d/AVX2", n), func(b *testing.B) {
			b.SetBytes(int64(n))
			for b.Loop() {
				sink += stringStopIndexAVX2(data)
			}
		})
	}
}
