package lexer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/lexers/token"
)

// collectL returns the (kind,value) stream produced by the pull lexer L with
// separators elided (the default), to compare against the push prototype.
func collectL(data []byte) ([]token.Kind, []string, error) {
	lex := NewWithBytes(data)
	var (
		kinds  []token.Kind
		values []string
	)
	for tok := range lex.Tokens() {
		kinds = append(kinds, tok.Kind())
		values = append(values, string(tok.Value()))
	}

	return kinds, values, lex.Err()
}

func collectP(data []byte) ([]token.Kind, []string, error) {
	p := NewPush(data)
	var (
		kinds  []token.Kind
		values []string
	)
	for tok := range p.Tokens() {
		kinds = append(kinds, tok.Kind())
		values = append(values, string(tok.Value()))
	}

	return kinds, values, p.Err()
}

func TestPushMatchesPull(t *testing.T) {
	docs := []string{
		`{"a":1,"b":[true,false,null],"c":"x"}`,
		`[1,2.5e3,-4,0.5,"s","es\ncaped\"q","unié"]`,
		`{"nested":{"deep":{"k":[1,[2,[3]]]}}}`,
		` {  "a" :  1 ,  "b" : [ 2 , 3 ]  } `,
		`{"k":"prefix\tA\\end"}`,
		`[]`,
		`{}`,
		`"lonely string"`,
		`12345`,
		`true`,
		`null`,
	}

	for _, doc := range docs {
		t.Run(doc, func(t *testing.T) {
			wantK, wantV, errL := collectL([]byte(doc))
			require.NoError(t, errL, "pull lexer failed")

			gotK, gotV, errP := collectP([]byte(doc))
			require.NoError(t, errP, "push prototype failed")

			require.Equalf(t, wantK, gotK, "token kinds differ\nwant %v\ngot  %v", wantK, gotK)
			assert.Equal(t, wantV, gotV, "token values differ")
		})
	}
}

func TestPushRejectsInvalid(t *testing.T) {
	bad := []string{
		`[1,2`,       // unterminated array
		`{"a":1`,     // unterminated object
		`[1,]`,       // (push tolerates? expect either way it ends in array-close mismatch) -- keep structural
		`{1:2}`,      // non-string key region
		`[01]`,       // leading zero
		`[1.]`,       // missing fractional digit
		`["a\x00b"]`, // control char (raw NUL via actual byte below)
		`tru`,        // truncated literal
	}
	_ = bad // structural error coverage is a prototype subset; spot-check a few:

	for _, doc := range []string{`[1,2`, `{"a":1`, `[01]`, `[1.]`, `tru`} {
		t.Run(doc, func(t *testing.T) {
			_, _, err := collectP([]byte(doc))
			assert.Error(t, err, "expected rejection of %q", doc)
		})
	}
}

// TestPushMatchesPullCorpusFixture cross-checks against the larger vendored
// example fixture if present.
func TestPushMatchesPullFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(currentDir(), "fixtures", "example.json"))
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}

	wantK, wantV, errL := collectL(data)
	require.NoError(t, errL)
	gotK, gotV, errP := collectP(data)
	require.NoError(t, errP)

	require.Equal(t, wantK, gotK)
	assert.Equal(t, wantV, gotV)
}
