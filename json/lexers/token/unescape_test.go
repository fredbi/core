package token

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVTUnescaped(t *testing.T) {
	cases := []struct {
		raw string // VT.Value() (raw source, escapes intact)
		dec string // expected VT.Unescaped()
	}{
		{"plain", "plain"},
		{"a\\nb", "a\nb"},
		{"tab\\tx", "tab\tx"},
		{"q\\\"q", `q"q`},
		{"back\\\\slash", `back\slash`},
		{"slash\\/x", "slash/x"},
		{"\\b\\f\\r", "\b\f\r"},
		{"\\u0041", "A"},
		{"caf\\u00e9", "café"},
		{"\\ud83d\\ude00", "😀"}, // surrogate pair
		{"a\\u0041\\u0042b", "aABb"},
		{"literalé", "literalé"}, // already-decoded UTF-8 passes through
		{"", ""},
	}
	for _, c := range cases {
		vt := MakeWithValue(String, []byte(c.raw)).AsVerbatim(nil)
		assert.Equalf(t, c.dec, vt.UnescapedString(), "decoded for raw %q", c.raw)
		assert.Equalf(t, c.dec, string(vt.Unescaped()), "decoded bytes for raw %q", c.raw)
		// Value() is always the raw form, untouched
		assert.Equalf(t, c.raw, string(vt.Value()), "Value stays raw for %q", c.raw)
	}
}

func TestVTUnescapedNonStringPassthrough(t *testing.T) {
	// numbers/booleans/null never carry escapes: Unescaped == Value, no alloc.
	num := MakeWithValue(Number, []byte("12.5")).AsVerbatim(nil)
	assert.Equal(t, []byte("12.5"), num.Unescaped())

	// escape-free string returns the same backing slice (no allocation).
	s := MakeWithValue(String, []byte("clean")).AsVerbatim(nil)
	assert.Equal(t, "clean", s.UnescapedString())
}
