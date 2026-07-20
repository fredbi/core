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

func drainBuffer(data []byte) {
	lx := lab.NewWithBytes(data)
	for {
		t := lx.NextToken()
		if !lx.Ok() || t.Kind() == token.EOF {
			break
		}
		gapSink += int(t.Kind())
	}
}

func drainStream(data []byte, bufSize int) {
	lx := lab.New(bytes.NewReader(data), lab.WithBufferSize(bufSize))
	for {
		t := lx.NextToken()
		if !lx.Ok() || t.Kind() == token.EOF {
			break
		}
		gapSink += int(t.Kind())
	}
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
