package lab

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/fredbi/core/json/lexers/token"
)

// collectPullValues drains a lexer via NextToken, returning (kind, value) pairs and
// the terminal error. It is mode-agnostic: buffer or stream, any buffer size.
func collectPullValues(l *L) ([][2]string, error) {
	var out [][2]string
	for {
		t := l.NextToken()
		if !l.Ok() {
			return out, l.Err()
		}
		if t.Kind() == token.EOF {
			return out, nil
		}
		out = append(out, [2]string{t.Kind().String(), string(t.Value())})
	}
}

// streamFastInputs exercises the streaming string fast path (§10.3 Phase 1) and its
// delegation seam: clean strings of many lengths (aliased when they fit the window,
// spanning when they don't), escapes at assorted offsets, \u sequences, control/
// unterminated errors, and mixed value/number/structure documents.
func streamFastInputs() []string {
	in := []string{
		`""`, `"a"`, `"ab"`, `"abcdefg"`, `"abcdefgh"`, `"abcdefghi"`,
		`"0123456789abcdef"`,                    // 16
		`"0123456789abcdef0123456789abcdef"`,    // 32
		`"0123456789abcdef0123456789abcdefX"`,   // 33
		`"` + strings.Repeat("x", 200) + `"`,    // long clean (AVX2 territory)
		`"esc\ttab"`, `"q\"q"`, `"back\\slash"`, `"sl\/ash"`,
		`"\n\r\t\b\f"`, `"a\nb\tc"`,
		`"uniécode"`, `"surrogate😀end"`,
		`"lead\nthen a long clean tail ` + strings.Repeat("y", 100) + `"`,
		`{"key":"value","k2":123,"k3":[true,false,null]}`,
		`{"description":"` + strings.Repeat("word ", 50) + `"}`,
		`[0,1,-1,42,-42,3.14,-3.14,1e10,1E-10,-0.44e10,12.3456E-3]`,
		"[\n\t \"a\" ,\n\t \"b\" \n]",
		// errors — buffer and stream must reach the same terminal state
		"\"unterminated", `"ctrl` + "\x01" + `"`, `"bad\xescape"`,
	}
	// strings whose closing quote lands at every offset near the small-buffer
	// boundaries, to stress the "stop exactly at window end" case.
	for _, l := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 15, 16, 17, 31, 32, 33} {
		in = append(in, `"`+strings.Repeat("a", l)+`"`)
	}

	return in
}

// TestStreamFastEquivalence pins the core Phase-1 invariant: for ANY input and ANY
// buffer size, streaming NextToken yields the exact same token stream (kinds AND
// decoded values) and terminal error as whole-buffer NextToken. Small buffers drive
// the span/delegate path; large ones drive the zero-copy alias fast path; the sizes
// between land the closing quote on either side of a window boundary.
func TestStreamFastEquivalence(t *testing.T) {
	bufSizes := []int{1, 2, 3, 4, 5, 7, 8, 15, 16, 17, 31, 32, 33, 64, 128, 1024}

	for _, in := range streamFastInputs() {
		data := []byte(in)

		want, wantErr := collectPullValues(NewWithBytes(data))

		for _, bs := range bufSizes {
			got, gotErr := collectPullValues(New(bytes.NewReader(data), WithBufferSize(bs)))

			if !sameErr(wantErr, gotErr) {
				t.Errorf("input %q bufsize %d: err mismatch: buffer=%v stream=%v", in, bs, wantErr, gotErr)
			}
			if fmt.Sprint(want) != fmt.Sprint(got) {
				t.Errorf("input %q bufsize %d: token stream mismatch\n buffer=%v\n stream=%v", in, bs, want, got)
			}
		}
	}
}

// TestStreamFastAliasesWindow asserts the fast path actually aliases the buffer
// (zero-copy) for a clean string that fits the window, rather than always copying
// through currentValue — the whole point of Phase 1. A returned String token whose
// value header points inside l.buffer proves the alias.
func TestStreamFastAliasesWindow(t *testing.T) {
	data := []byte(`"a clean string that fits comfortably in the window"`)
	l := New(bytes.NewReader(data), WithBufferSize(256))

	tok := l.NextToken()
	if tok.Kind() != token.String {
		t.Fatalf("expected String, got %v", tok.Kind())
	}
	val := tok.Value()
	buf := l.buffer[:cap(l.buffer)]
	// the value must be a sub-slice of the lexer's window (alias), not a copy.
	aliased := len(val) > 0 &&
		&val[0] == &buf[l.consumed-1-len(val)] // value ends just before the closing quote at consumed-1
	if !aliased {
		t.Errorf("streaming clean string was not aliased to the window (copied instead)")
	}
}
