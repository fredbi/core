package writer

import (
	"encoding"
	"errors"
	"fmt"
	"io"
	"runtime"

	"github.com/fredbi/core/json/writers"
)

var (
	_ writers.StoreWriter = &W{}
	_ writers.JSONWriter  = &W{}
	_ writers.TokenWriter = &W{}
)

// W is a JSON writer that implements [writers.Writer].
//
// It knows how to render JSON tokens and JSON values.
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
// [W] is not intended for concurrent use.
type W struct {
	err error
	*options
}

// New JSON writer that copies JSON to [io.Writer] w.
//
// The [W] writer may be suitable for 2 different use-cases:
//
//   - seamless unbuffered writing to an underlying writer (see [bufio.Unbuffered])
//   - buffered writing to a chunked buffer (use [WithBuffer] (true): see [bufio.ChunkedBuffer])
//
// When used in buffered mode, [W] supports [encoding.TextAppender] and [writers.Flusher], so you
// may either copy the buffer to a slice of bytes using [W.AppendText] or use [W.Flush] to output to the
// underlying writer.
func New(w io.Writer, opts ...Option) *W {
	writer := &W{
		options: optionsWithDefaults(w, opts),
	}

	// when using New, borrowed inner resources must be relinquished when the gc claims the writer.
	runtime.AddCleanup(&w, func(o *options) {
		o.redeem()
	}, writer.options)

	return writer
}

// Ok tells the status of the writer.
func (w *W) Ok() bool {
	return w.err == nil
}

// Err yields the current error status of the writer.
func (w *W) Err() error {
	if w.err != nil {
		return errors.Join(w.err, ErrDefaultWriter)
	}

	return nil
}

// SetErr injects an error into the writer.
//
// Whenever an error is injected, [W] short-circuits all operations.
func (w *W) SetErr(err error) {
	w.err = err
}

// Reset the writer, which may be thus recycled.
func (w *W) Reset() {
	w.err = nil
	if w.options != nil {
		w.options.Reset()
	}
}

// Size returns the bytes that have been written so far.
func (w *W) Size() int64 {
	return w.buffer.Size()
}

// AppendText appends the textual representation of itself to the end of b
// (allocating a larger slice if necessary) and returns the updated slice.
//
// This method yields an error when the writer goes unbuffered.
func (w *W) AppendText(b []byte) ([]byte, error) {
	if appender, ok := w.buffer.(encoding.TextAppender); ok {
		return appender.AppendText(b)
	}

	return nil, fmt.Errorf(
		"AppendText is called, but the selected buffer is not compatible: %T: %w",
		w.buffer, ErrUnsupportedInterface,
	)
}

// Flush written output to the configured writer, if the implementation of the internal
// [Buffer] requires some flush to be carried out.
//
// This method yields an error when the writer goes unbuffered.
func (w *W) Flush() error {
	// TODO: separate object type
	return nil
	/*
		if writerTo, ok := w.buffer.(io.WriterTo); ok {
			_, err := writerTo.WriteTo(w.w)

			return fmt.Errorf("error flushing writer: %w: %w", err, ErrDefaultWriter)
		}

		return fmt.Errorf(
			"Flush is called, but the selected buffer is not compatible: %T: %w",
			w.buffer, ErrUnsupportedInterface,
		)
	*/
}
