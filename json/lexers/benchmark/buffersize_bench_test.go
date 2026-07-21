package benchmark

// THROWAWAY (§10.5e): does the L streaming push/pull gap close as the buffer grows?
// If refill copies + span fallbacks are the culprit, reader should approach buffer as
// WithBufferSize -> filesize (one read, no spanning). If it plateaus BELOW buffer even
// when the buffer holds the whole file, the cost is structural (the stream core's
// per-token path, or — for push — the range-over-func closure over NextToken), not
// refills. Run on the largest-gap workloads (marine_ik 2.85MB, canada 0.26MB,
// twitter 0.6MB). Delete after diagnosis.

import (
	"bytes"
	"testing"

	lab "github.com/fredbi/core/json/lexers/default-lexer/lab"
	"github.com/fredbi/core/json/lexers/benchmark/workloads"
	"github.com/fredbi/core/json/lexers/token"
)

var bsSink int

func bsWorkloads(b *testing.B) map[string][]byte {
	suite, err := workloads.Corpus()
	if err != nil {
		b.Fatal(err)
	}
	out := map[string][]byte{}
	for _, wl := range suite {
		switch wl.Name {
		case "marine_ik", "canada_geometry", "twitter_status":
			out[wl.Name] = wl.Data
		}
	}

	return out
}

var bsSizes = []struct {
	name string
	size int
}{
	{"4K", 4 << 10}, {"16K", 16 << 10}, {"64K", 64 << 10},
	{"256K", 256 << 10}, {"1M", 1 << 20}, {"4M", 4 << 20},
}

func BenchmarkBufferSizeSweep(b *testing.B) {
	for name, data := range bsWorkloads(b) {
		data := data
		b.Run(name, func(b *testing.B) {
			// whole-buffer baselines
			b.Run("buffer/pull", func(b *testing.B) {
				lx := lab.NewWithBytes(data)
				b.SetBytes(int64(len(data)))
				b.ResetTimer()
				for b.Loop() {
					lx.ResetWithBytes(data)
					for {
						t := lx.NextToken()
						if !lx.Ok() || t.Kind() == token.EOF {
							break
						}
						bsSink += int(t.Kind())
					}
				}
			})
			b.Run("buffer/push", func(b *testing.B) {
				lx := lab.NewWithBytes(data)
				b.SetBytes(int64(len(data)))
				b.ResetTimer()
				for b.Loop() {
					lx.ResetWithBytes(data)
					for t := range lx.Tokens() {
						bsSink += int(t.Kind())
					}
				}
			})

			for _, bs := range bsSizes {
				bs := bs
				b.Run("reader/pull/"+bs.name, func(b *testing.B) {
					var br bytes.Reader
					br.Reset(data)
					lx := lab.New(&br, lab.WithBufferSize(bs.size))
					b.SetBytes(int64(len(data)))
					b.ResetTimer()
					for b.Loop() {
						br.Reset(data)
						lx.ResetWithReader(&br)
						for {
							t := lx.NextToken()
							if !lx.Ok() || t.Kind() == token.EOF {
								break
							}
							bsSink += int(t.Kind())
						}
					}
				})
				b.Run("reader/push/"+bs.name, func(b *testing.B) {
					var br bytes.Reader
					br.Reset(data)
					lx := lab.New(&br, lab.WithBufferSize(bs.size))
					b.SetBytes(int64(len(data)))
					b.ResetTimer()
					for b.Loop() {
						br.Reset(data)
						lx.ResetWithReader(&br)
						for t := range lx.Tokens() {
							bsSink += int(t.Kind())
						}
					}
				})
			}
		})
	}
}
