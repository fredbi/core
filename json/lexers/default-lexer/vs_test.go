package lexer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fredbi/core/json/lexers/token"
)

// vsInputs are well-formed documents exercising whitespace (leading/trailing/pretty),
// escapes, \u, unicode, numbers, nesting and multi-line positions — the surface the
// verbatim lexer VL must reproduce faithfully.
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

// renderRaw reconstructs a token's raw source text (VL keeps string/number values
// raw, so this is exact). Separators are not elided by VL, so they arrive as
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

// TestVerbatimRoundTrip proves the verbatim lexer is FAITHFUL: reconstructing the
// input from each token's LeadingSpace() + raw text yields the original document byte
// for byte — the whole point of a verbatim lexer, delivered with a light token.T +
// accessors instead of a heavy per-token verbatim token. Whole-buffer and streaming.
func TestVerbatimRoundTrip(t *testing.T) {
	for _, in := range vsInputs() {
		data := []byte(in)

		if got, ok := reconstruct(NewVerbatimWithBytes(data)); ok && got != in {
			t.Errorf("buffer round-trip mismatch\n in=%q\nout=%q", in, got)
		}

		for _, bs := range []int{1, 3, 8, 32, 256} {
			if got, ok := reconstruct(NewVerbatim(bytes.NewReader(data), WithBufferSize(bs))); ok && got != in {
				t.Errorf("stream(bs=%d) round-trip mismatch\n in=%q\nout=%q", bs, in, got)
			}
		}
	}
}

// reconstruct rebuilds the source from a VL token stream: LeadingSpace() before each
// token (and before EOF, the trailing blanks) plus the token's raw text. ok is false
// if the document was rejected (nothing to round-trip against).
func reconstruct(vl *VL) (string, bool) {
	var b strings.Builder
	for {
		tok := vl.NextToken()
		b.Write(vl.LeadingSpace())
		if !vl.Ok() {
			return "", false
		}
		if tok.Kind() == token.EOF {
			return b.String(), true
		}
		b.WriteString(renderRaw(tok))
	}
}
