//nolint:gochecknoglobals  // pools are globals
package writer

import (
	"io"

	"github.com/fredbi/core/swag/pools"
)

const (
	defaultCapacityForNumbers = 20
	defaultCapacityForReaders = 4096
	defaultCapacityForEscaped = 512
)

var (
	poolOfUnbuffered        = pools.New[Unbuffered]()
	poolOfUnbufferedOptions = pools.New[unbufferedOptions]()
	poolOfBuffered          = pools.New[Buffered]()
	poolOfBuffered2         = pools.New[Buffered2]()
	poolOfBufferedOptions   = pools.New[bufferedOptions]()
	poolOfIndented          = pools.New[Indented]()
	poolOfIndentedOptions   = pools.New[indentedOptions]()

	poolOfNumberBuffers = pools.NewPoolSlice[byte](
		pools.WithMinimumCapacity(defaultCapacityForNumbers),
	)
	poolOfReadBuffers = pools.NewPoolSlice[byte](
		pools.WithLength(defaultCapacityForReaders),
	)
	poolOfEscapedBuffers = pools.NewPoolSlice[byte]( // TODO: improve that one
		pools.WithMinimumCapacity(defaultCapacityForEscaped),
	)

	poolOfBuffers = pools.NewPoolSlice[byte]()
)

// BorrowUnbuffered recycles an [Unbuffered] writer from the global pool.
//
// [BorrowUnbuffered] is equivalent to [NewUnbuffered], but may save the allocation of new resources if
// they are readily available in the pool.
//
// The caller is responsible for calling [RedeemUnbuffered] after the work is done, and relinquish resources to the pool.
func BorrowUnbuffered(writer io.Writer, opts ...UnbufferedOption) *Unbuffered {
	w := poolOfUnbuffered.Borrow()
	w.w = writer
	w.unbufferedOptions = unbufferedOptionsWithDefaults(opts)

	return w
}

// RedeemUnbuffered relinquishes a borrowed [Unbuffered] writer back to the global pool.
//
// Inner resources are relinquished by this call.
func RedeemUnbuffered(w *Unbuffered) {
	w.redeem() // redeem inner resources
	poolOfUnbuffered.Redeem(w)
}

func BorrowBuffered(writer io.Writer, opts ...BufferedOption) *Buffered {
	w := poolOfBuffered.Borrow()
	w.w = writer
	w.bufferedOptions = bufferedOptionsWithDefaults(opts)

	return w
}

// RedeemBuffered relinquishes a borrowed [Buffered] writer back to the global pool.
//
// Inner resources are relinquished by this call.
func RedeemBuffered(w *Buffered) {
	w.redeem() // redeem inner resources
	poolOfBuffered.Redeem(w)
}

func BorrowBuffered2(writer io.Writer, opts ...BufferedOption) *Buffered2 {
	w := poolOfBuffered2.Borrow()
	w.w = writer
	w.bufferedOptions = bufferedOptionsWithDefaults(opts)
	w.jw = &w.buffered2

	return w
}

func RedeemBuffered2(w *Buffered2) {
	w.redeem() // redeem inner resources
	poolOfBuffered2.Redeem(w)
}

func BorrowIndented(writer io.Writer, opts ...IndentedOption) *Indented {
	w := poolOfIndented.Borrow()
	w.indentedOptions = indentedOptionsWithDefaults(opts)

	if w.Buffered2 == nil {
		// this is a new Indented: we need to borrow the inner Buffered2
		w.Buffered2 = BorrowBuffered2(writer, w.applyBufferedOptions...)
		w.redeemBuffered2 = w.Buffered2 // mark for redemption later on

		return w
	}

	// this is a recycled Indented: we already have a Buffered2, we just need to Reset it
	w.Buffered2.Reset()

	// now ensure that the recycled Buffered2 got the correct options
	if w.bufferedOptions == nil {
		w.bufferedOptions = bufferedOptionsWithDefaults(w.applyBufferedOptions)
	} else {
		w.bufferedOptions.updateOptions(w.applyBufferedOptions)
	}

	// set the new underlying writer for this recycled instance
	w.w = writer

	return w
}

func RedeemIndented(w *Indented) {
	w.redeem() // redeem inner resources
	poolOfIndented.Redeem(w)
}
