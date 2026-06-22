package lexers

import (
	"strings"
	"testing"

	"github.com/fredbi/core/json/benchmarks/lexers/easyjson"
	"github.com/fredbi/core/json/benchmarks/lexers/stdlib"
	"github.com/fredbi/core/json/benchmarks/lexers/workloads"
	jlex "github.com/fredbi/core/json/lexers"
	deflex "github.com/fredbi/core/json/lexers/default-lexer"
)

// deeplyNested workloads (tens of thousands of levels) are excluded from the
// easyjson comparison: its typed API forces a recursive descent that would
// overflow the goroutine stack, whereas the default-lexer scans iteratively.
// That is an API-shape difference, not a lexing-speed one.
func deeplyNested(name string) bool {
	return strings.HasPrefix(name, "nested_")
}

// factory builds a fresh lexers.Lexer over data, together with a release
// callback (e.g. to redeem a pooled lexer); release is a no-op when not needed.
type factory struct {
	name string
	make func(data []byte) (jlex.Lexer, func())
}

func factories() []factory {
	noRelease := func() {}

	return []factory{
		{
			name: "default-lexer/bytes",
			make: func(d []byte) (jlex.Lexer, func()) {
				return deflex.NewWithBytes(d), noRelease
			},
		},
		{
			name: "default-lexer/pooled",
			make: func(d []byte) (jlex.Lexer, func()) {
				l := deflex.BorrowLexerWithBytes(d)

				return l, func() { deflex.RedeemLexer(l) }
			},
		},
		{
			name: "stdlib/bytes",
			make: func(d []byte) (jlex.Lexer, func()) {
				return stdlib.NewWithBytes(d), noRelease
			},
		},
	}
}

// drain consumes every token until EOF or error, returning the token count.
func drain(lex jlex.Lexer) (int, error) {
	n := 0
	for {
		tok := lex.NextToken()
		if !lex.Ok() {
			return n, lex.Err()
		}
		if tok.IsEOF() {
			return n, nil
		}
		n++
	}
}

// TestWorkloadsLex guards the benchmarks: every workload must lex cleanly to EOF
// on every implementation, otherwise a benchmark could silently time a partial
// or errored scan.
func TestWorkloadsLex(t *testing.T) {
	suite, err := workloads.Suite()
	if err != nil {
		t.Fatalf("loading workloads: %v", err)
	}

	for _, w := range suite {
		for _, f := range factories() {
			lex, release := f.make(w.Data)
			n, err := drain(lex)
			release()
			if err != nil {
				t.Errorf("%s / %s: unexpected error after %d tokens: %v", w.Name, f.name, n, err)
			}
			if n == 0 {
				t.Errorf("%s / %s: no tokens produced", w.Name, f.name)
			}
		}

		// verbatim lexer has its own token type, drive it separately
		vl := deflex.NewVerbatimWithBytes(w.Data)
		n := 0
		for {
			tok := vl.NextToken()
			if !vl.Ok() || tok.IsEOF() {
				break
			}
			n++
		}
		if err := vl.Err(); err != nil {
			t.Errorf("%s / default-lexer/verbatim: unexpected error after %d tokens: %v", w.Name, n, err)
		}

		// easyjson generic walk (recursive; skip the deeply-nested synthetics)
		if !deeplyNested(w.Name) {
			if err := easyjson.Walk(w.Data); err != nil {
				t.Errorf("%s / easyjson: unexpected error: %v", w.Name, err)
			}
		}
	}
}

func BenchmarkLexers(b *testing.B) {
	suite, err := workloads.Suite()
	if err != nil {
		b.Fatalf("loading workloads: %v", err)
	}

	for _, w := range suite {
		b.Run(w.Name, func(b *testing.B) {
			for _, f := range factories() {
				b.Run(f.name, func(b *testing.B) {
					b.SetBytes(int64(len(w.Data)))
					b.ReportAllocs()
					b.ResetTimer()

					for range b.N {
						lex, release := f.make(w.Data)
						_, _ = drain(lex)
						release()
					}
				})
			}

			b.Run("default-lexer/verbatim", func(b *testing.B) {
				b.SetBytes(int64(len(w.Data)))
				b.ReportAllocs()
				b.ResetTimer()

				for range b.N {
					vl := deflex.NewVerbatimWithBytes(w.Data)
					for {
						tok := vl.NextToken()
						if !vl.Ok() || tok.IsEOF() {
							break
						}
					}
				}
			})

			// easyjson []byte-only pull lexer, generic recursive walk; numbers
			// taken raw (no numeric conversion) for an apples-to-apples comparison
			if !deeplyNested(w.Name) {
				b.Run("easyjson/bytes", func(b *testing.B) {
					b.SetBytes(int64(len(w.Data)))
					b.ReportAllocs()
					b.ResetTimer()

					for range b.N {
						_ = easyjson.Walk(w.Data)
					}
				})
			}
		})
	}
}
