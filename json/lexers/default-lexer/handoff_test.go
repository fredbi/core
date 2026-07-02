package lexer

// Handoff gate: a caller may drive the lexer with Tokens() and then switch to
// NextToken() mid-stream (or the reverse). Both APIs share the lexer's state —
// the push back-end of Tokens() writes the cursor back to the struct on every
// exit path — so NextToken() resumes exactly where the iterator stopped.
//
// The one contract these tests pin down is the standard range-over-func
// semantics: the token delivered in the iteration where the caller breaks IS
// consumed, so NextToken() returns the FOLLOWING token (no repeat, no loss).
// Hence the loop records the token then breaks; NextToken() continues from there.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/lexers/token"
)

func acceptFixtures(t *testing.T) map[string][]byte {
	t.Helper()
	dir := filepath.Join(currentDir(), "testdata", "JSONTestSuite", "test_parsing")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	out := make(map[string][]byte)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".json") || !strings.HasPrefix(name, "y_") {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, rerr)
		out[name] = data
	}
	require.NotEmpty(t, out)

	return out
}

// TestHandoffTokensThenNextToken_Semantic: for L, taking `cut` tokens via
// Tokens() then finishing via NextToken() yields the same stream as a pure
// NextToken() drain, for every cut point on every accept fixture.
func TestHandoffTokensThenNextToken_Semantic(t *testing.T) {
	type tk struct {
		kind  token.Kind
		value string
	}
	mk := func(x token.T) tk { return tk{x.Kind(), string(x.Value())} }

	for name, data := range acceptFixtures(t) {
		var want []tk
		lp := NewWithBytes(data)
		for {
			x := lp.NextToken()
			if x.IsEOF() {
				break
			}
			want = append(want, mk(x))
		}
		require.NoError(t, lp.Err(), name)

		for cut := 1; cut <= len(want); cut++ {
			l := NewWithBytes(data)
			var got []tk
			n := 0
			for x := range l.Tokens() {
				got = append(got, mk(x))
				n++
				if n == cut {
					break
				}
			}
			for {
				x := l.NextToken()
				if x.IsEOF() {
					break
				}
				got = append(got, mk(x))
			}
			require.NoErrorf(t, l.Err(), "%s cut=%d", name, cut)
			require.Equalf(t, want, got, "%s cut=%d: Tokens()->NextToken() mismatch", name, cut)
		}
	}
}

// TestHandoffTokensThenNextToken_Verbatim: same as the semantic case but for VL,
// also checking that blanks and position survive the handoff.
func TestHandoffTokensThenNextToken_Verbatim(t *testing.T) {
	type tk struct {
		kind      token.Kind
		value     string
		blanks    string
		line, col int
	}
	mk := func(x token.VT) tk {
		return tk{x.Kind(), string(x.Value()), string(x.Blanks()), x.Line(), x.Column()}
	}

	for name, data := range acceptFixtures(t) {
		var want []tk
		lp := NewVerbatimWithBytes(data)
		for {
			x := lp.NextToken()
			if x.IsEOF() {
				break
			}
			want = append(want, mk(x))
		}
		require.NoError(t, lp.Err(), name)

		for cut := 1; cut <= len(want); cut++ {
			l := NewVerbatimWithBytes(data)
			var got []tk
			n := 0
			for x := range l.Tokens() {
				got = append(got, mk(x))
				n++
				if n == cut {
					break
				}
			}
			for {
				x := l.NextToken()
				if x.IsEOF() {
					break
				}
				got = append(got, mk(x))
			}
			require.NoErrorf(t, l.Err(), "%s cut=%d", name, cut)
			require.Equalf(t, want, got, "%s cut=%d: Tokens()->NextToken() mismatch", name, cut)
		}
	}
}
