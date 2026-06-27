package lexers

import (
	"testing"

	"github.com/fredbi/core/json/benchmarks/lexers/jsontext"
	"github.com/fredbi/core/json/benchmarks/lexers/workloads"
	jlex "github.com/fredbi/core/json/lexers"
	deflex "github.com/fredbi/core/json/lexers/default-lexer"
	lab "github.com/fredbi/core/json/lexers/default-lexer/lab"
)

// TestMicroWorkloads guards the micro-benchmark suite: every laser-focused
// workload must (1) be valid JSON per jsontext (an independent RFC 8259 oracle —
// this is what proves shapes like "-0.44e10" really are legal), and (2) lex
// cleanly to EOF on both the reference and the lab lexer. A benchmark over a
// workload that errors mid-scan would silently time a partial run.
func TestMicroWorkloads(t *testing.T) {
	for _, w := range workloads.Micro() {
		if err := jsontext.Walk(w.Data); err != nil {
			t.Errorf("%s: not valid JSON per jsontext oracle: %v", w.Name, err)

			continue
		}

		ref := deflex.NewWithBytes(w.Data)
		if n, err := drain(ref); err != nil {
			t.Errorf("%s / reference: error after %d tokens: %v", w.Name, n, err)
		} else if n == 0 {
			t.Errorf("%s / reference: no tokens produced", w.Name)
		}

		sandbox := lab.NewWithBytes(w.Data)
		if n, err := drain(sandbox); err != nil {
			t.Errorf("%s / lab: error after %d tokens: %v", w.Name, n, err)
		} else if n == 0 {
			t.Errorf("%s / lab: no tokens produced", w.Name)
		}
	}
}

// BenchmarkMicro is the lean A/B harness for unitary design decisions: the
// reference default-lexer against the lab sandbox, each in the three modes we
// care about — bytes (pull NextToken), tokens (push Tokens iterator) and reset
// (pull, lexer reused across iterations so steady-state scanning reports 0
// allocs/op). It deliberately omits the full corpus gauntlet and the
// jsontext/easyjson peers (see plan §2.3, paused until gate reviews); calibrate
// against jsontext with BenchmarkLexers when a gap check is wanted.
func BenchmarkMicro(b *testing.B) {
	for _, w := range workloads.Micro() {
		data := w.Data

		b.Run(w.Name, func(b *testing.B) {
			// bytes: per-iteration construction + pull drain
			b.Run("reference/bytes", func(b *testing.B) {
				benchBytes(b, data, func(d []byte) jlex.Lexer { return deflex.NewWithBytes(d) })
			})
			b.Run("lab/bytes", func(b *testing.B) {
				benchBytes(b, data, func(d []byte) jlex.Lexer { return lab.NewWithBytes(d) })
			})

			// tokens: per-iteration construction + push iterator
			b.Run("reference/tokens", func(b *testing.B) {
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					for tok := range deflex.NewWithBytes(data).Tokens() {
						sink += int(tok.Kind())
					}
				}
			})
			b.Run("lab/tokens", func(b *testing.B) {
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					for tok := range lab.NewWithBytes(data).Tokens() {
						sink += int(tok.Kind())
					}
				}
			})

			// reset: lexer allocated once, reused — isolates steady-state scan cost
			b.Run("reference/reset", func(b *testing.B) {
				lex := deflex.NewWithBytes(nil)
				benchReset(b, data, lex.ResetWithBytes, lex)
			})
			b.Run("lab/reset", func(b *testing.B) {
				lex := lab.NewWithBytes(nil)
				benchReset(b, data, lex.ResetWithBytes, lex)
			})
		})
	}
}

// benchBytes times per-iteration construction plus a full pull drain.
func benchBytes(b *testing.B, data []byte, make func([]byte) jlex.Lexer) {
	b.Helper()
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = drain(make(data))
	}
}

// benchReset times steady-state scanning of a lexer reused across iterations
// (constructed once, outside the timed loop), so allocs/op should be 0.
func benchReset(b *testing.B, data []byte, reset func([]byte), lex jlex.Lexer) {
	b.Helper()
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		reset(data)
		_, _ = drain(lex)
	}
}
