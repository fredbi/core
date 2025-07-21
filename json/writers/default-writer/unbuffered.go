package writer

import (
	"io"
	"runtime"

	"github.com/fredbi/core/json/writers"
)

var (
	_ writers.StoreWriter    = &Unbuffered{}
	_ writers.JSONWriter     = &Unbuffered{}
	_ writers.TokenWriter    = &Unbuffered{}
	_ writers.VerbatimWriter = &Unbuffered{}
)

// Unbuffered JSON writer.
// [Unbuffered] implements
// [writers.StoreWriter], [writers.JSONWriter], [writers.TokenWriter] and [writers.VerbatimWriter].
//
// It knows how to render JSON tokens and JSON values to an underlying [io.Writer].
//
// All writes are passed straight-through, and no flushing is required.
//
// Strings are escaped with default JSON escaping rule for tabs, new lines, line feeds, backslashes and double quotes.
//
// There is no attempt to do anything special regarding empty or null values:
//
//   - an undefined value (or nil data) is not rendered
//   - a null value is necessarily defined explicitly and is rendered with the "null" token
//
// # Concurrency
//
// [Unbuffered] is not intended for concurrent use.
type Unbuffered struct {
	unbuffered
	commonWriter[*unbuffered]
}

type unbuffered struct {
	baseWriter
	*unbufferedOptions
}

// NewUnbuffered JSON writer that copies JSON to [io.Writer] w, without buffering.
func NewUnbuffered(w io.Writer, opts ...UnbufferedOption) *Unbuffered {
	writer := &Unbuffered{
		unbuffered: unbuffered{
			baseWriter: baseWriter{
				w: w,
			},
			unbufferedOptions: unbufferedOptionsWithDefaults(
				opts,
			), // always borrow options from the pool
		},
	}
	writer.jw = &writer.unbuffered

	// when using New, borrowed inner resources must be relinquished when the gc claims the writer.
	runtime.AddCleanup(writer, func(o *unbufferedOptions) {
		if o != nil {
			o.redeem()
			poolOfUnbufferedOptions.Redeem(o)
		}
	}, writer.unbufferedOptions)

	return writer
}

func (w *Unbuffered) Reset() {
	w.baseWriter.Reset()
	if w.unbufferedOptions != nil {
		w.unbufferedOptions.Reset()
	}
}

func (w *unbuffered) redeem() {
	if w.unbufferedOptions != nil {
		w.unbufferedOptions.redeem()

		poolOfUnbufferedOptions.Redeem(w.unbufferedOptions)
		w.unbufferedOptions = nil
	}
}

func (w *unbuffered) writeSingleByte(c byte) {
	if w.err != nil {
		return
	}

	if bytesWriter, ok := w.w.(io.ByteWriter); ok {
		w.err = bytesWriter.WriteByte(c)
		if w.err == nil {
			w.inc(1)
		}

		return
	}

	n, err := w.w.Write([]byte{c})
	w.err = err
	w.inc(n)
}

func (w *unbuffered) writeBinary(data []byte) {
	n, err := w.w.Write(data)
	w.inc(n)
	w.err = err
}

func (w *unbuffered) writeEscaped(data []byte) []byte {
	writeEscaped(w, data)

	return nil
}
