package writer

import (
	"io"

	"github.com/fredbi/core/json/writers/default-writer/internal/bufio"
)

// Option configures the [Writer].
type Option func(*options)

// WithBuffered will use the [bufio.ChunkedBuffer] implementation of [Buffer] when true.
//
// By default (enabled is false), the [bufio.Unbuffered] implementation is used.
//
// This option is inoperant when providing a buffer directly with [WithBuffer].
func WithBuffered(enabled bool) Option {
	return func(o *options) {
		o.isBuffered = enabled
	}
}

// WithEscapeHTML enables extra escaping
//
// This option is inoperant when providing a buffer directly with [WithBuffer].
func WithEscapeHTML(enabled bool) Option {
	return func(o *options) {
		if enabled {
			o.escape = bufio.EscapeHTML
			return
		}

		o.escape = bufio.NoEscapeHTML
	}
}

// WithBuffer replaces the buffer implementation by a custom one.
//
// The caller is responsible for the allocation (and possibly pool management) of the passed [Buffer].
//
// This disables options that apply to the [Buffer] provided by default.
func WithBuffer(buffer Buffer) Option {
	return func(o *options) {
		o.buffer = buffer
	}
}

type options struct {
	w            io.Writer
	buffer       Buffer            // inner [Buffer], may be overridden
	isBuffered   bool              // when no buffer is provided, enabled [bufio.ChunkedBuffer]. If false [bufio.Unbuffered] is used.
	escape       bufio.EscapeFlags //
	redeemBuffer func()
}

// Reset options to default and reset the inner [Buffer].
func (o *options) Reset() {
	o.escape = bufio.NoEscapeHTML
}

func (o *options) borrowBuffer() {
	if o.isBuffered {
		o.buffer, o.redeemBuffer = poolOfChunkedBuffers.BorrowWithRedeem()
		// TODO: Set flags
		return
	}

	o.buffer, o.redeemBuffer = poolOfUnbuffered.BorrowWithRedeem()
	settable := o.buffer.(interface {
		Set(io.Writer, bufio.EscapeFlags)
	})
	settable.Set(o.w, o.escape)
}

// redeem the options and its internal buffer
func (o *options) redeem() {
	if o.redeemBuffer != nil {
		o.redeemBuffer()
		o.redeemBuffer = nil
	}

	poolOfOptions.Redeem(o)
}

func optionsWithDefaults(w io.Writer, opts []Option) *options {
	o := poolOfOptions.Borrow()
	o.w = w

	for _, apply := range opts {
		apply(o)
	}

	o.borrowBuffer()

	return o
}
