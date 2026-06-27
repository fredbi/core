package lab

import (
	"testing"
	"unsafe"
)

// Sizing experiment for the compact-token idea (plan §5.3): before refactoring the
// lab to a smaller token, measure the per-token cost the change would actually
// move — the by-value copy on each push yield / pull return plus the value access —
// in isolation, for both a kind-only consumer (our throughput benches: never
// touches the value) and a value-reading consumer (a real tree/store builder).
//
// Three representations:
//   tok32 — current token.T layout: []byte header (24) + delim+kind+bool + pad = 32.
//   tok16 — *[]byte (8) + 3 + pad = 16; needs a STABLE backing field to point at,
//           so the 24-byte slice header is materialized into memory per token.
//   tokU  — unsafe.Pointer (8) into the buffer + uint32 len (4) + 3 + pad = 16; the
//           pointer is computed directly (no header store, no backing field) and
//           the value is rebuilt on read via unsafe.Slice.

type tok32 struct {
	value          []byte
	valueDelimiter uint8
	kind           uint8
	valueBool      bool
}

type tok16 struct {
	value          *[]byte
	valueDelimiter uint8
	kind           uint8
	valueBool      bool
}

type tokU struct {
	ptr            unsafe.Pointer
	vlen           uint32
	valueDelimiter uint8
	kind           uint8
	valueBool      bool
}

var (
	gKindSink int
	gLenSink  int
)

// noinline consumers force a real call so the token is copied into the parameter
// (modelling the range-over-func yield / NextToken return copy). The readval
// variants touch the first value byte so the value-access cost is real for all
// three (a bare len() would be a field read and hide the deref difference).
//
//go:noinline
func consumeKind32(t tok32) bool { gKindSink += int(t.kind); return true }

//go:noinline
func consumeKind16(t tok16) bool { gKindSink += int(t.kind); return true }

//go:noinline
func consumeKindU(t tokU) bool { gKindSink += int(t.kind); return true }

//go:noinline
func consumeVal32(t tok32) bool { gKindSink += int(t.kind); gLenSink += int(t.value[0]); return true }

//go:noinline
func consumeVal16(t tok16) bool { gKindSink += int(t.kind); gLenSink += int((*t.value)[0]); return true }

//go:noinline
func consumeValU(t tokU) bool { gKindSink += int(t.kind); gLenSink += int(*(*byte)(t.ptr)); return true }

// span lists model the token-size/count distribution of three workload shapes at
// ~256KiB total content (matching the micro-benchmark corpus).
func spansFor(tokenLen int) ([][2]int, []byte) {
	const total = 256 * 1024
	count := total / tokenLen
	data := make([]byte, total)
	for i := range data {
		data[i] = byte('0' + i%10)
	}
	spans := make([][2]int, count)
	for i := range spans {
		off := (i * tokenLen) % (total - tokenLen)
		spans[i] = [2]int{off, off + tokenLen}
	}

	return spans, data
}

func TestCompactTokenSizes(t *testing.T) {
	if got := unsafe.Sizeof(tok32{}); got != 32 {
		t.Errorf("tok32 size = %d, want 32", got)
	}
	if got := unsafe.Sizeof(tok16{}); got != 16 {
		t.Errorf("tok16 size = %d, want 16", got)
	}
	if got := unsafe.Sizeof(tokU{}); got != 16 {
		t.Errorf("tokU size = %d, want 16", got)
	}
}

func BenchmarkCompactToken(b *testing.B) {
	shapes := []struct {
		name     string
		tokenLen int
	}{
		{"len1_separators", 1},
		{"len6_numbers", 6},
		{"len20_strings", 20},
	}

	for _, sh := range shapes {
		spans, data := spansFor(sh.tokenLen)
		b.Run(sh.name, func(b *testing.B) {
			b.Run("kindonly/tok32", func(b *testing.B) {
				b.SetBytes(int64(len(data)))
				b.ResetTimer()
				for range b.N {
					for _, s := range spans {
						if !consumeKind32(tok32{value: data[s[0]:s[1]:s[1]], kind: 4}) {
							break
						}
					}
				}
			})
			b.Run("kindonly/tok16", func(b *testing.B) {
				var stable []byte
				b.SetBytes(int64(len(data)))
				b.ResetTimer()
				for range b.N {
					for _, s := range spans {
						stable = data[s[0]:s[1]:s[1]]
						if !consumeKind16(tok16{value: &stable, kind: 4}) {
							break
						}
					}
				}
			})
			b.Run("kindonly/tokU", func(b *testing.B) {
				b.SetBytes(int64(len(data)))
				b.ResetTimer()
				for range b.N {
					for _, s := range spans {
						if !consumeKindU(tokU{ptr: unsafe.Pointer(&data[s[0]]), vlen: uint32(s[1] - s[0]), kind: 4}) {
							break
						}
					}
				}
			})

			b.Run("readval/tok32", func(b *testing.B) {
				b.SetBytes(int64(len(data)))
				b.ResetTimer()
				for range b.N {
					for _, s := range spans {
						if !consumeVal32(tok32{value: data[s[0]:s[1]:s[1]], kind: 4}) {
							break
						}
					}
				}
			})
			b.Run("readval/tok16", func(b *testing.B) {
				var stable []byte
				b.SetBytes(int64(len(data)))
				b.ResetTimer()
				for range b.N {
					for _, s := range spans {
						stable = data[s[0]:s[1]:s[1]]
						if !consumeVal16(tok16{value: &stable, kind: 4}) {
							break
						}
					}
				}
			})
			b.Run("readval/tokU", func(b *testing.B) {
				b.SetBytes(int64(len(data)))
				b.ResetTimer()
				for range b.N {
					for _, s := range spans {
						if !consumeValU(tokU{ptr: unsafe.Pointer(&data[s[0]]), vlen: uint32(s[1] - s[0]), kind: 4}) {
							break
						}
					}
				}
			})
		})
	}
}
