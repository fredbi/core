package writer

import (
	"encoding"
	"errors"
	"fmt"
	"io"
	"math/big"
	"unicode/utf8"
)

type baseWriter struct {
	w       io.Writer
	written int64
	err     error
}

// Ok tells the status of the writer.
func (w *baseWriter) Ok() bool {
	return w.err == nil
}

// Err yields the current error status of the writer.
func (w baseWriter) Err() error {
	if w.err != nil {
		return errors.Join(w.err, ErrDefaultWriter)
	}

	return nil
}

// SetErr injects an error into the writer.
//
// Whenever an error is injected, [W] short-circuits all operations.
func (w *baseWriter) SetErr(err error) {
	w.err = err
}

// Reset the writer, which may be thus recycled.
func (w *baseWriter) Reset() {
	w.err = nil
	w.written = 0
}

// Size returns the bytes that have been written so far.
func (w *baseWriter) Size() int64 {
	return w.written
}

func (w *baseWriter) inc(n int) {
	w.written += int64(n)
}

type wrt interface {
	writeSingleByte(byte)
	writeBinary([]byte)
	writeText([]byte) []byte
	writeEscaped([]byte) []byte
	Err() error
	SetErr(error)
}

func stringCopy(w wrt, r io.Reader) {
	w.writeSingleByte(quote)
	if w.Err() != nil {
		return
	}

	var remainder []byte
	bufHolder, redeemReadBuffer := poolOfReadBuffers.BorrowWithRedeem()
	extraBufHolder, redeemExtraBuf := poolOfEscapedBuffers.BorrowWithSizeAndRedeem(bufHolder.Len() + utf8.UTFMax)
	escapedHolder, redeemEscaped := poolOfEscapedBuffers.BorrowWithSizeAndRedeem(bufHolder.Len())
	defer func() {
		redeemReadBuffer()
		redeemEscaped()
		redeemExtraBuf()
	}()

	buf := bufHolder.Slice()
	escapedBuffer := escapedHolder.Slice()
	extra := extraBufHolder.Slice()

	for {
		n, err := r.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			w.SetErr(err)

			return
		}

		if n > 0 {
			if len(extra) > 0 {
				// if the previous read reported an incomplete rune, the incomplete part is prepended to the input now
				extra = append(extra, buf[:n]...)
				escapedBuffer, remainder = escapedBytes(extra, escapedBuffer)
				extra = extra[:0]
			} else {
				escapedBuffer, remainder = escapedBytes(buf[:n], escapedBuffer)
			}

			w.writeBinary(escapedBuffer)
			if w.Err() != nil {
				return
			}

			if len(remainder) > 0 {
				if len(remainder) >= utf8.UTFMax {
					w.SetErr(fmt.Errorf("unexpected incomplete rune: %c: %w", remainder, ErrDefaultWriter))

					return
				}

				extra = extra[:0]
				extra = append(extra, remainder...)
			}
		}

		if n == 0 || (err != nil && errors.Is(err, io.EOF)) {
			if len(remainder) > 0 {
				w.SetErr(fmt.Errorf("unexpected incomplete rune at end of input: %c: %w", remainder, ErrDefaultWriter))

				return
			}

			break
		}
	}

	w.writeSingleByte(quote)
}

/*
func writeRawString(w wrt, input string) {
	stringBuffer, redeem := poolOfEscapedBuffers.BorrowWithSizeAndRedeem(len(input))
	defer redeem()
	data := stringBuffer.Slice()
	data = append(data, input...)

	w.writeBinary(data)
}
*/

func writeTextString(w wrt, input string) {
	stringBuffer, redeem := poolOfEscapedBuffers.BorrowWithSizeAndRedeem(len(input))
	defer redeem()
	data := stringBuffer.Slice()
	data = append(data, input...)

	writeText(w, data)
}

/*
func writeTextStringG[T wrt](w T, input string) {
	stringBuffer, redeem := poolOfEscapedBuffers.BorrowWithSizeAndRedeem(len(input))
	defer redeem()
	data := stringBuffer.Slice()
	data = append(data, input...)

	writeText(w, data)
}
*/

func writeTextRunes(w wrt, data []rune) {
	if w.Err() != nil || data == nil {
		return
	}
	holder, redeem := poolOfEscapedBuffers.BorrowWithSizeAndRedeem(len(data) * utf8.MaxRune)
	defer redeem()

	buf := holder.Slice()
	for _, r := range data {
		buf = utf8.AppendRune(buf, r)
	}

	writeText(w, buf)
}

/*
func writeTextRunesG[T wrt](w T, data []rune) {
	if w.Err() != nil || data == nil {
		return
	}
	holder, redeem := poolOfEscapedBuffers.BorrowWithSizeAndRedeem(len(data) * utf8.MaxRune)
	defer redeem()

	buf := holder.Slice()
	for _, r := range data {
		buf = utf8.AppendRune(buf, r)
	}

	writeText(w, buf)
}
*/

func writeEscaped(w wrt, data []byte) {
	if w.Err() != nil {
		return
	}

	var remainder []byte

	escapedHolder, redeemEscaped := poolOfEscapedBuffers.BorrowWithSizeAndRedeem(len(data)) // TODO: more elaborate pool
	escapedBuffer := escapedHolder.Slice()
	escapedBuffer, remainder = escapedBytes(data, escapedBuffer)
	w.writeBinary(escapedBuffer)
	redeemEscaped()
	if len(remainder) > 0 {
		w.SetErr(fmt.Errorf("incomplete rune at end of input: %c: %w", remainder, ErrDefaultWriter))
	}

}
func writeText(w wrt, data []byte) {
	w.writeSingleByte(quote)
	writeEscaped(w, data)
	w.writeSingleByte(quote)
}

/*
func writeTextG[T wrt](w T, data []byte) {
	w.writeSingleByte(quote)
	if w.Err() != nil {
		return
	}

	var remainder []byte

	escapedHolder, redeemEscaped := poolOfEscapedBuffers.BorrowWithSizeAndRedeem(len(data)) // TODO: more elaborate pool
	escapedBuffer := escapedHolder.Slice()
	escapedBuffer, remainder = escapedBytes(data, escapedBuffer)
	w.writeBinary(escapedBuffer)
	redeemEscaped()
	if len(remainder) > 0 {
		w.SetErr(fmt.Errorf("incomplete rune at end of input: %c: %w", remainder, ErrDefaultWriter))

		return
	}

	w.writeSingleByte(quote)
}
*/

func appendRaw(w wrt, n encoding.TextAppender) {
	buf, redeem := poolOfNumberBuffers.BorrowWithRedeem()
	defer redeem()
	b := buf.Slice()

	b, err := n.AppendText(b)
	if err != nil {
		w.SetErr(err)

		return
	}

	w.writeBinary(b)
}

/*
func appendRawG[T wrt](w T, n encoding.TextAppender) {
	buf, redeem := poolOfNumberBuffers.BorrowWithRedeem()
	defer redeem()
	b := buf.Slice()

	b, err := n.AppendText(b)
	if err != nil {
		w.SetErr(err)

		return
	}

	w.writeBinary(b)
}
*/

func appendFloat(w wrt, n *big.Float) {
	buf, redeem := poolOfNumberBuffers.BorrowWithRedeem()
	defer redeem()
	b := buf.Slice()

	b = n.Append(b, 'g', int(n.MinPrec()))
	w.writeBinary(b)
}

/*
func appendFloatG[T wrt](w T, n *big.Float) {
	buf, redeem := poolOfNumberBuffers.BorrowWithRedeem()
	defer redeem()
	b := buf.Slice()

	b = n.Append(b, 'g', int(n.MinPrec()))
	w.writeBinary(b)
}
*/
