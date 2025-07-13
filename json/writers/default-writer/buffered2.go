package writer

import (
	"io"
	"runtime"

	"github.com/fredbi/core/json/writers"
)

var (
	_ writers.StoreWriter    = &Buffered2{}
	_ writers.JSONWriter     = &Buffered2{}
	_ writers.TokenWriter    = &Buffered2{}
	_ writers.VerbatimWriter = &Buffered2{}
)

// Buffered2 JSON writer.
type Buffered2 struct {
	buffered2
	commonWriter[*buffered2]
}

type buffered2 struct {
	baseWriter
	*bufferedOptions
}

func NewBuffered2(w io.Writer, opts ...BufferedOption) *Buffered2 {
	writer := &Buffered2{
		buffered2: buffered2{
			baseWriter: baseWriter{
				w: w,
			},
			bufferedOptions: bufferedOptionsWithDefaults(opts), // always borrow options from the pool
		},
	}
	writer.commonWriter.jw = &writer.buffered2

	// when using New, borrowed inner resources must be relinquished when the gc claims the writer.
	runtime.AddCleanup(writer, func(o *bufferedOptions) {
		if o != nil {
			o.redeem()

			poolOfBufferedOptions.Redeem(o)
		}
	}, writer.bufferedOptions)

	return writer
}

func (w *Buffered2) Reset() {
	w.baseWriter.Reset()
	if w.bufferedOptions != nil {
		w.bufferedOptions.Reset()
	}
}

// Flush the internal buffer of the [Buffered2] writer to the underlying [io.Writer].
func (w *Buffered2) Flush() error {
	if w.err != nil {
		return w.err
	}

	w.flush()

	return w.err
}

func (w *buffered2) flush() {
	n, err := w.w.Write(w.buffer)
	w.inc(n)
	w.err = err
	w.buffer = w.buffer[:0]
}

// redeem inner resources
func (w *buffered2) redeem() {
	if w.bufferedOptions != nil {
		w.bufferedOptions.redeem()

		poolOfBufferedOptions.Redeem(w.bufferedOptions)
		w.bufferedOptions = nil
	}
}

func (w *buffered2) writeSingleByte(c byte) {
	if len(w.buffer) == cap(w.buffer) {
		w.flush()
	}

	if w.err != nil {
		return
	}

	w.buffer = append(w.buffer, c)
}

func (w *buffered2) writeBinary(data []byte) {
	var offset int

	for offset < len(data) {
		if len(w.buffer) == cap(w.buffer) {
			w.flush()
			if w.err != nil {
				return
			}
		}

		chunkSize := min(len(data[offset:]), cap(w.buffer)-len(w.buffer))
		w.buffer = append(w.buffer, data[offset:offset+chunkSize]...) // copy data to the buffer

		offset += chunkSize
	}
}
