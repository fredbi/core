package lab

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fredbi/core/json/lexers/token"
)

// vsInputs are well-formed documents exercising whitespace (leading/trailing/pretty),
// escapes, \u, unicode, numbers, nesting and multi-line positions — the surface the
// state-based VL must reproduce faithfully.
func vsInputs() []string {
	return []string{
		`""`, `"a"`, `"hello world"`, `"esc\ttab\n\"q\""`, `"uniécode"`, `"héllo😀"`,
		`0`, `-0`, `42`, `-3.14`, `1e10`, `12.34E-5`,
		`true`, `false`, `null`,
		`[]`, `{}`, `[1,2,3]`, `{"a":1}`, `{"a":1,"b":[true,false,null]}`,
		`  42  `, "\n\t[ 1 ,\n\t  2 ]\n", `{ "k" : "v" , "n" : 3.5 }`,
		"{\n  \"a\": 1,\n  \"b\": {\n    \"c\": [ 10, 20 ]\n  }\n}",
		`   `, // whitespace-only (no value → ErrNoData; still must reconstruct the blanks)
		`{"desc":"` + strings.Repeat("word ", 40) + `\n" }`,
		`[` + strings.Repeat(`"`+strings.Repeat("x", 100)+`",`, 20) + `"tail"]`,
	}
}

// renderRaw reconstructs a token's raw source text (VS keeps string/number values
// raw, so this is exact). Separators are not elided by VS, so they arrive as
// delimiter tokens.
func renderRaw(t token.T) string {
	switch t.Kind() {
	case token.String, token.Key:
		return `"` + string(t.Value()) + `"`
	case token.Number:
		return string(t.Value())
	case token.Boolean:
		if t.Bool() {
			return "true"
		}

		return "false"
	case token.Null:
		return "null"
	case token.Delimiter:
		return t.Delimiter().String()
	default:
		return ""
	}
}

// TestVSEquivalenceWithVL pins that the state-based lexer VS produces, token for
// token, the SAME information as the verbatim lexer VL — kind, raw value, preceding
// blanks and 1-based position — only carried as lexer state instead of baked into a
// token.VT. Checked in both whole-buffer and streaming modes (several window sizes),
// and terminal error states must match.
func TestVSEquivalenceWithVL(t *testing.T) {
	for _, in := range vsInputs() {
		data := []byte(in)

		t.Run("buffer/"+in, func(t *testing.T) {
			compareVLVS(t, NewVerbatimWithBytes(data), NewVerbatimStateWithBytes(data))
		})

		for _, bs := range []int{1, 4, 16, 64, 1024} {
			t.Run("stream/"+in, func(t *testing.T) {
				vl := NewVerbatim(bytes.NewReader(data), WithBufferSize(bs))
				vs := NewVerbatimState(bytes.NewReader(data), WithBufferSize(bs))
				compareVLVS(t, vl, vs)
			})
		}
	}
}

func compareVLVS(t *testing.T, vl *VL, vs *VS) {
	t.Helper()
	for i := 0; ; i++ {
		vt := vl.NextToken()
		st := vs.NextToken()

		if vt.Kind() != st.Kind() {
			t.Fatalf("token %d: kind VL=%v VS=%v", i, vt.Kind(), st.Kind())
		}
		if !bytes.Equal(vt.Value(), st.Value()) {
			t.Fatalf("token %d (%v): value VL=%q VS=%q", i, vt.Kind(), vt.Value(), st.Value())
		}
		if !bytes.Equal(vt.Blanks(), vs.LeadingSpace()) {
			t.Fatalf("token %d (%v): blanks VL=%q VS=%q", i, vt.Kind(), vt.Blanks(), vs.LeadingSpace())
		}
		// Position equivalence holds for real tokens: both reflect l.tokLine/tokCol
		// snapshotted at the token start. The EOF token is exempt — token.VT bakes
		// (0,0) into MakeVerbatimEOF, while the accessor (on BOTH VL and VS) keeps the
		// last real token's position; the two representations disagree there by design.
		if vt.Kind() != token.EOF && (vt.Line() != vs.Line() || vt.Column() != vs.Column()) {
			t.Fatalf("token %d (%v): pos VL=(%d,%d) VS=(%d,%d)", i, vt.Kind(), vt.Line(), vt.Column(), vs.Line(), vs.Column())
		}

		if vl.Ok() != vs.Ok() {
			t.Fatalf("token %d: Ok mismatch VL=%v VS=%v (errs VL=%v VS=%v)", i, vl.Ok(), vs.Ok(), vl.Err(), vs.Err())
		}
		if !vl.Ok() || vt.Kind() == token.EOF {
			return
		}
	}
}

// TestVSRoundTrip proves the state-based lexer is FAITHFUL: reconstructing the input
// from each token's LeadingSpace() + raw text yields the original document byte for
// byte — the whole point of a verbatim lexer, now delivered with a light token.T +
// accessors instead of a heavy token.VT. Whole-buffer and streaming modes.
func TestVSRoundTrip(t *testing.T) {
	for _, in := range vsInputs() {
		data := []byte(in)

		if got, ok := reconstruct(NewVerbatimStateWithBytes(data)); ok && got != in {
			t.Errorf("buffer round-trip mismatch\n in=%q\nout=%q", in, got)
		}

		for _, bs := range []int{1, 3, 8, 32, 256} {
			if got, ok := reconstruct(NewVerbatimState(bytes.NewReader(data), WithBufferSize(bs))); ok && got != in {
				t.Errorf("stream(bs=%d) round-trip mismatch\n in=%q\nout=%q", bs, in, got)
			}
		}
	}
}

// reconstruct rebuilds the source from a VS token stream: LeadingSpace() before each
// token (and before EOF, the trailing blanks) plus the token's raw text. ok is false
// if the document was rejected (nothing to round-trip against).
func reconstruct(vs *VS) (string, bool) {
	var b strings.Builder
	for {
		tok := vs.NextToken()
		b.Write(vs.LeadingSpace())
		if !vs.Ok() {
			return "", false
		}
		if tok.Kind() == token.EOF {
			return b.String(), true
		}
		b.WriteString(renderRaw(tok))
	}
}
