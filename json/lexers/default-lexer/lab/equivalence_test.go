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
	"strings"
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

// TestLabVerbatimPullMatchesPush gates stage 2: VL.NextToken (the unified generic
// pull core) must produce exactly the same stream and error state as VL.Tokens
// (the unified push core) on every fixture, valid or not. Since the push path is
// independently validated against the unified contract
// (TestLabVerbatimPushEquivalence), pull==push transitively validates the pull
// core. This replaces the old pull-vs-reference-VL check, which would now flag
// the intended unified-semantics changes (\u decoding fix + surrogate validation).
func TestLabVerbatimPullMatchesPush(t *testing.T) {
	fixtures := conformanceFixtures(t)

	for name, data := range fixtures {
		pull, pullErr := drainVerbatim(lab.NewVerbatimWithBytes(data))

		var push []vtok
		lp := lab.NewVerbatimWithBytes(data)
		for tk := range lp.Tokens() {
			push = append(push, vtok{
				kind:   tk.Kind(),
				value:  string(tk.Value()),
				blanks: string(tk.Blanks()),
				line:   tk.Line(),
				col:    tk.Col(),
			})
		}
		pushErr := errStr(lp.Err())

		require.Equalf(t, pushErr, pullErr, "VL pull/push error mismatch on %s", name)
		require.Equalf(t, push, pull, "VL pull/push stream mismatch on %s", name)
	}
}

// TestLabVerbatimPushEquivalence gates stage 1: lab's native verbatim push path
// (VL.Tokens via the generic core) must reproduce the reference VL token stream
// — values, blanks AND positions — on every must-accept (y_) fixture.
//
// Scope is the y_ set deliberately. On invalid (n_) input the push core uses L's
// folded/deferred-error semantics, which may differ from VL's look-ahead. On
// implementation-defined (i_) input L and VL legitimately diverge — notably L
// validates \u surrogate escapes while the reference VL did not; the unified push
// core uses L's scanner, so unified VL now adopts L's stricter (correct)
// surrogate validation. That is an intended stage-1 behavior change, so i_ cases
// are out of scope for strict parity.
func TestLabVerbatimPushEquivalence(t *testing.T) {
	fixtures := conformanceFixtures(t)

	checked := 0
	for name, data := range fixtures {
		if !strings.HasPrefix(name, "y_") {
			continue // only must-accept fixtures have a single legitimate stream
		}

		// Oracle for the unified VL contract = reference L (non-eliding) for the
		// kind+value sequence (L's decoded values are the correct ones — the
		// reference VL has a \u-decoding bug) zipped with reference VL for the
		// blanks+position of each token.
		valOracle, vErr := drainSemantic(lexer.NewWithBytes(data, lexer.WithElideSeparator(false)))
		posOracle, pErr := drainVerbatim(lexer.NewVerbatimWithBytes(data))
		if vErr != "" || pErr != "" {
			continue
		}
		require.Equalf(t, len(valOracle), len(posOracle), "oracle length mismatch on %s", name)

		// build the expected unified stream: L's kind+value, VL's blanks+pos
		want := make([]vtok, len(valOracle))
		for i := range valOracle {
			want[i] = vtok{
				kind:   valOracle[i].kind,
				value:  valOracle[i].value,
				blanks: posOracle[i].blanks,
				line:   posOracle[i].line,
				col:    posOracle[i].col,
			}
		}

		// drive lab's VL through the native push iterator
		var got []vtok
		labVL := lab.NewVerbatimWithBytes(data)
		for tk := range labVL.Tokens() {
			got = append(got, vtok{
				kind:   tk.Kind(),
				value:  string(tk.Value()),
				blanks: string(tk.Blanks()),
				line:   tk.Line(),
				col:    tk.Col(),
			})
		}
		require.Emptyf(t, errStr(labVL.Err()), "lab VL push errored on valid %s", name)
		require.Equalf(t, want, got, "VL push stream mismatch on valid %s", name)
		checked++
	}
	t.Logf("verbatim push: matched the unified-contract oracle on %d valid fixtures", checked)
}
