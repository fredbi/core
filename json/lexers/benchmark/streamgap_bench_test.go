package benchmark

// THROWAWAY: baseline stream-vs-buffer gap for the default-lexer LAB pull path
// (§10.3). Drives L.NextToken over each corpus in whole-buffer mode vs streaming
// mode at two window sizes. Delete after capturing numbers.

import (
	"bytes"
	"testing"

	lab "github.com/fredbi/core/json/lexers/default-lexer/lab"
	"github.com/fredbi/core/json/lexers/benchmark/workloads"
	"github.com/fredbi/core/json/lexers/token"
)

var gapSink int

// Both drains BORROW from the pool and redeem, so the per-iteration lexer struct —
// and, crucially, the streaming window buffer (make([]byte, bufferSize), allocated
// once per construction and reused across every refill of a real stream) — drop out
// of the measurement. That buffer alloc is a construction cost, not a per-token
// scanning cost, so measuring it per-op both inflates the stream gap and adds GC
// noise (which buried the lever-A A/B). Pooling isolates pure scanning throughput.
func drainBuffer(data []byte) {
	lx, redeem := lab.BorrowLexerWithBytes(data)
	for {
		t := lx.NextToken()
		if !lx.Ok() || t.Kind() == token.EOF {
			break
		}
		gapSink += int(t.Kind())
	}
	redeem()
}

func drainStream(data []byte, bufSize int) {
	lx, redeem := lab.BorrowLexerWithReader(bytes.NewReader(data), lab.WithBufferSize(bufSize))
	for {
		t := lx.NextToken()
		if !lx.Ok() || t.Kind() == token.EOF {
			break
		}
		gapSink += int(t.Kind())
	}
	redeem()
}

func BenchmarkStreamGap(b *testing.B) {
	suite, err := workloads.Corpus()
	if err != nil {
		b.Fatal(err)
	}
	for _, wl := range suite {
		wl := wl
		b.Run(wl.Name, func(b *testing.B) {
			b.Run("buffer", func(b *testing.B) {
				b.SetBytes(int64(len(wl.Data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					drainBuffer(wl.Data)
				}
			})
			b.Run("stream-4k", func(b *testing.B) {
				b.SetBytes(int64(len(wl.Data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					drainStream(wl.Data, 4096)
				}
			})
			b.Run("stream-64k", func(b *testing.B) {
				b.SetBytes(int64(len(wl.Data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					drainStream(wl.Data, 64*1024)
				}
			})
		})
	}
}
