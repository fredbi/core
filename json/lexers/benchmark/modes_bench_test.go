package benchmark

// Compass benchmark (§10.5): a FAIR matrix of every way to drive the two lexers,
// across the whole corpus, so we can see which usage laggards on which payloads.
//
// 8 modes = {L, VL} × {buffer, reader} × {push (Tokens), pull (NextToken)}:
//
//	L/buffer/push   devirt push core (whole-buffer champion; iterator.go fast path)
//	L/buffer/pull   scanTokenBufferG (yield->return port of the push core)
//	L/reader/push   Tokens() over a reader falls through to the NextToken loop
//	                (iterator.go:47) — pull + a range-over-func closure, NOT a
//	                distinct optimized path; expected ≈ L/reader/pull
//	L/reader/pull   scanTokenStreamG (§10.3 streaming fast paths)
//	VL/...          same four with the verbatim policy (blanks + line/col baked in)
//
// Fairness: each lexer is constructed ONCE and Reset per iteration (buffer aliases
// the input; reader reuses the internal window across resets), and a single
// bytes.Reader is rewound with Reset — so NO per-iteration construction / window
// allocation is charged, only steady-state scanning. Reader modes use the default
// 4KB window. THROWAWAY compass instrument; delete before retrofit to reference.

import (
	"bytes"
	"testing"

	lab "github.com/fredbi/core/json/lexers/default-lexer/lab"
	"github.com/fredbi/core/json/lexers/benchmark/workloads"
	"github.com/fredbi/core/json/lexers/token"
)

var modeSink int

func BenchmarkLexerModes(b *testing.B) {
	suite, err := workloads.Corpus()
	if err != nil {
		b.Fatal(err)
	}

	for _, wl := range suite {
		wl := wl
		data := wl.Data

		b.Run(wl.Name, func(b *testing.B) {
			// --- semantic lexer L ---

			b.Run("L/buffer/push", func(b *testing.B) {
				lx := lab.NewWithBytes(data)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					lx.ResetWithBytes(data)
					for t := range lx.Tokens() {
						modeSink += int(t.Kind())
					}
				}
			})

			b.Run("L/buffer/pull", func(b *testing.B) {
				lx := lab.NewWithBytes(data)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					lx.ResetWithBytes(data)
					for {
						t := lx.NextToken()
						if !lx.Ok() || t.Kind() == token.EOF {
							break
						}
						modeSink += int(t.Kind())
					}
				}
			})

			b.Run("L/reader/push", func(b *testing.B) {
				var br bytes.Reader
				br.Reset(data)
				lx := lab.New(&br)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					br.Reset(data)
					lx.ResetWithReader(&br)
					for t := range lx.Tokens() {
						modeSink += int(t.Kind())
					}
				}
			})

			b.Run("L/reader/pull", func(b *testing.B) {
				var br bytes.Reader
				br.Reset(data)
				lx := lab.New(&br)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					br.Reset(data)
					lx.ResetWithReader(&br)
					for {
						t := lx.NextToken()
						if !lx.Ok() || t.Kind() == token.EOF {
							break
						}
						modeSink += int(t.Kind())
					}
				}
			})

			// --- verbatim lexer VL ---

			b.Run("VL/buffer/push", func(b *testing.B) {
				lx := lab.NewVerbatimWithBytes(data)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					lx.ResetWithBytes(data)
					for t := range lx.Tokens() {
						modeSink += int(t.Kind())
					}
				}
			})

			b.Run("VL/buffer/pull", func(b *testing.B) {
				lx := lab.NewVerbatimWithBytes(data)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					lx.ResetWithBytes(data)
					for {
						t := lx.NextToken()
						if !lx.Ok() || t.Kind() == token.EOF {
							break
						}
						modeSink += int(t.Kind())
					}
				}
			})

			b.Run("VL/reader/push", func(b *testing.B) {
				var br bytes.Reader
				br.Reset(data)
				lx := lab.NewVerbatim(&br)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					br.Reset(data)
					lx.ResetWithReader(&br)
					for t := range lx.Tokens() {
						modeSink += int(t.Kind())
					}
				}
			})

			b.Run("VL/reader/pull", func(b *testing.B) {
				var br bytes.Reader
				br.Reset(data)
				lx := lab.NewVerbatim(&br)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					br.Reset(data)
					lx.ResetWithReader(&br)
					for {
						t := lx.NextToken()
						if !lx.Ok() || t.Kind() == token.EOF {
							break
						}
						modeSink += int(t.Kind())
					}
				}
			})

			// --- prototype state-based verbatim lexer VS (§10.5b): same verbatim
			// feature (raw values + blanks + position via accessors), light token.T ---

			b.Run("VS/buffer/push", func(b *testing.B) {
				lx := lab.NewVerbatimStateWithBytes(data)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					lx.ResetWithBytes(data)
					for t := range lx.Tokens() {
						modeSink += int(t.Kind())
					}
				}
			})

			b.Run("VS/buffer/pull", func(b *testing.B) {
				lx := lab.NewVerbatimStateWithBytes(data)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					lx.ResetWithBytes(data)
					for {
						t := lx.NextToken()
						if !lx.Ok() || t.Kind() == token.EOF {
							break
						}
						modeSink += int(t.Kind())
					}
				}
			})

			b.Run("VS/reader/push", func(b *testing.B) {
				var br bytes.Reader
				br.Reset(data)
				lx := lab.NewVerbatimState(&br)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					br.Reset(data)
					lx.ResetWithReader(&br)
					for t := range lx.Tokens() {
						modeSink += int(t.Kind())
					}
				}
			})

			b.Run("VS/reader/pull", func(b *testing.B) {
				var br bytes.Reader
				br.Reset(data)
				lx := lab.NewVerbatimState(&br)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					br.Reset(data)
					lx.ResetWithReader(&br)
					for {
						t := lx.NextToken()
						if !lx.Ok() || t.Kind() == token.EOF {
							break
						}
						modeSink += int(t.Kind())
					}
				}
			})
		})
	}
}
