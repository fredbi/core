package lexer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/lexers/token"
	"github.com/stretchr/testify/require"
)

func TestVerbatimExample(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join(currentDir(), "fixtures", "example.json"))
	require.NoError(t, err)

	t.Run("with reader", testFixtureVerbatim(func() *VL {
		rdr := bytes.NewBuffer(fixture)
		return NewVerbatim(rdr, WithBufferSize(50))
	}))

	t.Run("with buffer", testFixtureVerbatim(func() *VL {
		return NewVerbatimWithBytes(fixture)
	}))
}

func TestVerbatim(t *testing.T) {
	rdr := bytes.NewBufferString(`  {   "a":  "1",    "b":    123  , "c"  :  false }   `)
	lex := NewVerbatim(rdr, WithBufferSize(50))

	var (
		i   int
		tok token.VT
	)

	t.Run("should preseve blank space in tokens", func(t *testing.T) {
		for ; !tok.IsEOF(); i++ {
			tok = lex.NextToken()
			require.NoErrorf(t, lex.Err(), errorDetails(t, lex.Err(), lex))
			require.True(t, lex.Ok())
			t.Logf("-> %v", tok)
			t.Logf("-> %q", tok.Blanks())
			blanks := string(tok.Blanks())
			require.Empty(t, strings.TrimSpace(blanks))
			switch i {
			case 0: // {
				require.Len(t, blanks, 2)
			case 1: // "a"
				require.Len(t, blanks, 3)
			case 2: // :
				require.Empty(t, blanks)
			case 3: // "1"
				require.Len(t, blanks, 2)
			case 4: // ,
				require.Empty(t, blanks)
			case 5: // "b"
				require.Len(t, blanks, 4)
			case 6: // :
				require.Empty(t, blanks)
			case 7: // 123
				require.Len(t, blanks, 4)
			case 8: // ,
				require.Len(t, blanks, 2)
			case 9: // "c"
				require.Len(t, blanks, 1)
			case 10: // :
				require.Len(t, blanks, 2)
			case 11: // false
				require.Len(t, blanks, 2)
			case 12: // }
				require.Len(t, blanks, 1)
			case 13: // EOF
				t.Run("should keep trailing blanks before EOF", func(t *testing.T) {
					// EOF
					blanks := string(tok.Blanks())
					require.Empty(t, strings.TrimSpace(blanks))
					require.Len(t, blanks, 3)
				})
			default:
				t.Errorf("unexpected number of tokens: %d", i)
				t.FailNow()
			}
		}
	})

	t.Logf("last -> %v", tok)
	t.Logf("last -> %q", tok.Blanks())
}

func errorDetails(t testing.TB, err error, reporter interface{ ErrInContext() *codes.ErrContext }) string {
	t.Helper()

	if err == nil {
		return ""
	}

	ctx := reporter.ErrInContext()
	require.NotNil(t, ctx)
	return fmt.Sprintf("unexpected error: %#v\n%s",
		ctx, ctx.Pretty(50),
	)
}

func testFixtureVerbatim(newLexer func() *VL) func(*testing.T) {
	return func(t *testing.T) {
		lex := newLexer()

		var (
			i   int
			tok token.VT
		)

		for ; !tok.IsEOF(); i++ {
			tok = lex.NextToken()
			require.NoErrorf(t, lex.Err(), errorDetails(t, lex.Err(), lex))
			require.True(t, lex.Ok())
			t.Logf("-> %v", tok)
		}

		t.Logf("split %d tokens", i)
	}
}
