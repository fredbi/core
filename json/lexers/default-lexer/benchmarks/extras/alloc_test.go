//go:build profile

package lexer

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime/trace"
	"testing"

	"github.com/pkg/profile"
	"github.com/stretchr/testify/require"
)

func TestProfile(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join(currentDir(), "fixtures", "example.json"))
	require.NoError(t, err)
	rdr := bytes.NewReader(fixture)

	p := profile.Start(profile.MemProfile, profile.MemProfileRate(1), profile.ProfilePath("."))
	for i := 0; i < 10000; i++ {
		lex := BorrowLexer(rdr, WithKeepBlanks(false), WithMaxValueBytes(1000))
		measureIt := measureIt(lex)
		err = measureIt(t)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			t.FailNow()
		}
		RedeemLexer(lex)
		rdr.Seek(0, io.SeekStart)
	}
	p.Stop()
}

func TestHeap(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join(currentDir(), "fixtures", "example.json"))
	require.NoError(t, err)
	dump, err := os.Create("trace.out")
	require.NoError(t, err)
	t.Cleanup(func() { _ = dump.Close() })
	rdr := bytes.NewReader(fixture)
	trace.Start(dump)
	defer trace.Stop()

	for i := 0; i < 10000; i++ {
		lex := BorrowLexer(rdr, WithKeepBlanks(false))
		measureIt := measureIt(lex)
		err = measureIt(t)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			t.FailNow()
		}
		RedeemLexer(lex)
		rdr.Seek(0, io.SeekStart)
	}

	// debug.WriteHeapDump(dump.Fd())
}

func BenchmarkLexer(b *testing.B) {
	fixture, err := os.ReadFile(filepath.Join(currentDir(), "fixtures", "example.json"))
	require.NoError(b, err)
	rdr := bytes.NewReader(fixture)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		lex := BorrowLexer(rdr)
		measureIt := measureIt(lex)
		err = measureIt(b)
		if err != nil {
			b.Errorf("unexpected error: %v", err)
			b.FailNow()
		}
		RedeemLexer(lex)
		rdr.Seek(0, io.SeekStart)
	}
}

func measureIt(lex *L) func(testing.TB) error {
	return func(_ testing.TB) error {
		var (
			i     int
			token Token
			value []byte
		)

		// shouldn't report any allocs, after amortization
		for ; !token.IsEOF(); i++ {
			token = lex.NextToken()
			if !lex.Ok() {
				return lex.Err()
			}
			// keep the token unoptimized
			value = token.Value
		}

		if len(value) > 0 {
			return errors.New("unexpected non-empty EOF value")
		}

		return nil
	}
}
