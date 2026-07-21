package lexer

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/fredbi/core/json/lexers/token"
)

// forceStreamWindow builds a streaming lexer whose read window is EXACTLY bs bytes
// (reslicing past the WithBufferSize floor), so an input larger than bs stays in
// streaming mode — exercising the streaming push/pull cores rather than the
// whole-buffer short-circuit (§10.5f).
func forceStreamWindow(data []byte, bs int) *L {
	l := New(bytes.NewReader(data), WithBufferSize(bs))
	if bs < cap(l.buffer) {
		l.buffer = l.buffer[:bs]
	}

	return l
}

func collectPushT(seq func(func(token.T) bool)) [][2]string {
	var out [][2]string
	for t := range seq {
		out = append(out, [2]string{t.Kind().String(), string(t.Value())})
	}

	return out
}

// TestStreamPushEquivalence pins that the native streaming push core (§10.5g, backing
// Tokens() over a reader) yields exactly the same token stream as the streaming pull
// core (NextToken), for both the semantic lexer L and the state-based verbatim lexer
// VS, across window sizes that force streaming (small) and promotion (large).
func TestStreamPushEquivalence(t *testing.T) {
	inputs := append(streamFastInputs(), vsInputs()...)
	sizes := []int{1, 4, 16, 64, 1024}

	for _, in := range inputs {
		data := []byte(in)

		// reference: streaming pull via NextToken (forced small window).
		wantPull, pullErr := collectPullValues(forceStreamWindow(data, 8))

		for _, bs := range sizes {
			// L: Tokens() push over a reader.
			gotPush := collectPushT(func(yield func(token.T) bool) {
				lx := forceStreamWindow(data, bs)
				for tk := range lx.Tokens() {
					if !yield(tk) {
						return
					}
				}
			})
			if pullErr == nil && fmt.Sprint(wantPull) != fmt.Sprint(gotPush) {
				t.Errorf("L push vs pull mismatch, input %q bufsize %d\n pull=%v\n push=%v", in, bs, wantPull, gotPush)
			}
		}

		// VS: Tokens() push must equal VS NextToken pull (kinds + RAW values).
		vsPull := drainVSPull(forceStreamWindowVS(data, 8))
		for _, bs := range sizes {
			vsPush := drainVSPush(forceStreamWindowVS(data, bs))
			if vsPull.ok && fmt.Sprint(vsPull.toks) != fmt.Sprint(vsPush.toks) {
				t.Errorf("VS push vs pull mismatch, input %q bufsize %d\n pull=%v\n push=%v", in, bs, vsPull.toks, vsPush.toks)
			}
			// blanks accompanying each token must match too (accessor vs accessor).
			if vsPull.ok && fmt.Sprint(vsPull.blanks) != fmt.Sprint(vsPush.blanks) {
				t.Errorf("VS push vs pull BLANKS mismatch, input %q bufsize %d\n pull=%v\n push=%v", in, bs, vsPull.blanks, vsPush.blanks)
			}
		}
	}
}

func forceStreamWindowVS(data []byte, bs int) *VS {
	vs := NewVerbatimState(bytes.NewReader(data), WithBufferSize(bs))
	if bs < cap(vs.buffer) {
		vs.buffer = vs.buffer[:bs]
	}

	return vs
}

type vsDrain struct {
	toks   [][2]string
	blanks []string
	ok     bool
}

func drainVSPull(vs *VS) vsDrain {
	var d vsDrain
	for {
		t := vs.NextToken()
		if !vs.Ok() {
			d.ok = false

			return d
		}
		if t.Kind() == token.EOF {
			d.ok = true

			return d
		}
		d.toks = append(d.toks, [2]string{t.Kind().String(), string(t.Value())})
		d.blanks = append(d.blanks, string(vs.LeadingSpace()))
	}
}

func drainVSPush(vs *VS) vsDrain {
	var d vsDrain
	for t := range vs.Tokens() {
		d.toks = append(d.toks, [2]string{t.Kind().String(), string(t.Value())})
		d.blanks = append(d.blanks, string(vs.LeadingSpace()))
	}
	d.ok = vs.Ok()

	return d
}
