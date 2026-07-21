package lab

// THROWAWAY sizing (§10.5a): decompose the VL/pull per-token tax the compass found
// (VL/buffer/pull = 27% of L/buffer/pull). Where does the 3.7x go — the verbatim
// CORE's extra work (position tracking + per-byte whitespace walk + blanks slicing),
// or the per-token token.VT TOKEN cost (72B construct + return-by-value chain vs T's
// 32B)?
//
// Three policies over the SAME generic buffer core scanTokenBufferG (so the
// generics-dictionary emit cost is constant and cancels in the deltas):
//
//	S  semanticPolicy   token.T, tracksPosition=false  (batch-skip ws, cheap emit) = baseline
//	M  scanCostPolicy   token.T, tracksPosition=true   (position + per-byte ws + blanks, cheap 32B emit)
//	V  verbatimPolicy   token.VT, tracksPosition=true  (full: 72B VT construct + return chain)
//
//	S -> M  = verbatim CORE overhead (position/ws/blanks) with a cheap 32B return
//	M -> V  = VT TOKEN overhead (72B construct + return-by-value) on identical core work
//
// Payloads are synthetic (no cross-module corpus dep) and bracket the token-density
// regimes: dense compact tokens (worst per-token overhead), sparse long strings
// (amortized), and whitespace-heavy pretty (isolates the per-byte ws walk).
// Delete before retrofit to reference.

import (
	"strconv"
	"strings"
	"testing"
	"unsafe"

	"github.com/fredbi/core/json/lexers/token"
)

// scanCostPolicy tracks position like the verbatim policy (so the core does all the
// verbatim scanning work) but emits the cheap semantic token.T — isolating the core
// overhead from the VT token cost.
type scanCostPolicy struct{}

func (scanCostPolicy) emit(t token.T, _ []byte, _, _ int) token.T { return t }
func (scanCostPolicy) none() token.T                              { return token.None }
func (scanCostPolicy) eof(_ []byte) token.T                       { return token.EOFToken }
func (scanCostPolicy) tracksPosition() bool                       { return true }

// verbatimStatePolicy is the PROSPECTIVE state-based VL candidate (Fred's token-vs-
// state arbitrage): drop the heavy token.VT entirely and emit the light token.T,
// keeping the "verbatim feature" as LEXER STATE — the leading-blanks alias is stashed
// in l.blanks (retrievable via a LeadingSpace()-style accessor), the position is
// already in l.tokLine/l.tokCol (the core writes it when tracksPosition). So the
// per-token cost over the position-tracking core (M) is exactly ONE slice-header
// store. If C ≈ M and C ≪ V, the token SIZE was the whole VL/pull tax.
type verbatimStatePolicy struct{ l *L }

func (p verbatimStatePolicy) emit(t token.T, blanks []byte, _, _ int) token.T {
	p.l.blanks = blanks // stash the leading-blanks alias in lexer state (zero-copy)

	return t
}
func (verbatimStatePolicy) none() token.T        { return token.None }
func (verbatimStatePolicy) eof(_ []byte) token.T { return token.EOFToken }
func (verbatimStatePolicy) tracksPosition() bool { return true }

var vlCostSink int

func vlCostPayloads() []struct {
	name string
	data []byte
} {
	// dense compact ints: token-dense, ~no whitespace, tiny values — max per-token overhead.
	var di strings.Builder
	di.WriteByte('[')
	for i := 0; i < 4000; i++ {
		if i > 0 {
			di.WriteByte(',')
		}
		di.WriteString(strconv.Itoa(i % 1000))
	}
	di.WriteByte(']')

	// dense short strings.
	var ds strings.Builder
	ds.WriteByte('[')
	for i := 0; i < 4000; i++ {
		if i > 0 {
			ds.WriteByte(',')
		}
		ds.WriteString(`"ab"`)
	}
	ds.WriteByte(']')

	// sparse long strings: few tokens, big values — per-token overhead amortized.
	var ls strings.Builder
	ls.WriteByte('[')
	for i := 0; i < 200; i++ {
		if i > 0 {
			ls.WriteByte(',')
		}
		ls.WriteByte('"')
		ls.WriteString(strings.Repeat("x", 200))
		ls.WriteByte('"')
	}
	ls.WriteByte(']')

	// whitespace-heavy pretty object: isolates the verbatim core's per-byte ws walk.
	var pp strings.Builder
	pp.WriteString("{\n")
	for i := 0; i < 2000; i++ {
		if i > 0 {
			pp.WriteString(",\n")
		}
		pp.WriteString("    ")
		pp.WriteString(`"key`)
		pp.WriteString(strconv.Itoa(i))
		pp.WriteString(`" : `)
		pp.WriteString(strconv.Itoa(i))
	}
	pp.WriteString("\n}")

	return []struct {
		name string
		data []byte
	}{
		{"dense_ints", []byte(di.String())},
		{"dense_strs", []byte(ds.String())},
		{"long_strs", []byte(ls.String())},
		{"pretty_kv", []byte(pp.String())},
	}
}

func BenchmarkVLCost(b *testing.B) {
	b.Logf("sizeof token.T = %d, token.VT = %d bytes", unsafe.Sizeof(token.T{}), unsafe.Sizeof(token.VT{}))

	for _, pl := range vlCostPayloads() {
		data := pl.data

		b.Run(pl.name, func(b *testing.B) {
			b.Run("S_semantic", func(b *testing.B) {
				l := NewWithBytes(data)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					l.ResetWithBytes(data)
					for {
						t := scanTokenBufferG[token.T, semanticPolicy](l, semanticPolicy{})
						if l.err != nil || t.Kind() == token.EOF {
							break
						}
						vlCostSink += int(t.Kind())
					}
				}
			})

			b.Run("M_poscore", func(b *testing.B) {
				l := NewWithBytes(data)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					l.ResetWithBytes(data)
					for {
						t := scanTokenBufferG[token.T, scanCostPolicy](l, scanCostPolicy{})
						if l.err != nil || t.Kind() == token.EOF {
							break
						}
						vlCostSink += int(t.Kind())
					}
				}
			})

			b.Run("V_verbatim", func(b *testing.B) {
				l := NewWithBytes(data)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					l.ResetWithBytes(data)
					for {
						t := scanTokenBufferG[token.VT, verbatimPolicy](l, verbatimPolicy{})
						if l.err != nil || t.Kind() == token.EOF {
							break
						}
						vlCostSink += int(t.Kind())
					}
				}
			})

			// C = the prospective state-based VL candidate: light token.T + blanks
			// stashed in lexer state. Expected ≈ M, ≪ V if the theory holds.
			b.Run("C_vstate", func(b *testing.B) {
				l := NewWithBytes(data)
				p := verbatimStatePolicy{l: l}
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					l.ResetWithBytes(data)
					for {
						t := scanTokenBufferG[token.T, verbatimStatePolicy](l, p)
						if l.err != nil || t.Kind() == token.EOF {
							break
						}
						vlCostSink += int(t.Kind())
					}
				}
			})
		})
	}
}
