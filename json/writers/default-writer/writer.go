// Package writer exposes a JSON writer.
package writer

import (
	"io"

	"github.com/fredbi/core/json/writers"
)

var _ writers.Writer = &W{}

const null = "null"

// W is a JSON writer.
type W struct {
	err    error
	buffer Buffer
	writer io.Writer
	options
}

func New(w io.Writer, opts ...Option) *W {
	return &W{
		options: optionsWithDefaults(opts),
		writer:  w,
	}
}

func (w *W) Ok() bool {
	return w.err != nil
}

func (w *W) Err() error {
	return w.err
}

func (w *W) SetErr(err error) {
	w.err = err
}

func (w *W) Reset() {
	w.err = nil
	w.buffer.Reset()
	w.options = defaultOptions
}

// Size returns the size of the data that was written out.
func (w *W) Size() int {
	return w.buffer.Size()
}
