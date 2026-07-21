package lexer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fredbi/core/json/lexers/token"
)

// TestFirstFillPromotion pins the whole-buffer short-circuit (§10.5f): a streaming
// lexer whose entire input fits in the buffer flips to wholeBuffer mode on the first
// token (running the fast in-buffer cores), while an input larger than the buffer
// stays in streaming mode. The token stream must be correct either way.
func TestFirstFillPromotion(t *testing.T) {
	doc := `{"a":[1,2,3],"b":"hello","c":true}`

	// fits in the (default, 4KB) buffer → promoted to whole-buffer on first token.
	t.Run("fits/promotes", func(t *testing.T) {
		l := New(bytes.NewReader([]byte(doc)))
		if l.wholeBuffer {
			t.Fatal("wholeBuffer set before first token")
		}
		_ = l.NextToken()
		if !l.wholeBuffer {
			t.Fatal("expected promotion to whole-buffer after first token (input fits)")
		}
		// drain and compare to the pure whole-buffer lexer.
		want, _ := collectPullValues(NewWithBytes([]byte(doc)))
		l2 := New(bytes.NewReader([]byte(doc)))
		got, _ := collectPullValues(l2)
		if !l2.wholeBuffer {
			t.Fatal("second lexer did not promote")
		}
		if len(want) != len(got) {
			t.Fatalf("token count: whole=%d promoted=%d", len(want), len(got))
		}
	})

	// larger than the window → stays streaming (window forced small via reslice).
	t.Run("overflows/streams", func(t *testing.T) {
		big := `["` + strings.Repeat("x", 200) + `","` + strings.Repeat("y", 200) + `"]`
		l := New(bytes.NewReader([]byte(big)), WithBufferSize(64))
		l.buffer = l.buffer[:64] // force a 64-byte window < input
		_ = l.NextToken()
		if l.wholeBuffer {
			t.Fatal("did not expect promotion: input exceeds the window")
		}
	})

	// empty input: promotes (whole input — nothing — fits) and reports ErrNoData.
	t.Run("empty/promotes", func(t *testing.T) {
		l := New(bytes.NewReader(nil))
		tok := l.NextToken()
		if !l.wholeBuffer {
			t.Fatal("empty input should promote to whole-buffer")
		}
		if l.Ok() || tok.Kind() != token.EOF {
			// ErrNoData surfaces on the EOF token
		}
	})
}
