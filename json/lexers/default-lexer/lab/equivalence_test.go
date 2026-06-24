package lab_test

// Equivalence harness: the lab package starts as a verbatim copy of the
// reference lexer, and these tests assert it stays behaviorally identical (token
// stream + error state) across every conformance fixture, in every mode. As lab
// diverges during the unification spike, any behavior change shows up here
// immediately, side by side with the reference.

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	lexer "github.com/fredbi/core/json/lexers/default-lexer"
	lab "github.com/fredbi/core/json/lexers/default-lexer/lab"
	"github.com/fredbi/core/json/lexers/token"
)

func currentDir() string {
	_, filename, _, _ := runtime.Caller(1)

	return filepath.Dir(filename)
}

func errStr(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}

// tok is a comparable projection of a semantic token: kind + value (key vs
// string is already distinguished by kind).
type tok struct {
	kind  token.Kind
	value string
}

// semanticLexer is the common surface of lexer.L and lab.L we drive here.
type semanticLexer interface {
	NextToken() token.T
	Ok() bool
	Err() error
}

func drainSemantic(l semanticLexer) ([]tok, string) {
	var out []tok
	for {
		tk := l.NextToken()
		if !l.Ok() {
			return out, errStr(l.Err())
		}
		if tk.IsEOF() {
			return out, ""
		}
		out = append(out, tok{kind: tk.Kind(), value: string(tk.Value())})
	}
}

// conformanceFixtures returns the contents of every JSONTestSuite parsing
// fixture, keyed by file name.
func conformanceFixtures(t *testing.T) map[string][]byte {
	t.Helper()
	dir := filepath.Join(currentDir(), "..", "testdata", "JSONTestSuite", "test_parsing")
	entries, err := os.ReadDir(dir)
	require.NoErrorf(t, err, "cannot read conformance fixtures at %s", dir)

	out := make(map[string][]byte, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".json" {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, rerr)
		out[name] = data
	}
	require.NotEmpty(t, out)

	return out
}

func TestLabEquivalence(t *testing.T) {
	fixtures := conformanceFixtures(t)

	t.Run("bytes NextToken", func(t *testing.T) {
		for name, data := range fixtures {
			wantToks, wantErr := drainSemantic(lexer.NewWithBytes(data))
			gotToks, gotErr := drainSemantic(lab.NewWithBytes(data))
			require.Equalf(t, wantErr, gotErr, "error mismatch on %s", name)
			require.Equalf(t, wantToks, gotToks, "token stream mismatch on %s", name)
		}
	})

	t.Run("streaming NextToken (64B buffer)", func(t *testing.T) {
		for name, data := range fixtures {
			wantToks, wantErr := drainSemantic(lexer.New(bytes.NewReader(data), lexer.WithBufferSize(64)))
			gotToks, gotErr := drainSemantic(lab.New(bytes.NewReader(data), lab.WithBufferSize(64)))
			require.Equalf(t, wantErr, gotErr, "error mismatch on %s", name)
			require.Equalf(t, wantToks, gotToks, "token stream mismatch on %s", name)
		}
	})

	t.Run("Tokens() iterator (whole-buffer push path)", func(t *testing.T) {
		for name, data := range fixtures {
			refL := lexer.NewWithBytes(data)
			var want []tok
			for tk := range refL.Tokens() {
				want = append(want, tok{kind: tk.Kind(), value: string(tk.Value())})
			}
			wantErr := errStr(refL.Err())

			labL := lab.NewWithBytes(data)
			var got []tok
			for tk := range labL.Tokens() {
				got = append(got, tok{kind: tk.Kind(), value: string(tk.Value())})
			}
			gotErr := errStr(labL.Err())

			require.Equalf(t, wantErr, gotErr, "Tokens() error mismatch on %s", name)
			require.Equalf(t, want, got, "Tokens() stream mismatch on %s", name)
		}
	})
}

// vtok projects a verbatim token: kind + value + blanks + position.
type vtok struct {
	kind      token.Kind
	value     string
	blanks    string
	line, col int
}

// verbatimLexer is the common surface of lexer.VL and lab.VL.
type verbatimLexer interface {
	NextToken() token.VT
	Ok() bool
	Err() error
}

func drainVerbatim(l verbatimLexer) ([]vtok, string) {
	var out []vtok
	for {
		tk := l.NextToken()
		if !l.Ok() {
			return out, errStr(l.Err())
		}
		if tk.IsEOF() {
			return out, ""
		}
		out = append(out, vtok{
			kind:   tk.Kind(),
			value:  string(tk.Value()),
			blanks: string(tk.Blanks()),
			line:   tk.Line(),
			col:    tk.Col(),
		})
	}
}

func TestLabVerbatimEquivalence(t *testing.T) {
	fixtures := conformanceFixtures(t)

	for name, data := range fixtures {
		want, wantErr := drainVerbatim(lexer.NewVerbatimWithBytes(data))
		got, gotErr := drainVerbatim(lab.NewVerbatimWithBytes(data))

		require.Equalf(t, wantErr, gotErr, "VL error mismatch on %s", name)
		require.Equalf(t, want, got, "VL token stream mismatch on %s", name)
	}
}
