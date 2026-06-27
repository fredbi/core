package lab

import (
	"testing"

	"github.com/fredbi/core/json/benchmarks/lexers/workloads"
	"github.com/fredbi/core/json/lexers/token"
)

var benchSink int

// BenchmarkDevirt measures the devirtualization gap in ONE binary: the generic
// cores (routed through the generics dictionary) vs the generated monomorphized
// cores (direct, inlined policy calls), over the micro workloads. Both run in the
// same package so there is no cross-package code-alignment noise (unlike
// lab-vs-reference). Pull reuses one lexer (ResetWithBytes) to isolate the scan
// cost; push constructs per iteration as the iterator normally would.
func BenchmarkDevirt(b *testing.B) {
	for _, w := range workloads.Micro() {
		data := w.Data

		b.Run(w.Name, func(b *testing.B) {
			b.Run("generic/pull", func(b *testing.B) {
				lex := NewWithBytes(nil)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					lex.ResetWithBytes(data)
					for {
						t := lex.nextTokenGeneric()
						if !lex.Ok() || t.Kind() == token.EOF {
							break
						}
						benchSink += int(t.Kind())
					}
				}
			})
			b.Run("devirt/pull", func(b *testing.B) { // NextToken = devirt post-adoption
				lex := NewWithBytes(nil)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					lex.ResetWithBytes(data)
					for {
						t := lex.NextToken()
						if !lex.Ok() || t.Kind() == token.EOF {
							break
						}
						benchSink += int(t.Kind())
					}
				}
			})

			b.Run("generic/push", func(b *testing.B) {
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					for tok := range NewWithBytes(data).tokensGeneric() {
						benchSink += int(tok.Kind())
					}
				}
			})
			b.Run("devirt/push", func(b *testing.B) { // Tokens() is the devirt path post-adoption
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					for tok := range NewWithBytes(data).Tokens() {
						benchSink += int(tok.Kind())
					}
				}
			})
		})
	}
}
