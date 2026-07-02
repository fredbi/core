package lab

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/lexers/token"
)

func isStr(k token.Kind) bool { return k == token.String || k == token.Key }

// firstVerbatimString drains to the first String/Key token and returns raw + decoded.
func firstVerbatimString(t *testing.T, vl *VL, src string) (raw, dec string) {
	t.Helper()
	for {
		tok := vl.NextToken()
		require.Truef(t, vl.Ok(), "lex error on %q: %v", src, vl.Err())
		if tok.IsEOF() {
			t.Fatalf("no string token in %q", src)
		}
		if isStr(tok.Kind()) {
			return string(tok.Value()), tok.UnescapedString()
		}
	}
}

func TestVerbatimKeepsRawStrings(t *testing.T) {
	cases := []struct {
		src     string // a bare JSON string literal (with quotes)
		wantRaw string // expected VT.Value() (between quotes, escapes intact)
		wantDec string // expected VT.Unescaped()
	}{
		{`"plain"`, "plain", "plain"},
		{`"a\nb"`, `a\nb`, "a\nb"},
		{`"tab\tx"`, `tab\tx`, "tab\tx"},
		{`"q\"q"`, `q\"q`, `q"q`},
		{`"back\\slash"`, `back\\slash`, `back\slash`},
		{`"slash\/x"`, `slash\/x`, "slash/x"},
		{`"Az"`, `Az`, "Az"},
		{`"emoji😀!"`, `emoji😀!`, "emoji😀!"},
		{`"café"`, `café`, "café"},
		{`"literalé"`, "literalé", "literalé"},
	}
	for _, c := range cases {
		raw, dec := firstVerbatimString(t, NewVerbatimWithBytes([]byte(c.src)), c.src)
		assert.Equalf(t, c.wantRaw, raw, "raw for %s", c.src)
		assert.Equalf(t, c.wantDec, dec, "decoded for %s", c.src)
	}
}

func TestVerbatimRawStreamingMatchesWholeBuffer(t *testing.T) {
	srcs := []string{
		`"a\nb\tcA😀tail with spaces and é"`,
		`"éèê"`,
		`"no escapes just a fairly long plain ascii string value"`,
		`"mixed \"quotes\" and \\ backslashes \/ slashes"`,
	}
	for _, s := range srcs {
		wRaw, wDec := firstVerbatimString(t, NewVerbatimWithBytes([]byte(s)), s)
		// tiny buffer forces refills mid-string
		gRaw, gDec := firstVerbatimString(t, NewVerbatim(strings.NewReader(s), WithBufferSize(4)), s)
		assert.Equalf(t, wRaw, gRaw, "streaming raw for %s", s)
		assert.Equalf(t, wDec, gDec, "streaming decoded for %s", s)
	}
}

func TestSemanticStillDecodes(t *testing.T) {
	l := NewWithBytes([]byte(`"a\nbA"`))
	tok := l.NextToken()
	require.True(t, l.Ok())
	assert.Equal(t, "a\nbA", string(tok.Value()), "semantic Value must stay decoded")
}

func TestVerbatimRawRejectsBadEscapes(t *testing.T) {
	bad := []string{
		`"bad\q"`,
		`"short\u12"`,
		`"badhex\uZZZZ"`,
		`"lonesurr\ud83d"`,
	}
	for _, s := range bad {
		vl := NewVerbatimWithBytes([]byte(s))
		for vl.Ok() {
			if vl.NextToken().IsEOF() {
				break
			}
		}
		assert.Falsef(t, vl.Ok(), "expected error for %s", s)
	}
}
