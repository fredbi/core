package light

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	store "github.com/fredbi/core/json/stores/default-store"
	writer "github.com/fredbi/core/json/writers/default-writer"
)

var errBoom = errors.New("boom")

// failingWriter is an io.Writer that fails after letting through the first n bytes, and counts how
// many times Write was actually called on it.
type failingWriter struct {
	remaining int
	calls     int
}

func (f *failingWriter) Write(p []byte) (int, error) {
	f.calls++
	if f.remaining <= 0 {
		return 0, errBoom
	}
	if len(p) > f.remaining {
		n := f.remaining
		f.remaining = 0

		return n, errBoom
	}
	f.remaining -= len(p)

	return len(p), nil
}

func buildNested(s *store.Store) Node {
	scalar := func(v string) Node { return NewBuilder(s).StringValue(v).Node() }

	return NewBuilder(s).Object().
		AppendKey("a", scalar("A")).
		AppendKey("b", NewBuilder(s).Array().
			AppendElems(scalar("x"), scalar("y"), scalar("z")).Node()).
		AppendKey("c", scalar("C")).
		Node()
}

// TestEncodeWriterError covers E2: a mid-stream writer failure surfaces as an error (no panic) and the
// encoder stops feeding the writer instead of walking the whole structure.
func TestEncodeWriterError(t *testing.T) {
	s := store.New()
	n := buildNested(s)

	fw := &failingWriter{remaining: 1} // accept 1 byte, then fail every subsequent Write
	jw := writer.NewUnbuffered(fw)
	ctx := &ParentContext{S: s, W: jw}

	require.NotPanics(t, func() { n.Encode(ctx) })

	require.Error(t, ctx.W.Err())
	require.NotNil(t, ctx.C)
	require.ErrorContains(t, ctx.C.Err, "node error")

	// the underlying writer should have been hit only a handful of times: once it errors, the writer
	// goes sticky and the encoder bails — it must not keep issuing one Write per remaining node.
	assert.Lessf(t, fw.calls, 5, "encoder kept writing after the error (%d Write calls)", fw.calls)
}

// TestEncodeNilWriter and TestEncodeNilStore cover E3: missing context dependencies must not panic.
func TestEncodeNilWriter(t *testing.T) {
	s := store.New()
	n := buildNested(s)

	require.NotPanics(t, func() {
		n.Encode(&ParentContext{S: s}) // W is nil
	})
}

func TestEncodeNilStore(t *testing.T) {
	s := store.New()
	n := NewBuilder(s).StringValue("scalar-needs-a-store").Node()

	var buf testWriterSink
	jw := writer.NewUnbuffered(&buf)
	ctx := &ParentContext{W: jw} // S is nil, but the scalar needs it

	require.NotPanics(t, func() { n.Encode(ctx) })
	require.Error(t, ctx.W.Err())
	require.NotNil(t, ctx.C)
	assert.ErrorContains(t, ctx.C.Err, "nil store")
}

// testWriterSink is a trivial discard sink.
type testWriterSink struct{}

func (testWriterSink) Write(p []byte) (int, error) { return len(p), nil }
