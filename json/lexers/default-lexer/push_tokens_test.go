package lexer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/lexers/token"
)

// TestScanPushMatchesNextToken asserts that the whole-buffer push back-end of
// Tokens() (scanPush) yields byte-for-byte the same token stream — and reaches
// the same final error/Ok state — as the pull NextToken loop, across the entire
// JSONTestSuite parsing corpus (valid, invalid and implementation-defined).
func TestScanPushMatchesNextToken(t *testing.T) {
	type tok struct {
		kind  token.Kind
		value string
	}

	pullStream := func(data []byte) (out []tok, ok bool, errStr string) {
		l := NewWithBytes(data)
		for {
			tk := l.NextToken()
			if !l.Ok() {
				return out, false, errString(l.Err())
			}
			if tk.IsEOF() {
				return out, true, ""
			}
			out = append(out, tok{tk.Kind(), string(tk.Value())})
		}
	}

	pushStream := func(data []byte) (out []tok, ok bool, errStr string) {
		l := NewWithBytes(data)
		for tk := range l.Tokens() {
			out = append(out, tok{tk.Kind(), string(tk.Value())})
		}
		if !l.Ok() {
			return out, false, errString(l.Err())
		}

		return out, true, ""
	}

	dir := filepath.Join(currentDir(), "testdata", "JSONTestSuite", "test_parsing")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	var checked int
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, rerr)

		pullToks, pullOk, pullErr := pullStream(data)
		pushToks, pushOk, pushErr := pushStream(data)

		require.Equalf(t, pullOk, pushOk, "Ok mismatch on %s (pull err=%q push err=%q)", name, pullErr, pushErr)
		require.Equalf(t, pullErr, pushErr, "error mismatch on %s", name)
		require.Equalf(t, pullToks, pushToks, "token stream mismatch on %s", name)
		checked++
	}

	t.Logf("compared push vs pull token streams on %d fixtures", checked)
	require.Positive(t, checked)
}

func errString(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}

// naive reference for indexStringSpecial.
func refIndexStringSpecial(b []byte) int {
	for i := 0; i < len(b); i++ {
		if c := b[i]; c == '"' || c == '\\' || c < 0x20 {
			return i
		}
	}
	return len(b)
}

func TestIndexStringSpecial(t *testing.T) {
	// exhaustive-ish: every length up to past two words, every special position,
	// plus high bytes (>=0x80) which must not trigger false positives.
	for n := 0; n <= 20; n++ {
		for pos := 0; pos <= n; pos++ {
			for _, special := range []byte{'"', '\\', 0x00, 0x1f, '\n', '\t'} {
				b := make([]byte, n)
				for k := range b {
					b[k] = 'x' // safe filler
				}
				if pos < n {
					b[pos] = special
				}
				// sprinkle a high byte that must be ignored
				if n > 2 {
					b[(pos+1)%n] = 0xC3
				}
				got := indexStringSpecial(b)
				want := refIndexStringSpecial(b)
				require.Equalf(t, want, got, "n=%d pos=%d special=%#x b=%v", n, pos, special, b)
			}
		}
	}
}
