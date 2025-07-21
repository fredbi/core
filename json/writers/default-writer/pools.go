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

	poolOfBuffered        = pools.New[Buffered]()
	poolOfBufferedOptions = pools.New[bufferedOptions]()

	poolOfIndented        = pools.New[Indented]()
	poolOfIndentedOptions = pools.New[indentedOptions]()

	poolOfYAML        = pools.New[YAML]()
	poolOfYAMLOptions = pools.New[yamlOptions]()

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
	if w.unbufferedOptions != nil {
		poolOfUnbufferedOptions.Redeem(w.unbufferedOptions)
	}
	w.unbufferedOptions = unbufferedOptionsWithDefaults(opts)
	w.jw = &w.unbuffered

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
	if w.bufferedOptions != nil {
		poolOfBufferedOptions.Redeem(w.bufferedOptions)
	}
	w.bufferedOptions = bufferedOptionsWithDefaults(opts)
	w.jw = &w.buffered

	return w
}

func RedeemBuffered(w *Buffered) {
	w.redeem() // redeem inner resources
	poolOfBuffered.Redeem(w)
}

func BorrowIndented(writer io.Writer, opts ...IndentedOption) *Indented {
	w := poolOfIndented.Borrow()
	if w.indentedOptions != nil {
		poolOfIndentedOptions.Redeem(w.indentedOptions)
	}
	w.indentedOptions = indentedOptionsWithDefaults(opts)

	if w.Buffered == nil {
		// this is a new Indented: we need to borrow the inner Buffered
		w.Buffered = BorrowBuffered(writer, w.applyBufferedOptions...)
		w.redeemBuffered = w.Buffered // mark for redemption later on

		return w
	}

	// this is a recycled Indented: we already have a Buffered, we just need to Reset it
	w.Buffered.Reset()

	// now ensure that the recycled Buffered got the correct options
	if w.bufferedOptions == nil {
		w.bufferedOptions = bufferedOptionsWithDefaults(w.applyBufferedOptions)
	} else {
		w.updateOptions(w.applyBufferedOptions)
	}

	// set the new underlying writer for this recycled instance
	w.w = writer

	return w
}

func RedeemIndented(w *Indented) {
	w.redeem() // redeem inner resources
	poolOfIndented.Redeem(w)
}

func BorrowYAML(writer io.Writer, opts ...YAMLOption) *YAML {
	w := poolOfYAML.Borrow()
	if w.yamlOptions != nil {
		poolOfYAMLOptions.Redeem(w.yamlOptions)
	}
	w.yamlOptions = yamlOptionsWithDefaults(opts)

	if w.Buffered == nil {
		// this is a new Indented: we need to borrow the inner Buffered
		w.Buffered = BorrowBuffered(writer, w.applyBufferedOptions...)
		w.redeemBuffered = w.Buffered // mark for redemption later on

		return w
	}

	// this is a recycled Indented: we already have a Buffered, we just need to Reset it
	w.Buffered.Reset()

	// now ensure that the recycled Buffered got the correct options
	if w.bufferedOptions == nil {
		w.bufferedOptions = bufferedOptionsWithDefaults(w.applyBufferedOptions)
	} else {
		w.updateOptions(w.applyBufferedOptions)
	}

	// set the new underlying writer for this recycled instance
	w.w = writer

	return w
}

func RedeemYAML(w *YAML) {
	w.redeem() // redeem inner resources
	poolOfYAML.Redeem(w)
}
