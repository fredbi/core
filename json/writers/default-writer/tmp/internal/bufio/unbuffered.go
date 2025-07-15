package bufio

import (
	"errors"
	"io"
)

// Unbuffered implements straight-through io's to the underlying [io.Writer],
// with support for JSON escaping.
//
// [Unbuffered] won't allocate any extra-memory.
//
// If the underlying [io.Writer] implement any of the following methods, they will we used.
//
// Otherwise, these methods will be emulated using [io.Writer.Write].
//
//   - WriteSingleByte(byte)
//   - WriteByte(byte) error
//   - WriteText([]byte)
//   - WriteText([]byte) (int,error)
//   - WriteBinary([]byte)
//   - WriteBinary([]byte) (int,error)
//   - Reset
//
// For example, when providing a [bytes.Buffer] as a writer, [Unbuffered] will call directly
// [bytes.Buffer.WriteByte] and [bytes.Buffer.Reset].
type Unbuffered struct {
	w           io.Writer
	size        int64
	writeByte   func(byte) error
	writeText   func([]byte) (int, error)
	writeBinary func([]byte) (int, error)
	reset       func()
	err         error
	escape      EscapeFlags
}

// NewUnbuffered builds a new [Unbuffered] object on top of an [io.Writer].
func NewUnbuffered(w io.Writer, flags EscapeFlags) *Unbuffered {
	u := &Unbuffered{
		w:      w,
		escape: flags,
	}

	u.selectMethods()

	return u
}

func (b *Unbuffered) Set(w io.Writer, flags EscapeFlags) {
	b.w = w
	b.escape = flags

	b.selectMethods()
}

func (b *Unbuffered) WriteSingleByte(c byte) {
	if b.err != nil {
		return
	}

	b.err = b.writeByte(c)
	b.size++
}

func (b *Unbuffered) WriteText(data []byte) {
	if b.err != nil {
		return
	}

	n, err := writeEscapedBytes(data, b.escape, b.writeByte, b.writeText)
	b.err = err
	b.size += int64(n)
}

func (b *Unbuffered) WriteBinary(data []byte) {
	if b.err != nil {
		return
	}

	n, err := b.writeBinary(data)
	b.err = err
	b.size += int64(n)
}

func (b *Unbuffered) WriteString(data string) {
	if b.err != nil {
		return
	}

	n, err := writeEscapedBytes(data, b.escape, b.writeByte, b.writeText)
	b.err = err
	b.size += int64(n)
}

func (b *Unbuffered) Reset() {
	b.err = nil
	if b.w != nil {
		b.selectMethods()

		if b.reset != nil {
			b.reset()
			return
		}
	}

	b.w = nil
}

func (b *Unbuffered) Err() error {
	return b.err
}

func (b *Unbuffered) Ok() bool {
	return b.err == nil
}

func (b *Unbuffered) Size() int64 {
	return b.size
}

func (b *Unbuffered) WriteTextFrom(r io.Reader) {
	if b.err != nil {
		return
	}

	pooledBuf, redeem := copyBuffers.BorrowWithRedeem()
	defer redeem()
	pooledBuf.Append(copyPad...)
	buf := pooledBuf.Slice()

	for {
		nr, er := r.Read(buf)
		if nr > 0 {
			nw, ew := writeEscapedBytes(buf[0:nr], b.escape, b.writeByte, b.writeText)
			if nw < 0 || ew != nil { // after escaping, we may write more, but never less than read
				if ew == nil {
					ew = io.ErrShortWrite
				}
				b.err = ew

				break
			}
			b.size += int64(nw)
		}
		if er != nil {
			if !errors.Is(er, io.EOF) {
				b.err = er
			}

			break
		}
	}
}

func (b *Unbuffered) WriteBinaryFrom(r io.Reader) {
	if b.err != nil {
		return
	}

	buf, redeem := copyBuffers.BorrowWithRedeem()
	defer redeem()
	buf.Append(copyPad...)
	n, err := io.CopyBuffer(b.w, r, buf.Slice())
	b.err = err
	b.size += n
}

func (b *Unbuffered) selectMethods() {
	w := b.w

	switch v := w.(type) {
	case interface{ WriteSingleByte(byte) }:
		b.writeByte = func(b byte) error {
			v.WriteSingleByte(b)

			return nil
		}
	case interface{ WriteByte(byte) error }:
		b.writeByte = v.WriteByte
	default:
		b.writeByte = func(b byte) error {
			var single [1]byte
			single[0] = b
			_, err := w.Write(single[:])

			return err
		}
	}

	switch v := w.(type) {
	case interface{ WriteText([]byte) }:
		b.writeText = func(data []byte) (int, error) {
			v.WriteText(data)
			return len(data), nil
		}
	case interface{ WriteText([]byte) (int, error) }:
		b.writeText = v.WriteText
	default:
		b.writeText = w.Write
	}

	switch v := w.(type) {
	case interface{ WriteBinary([]byte) }:
		b.writeBinary = func(data []byte) (int, error) {
			v.WriteBinary(data)
			return len(data), nil
		}
	case interface{ WriteBinary([]byte) (int, error) }:
		b.writeBinary = v.WriteBinary
	default:
		b.writeBinary = b.writeText
	}

	if resettable, ok := w.(interface{ Reset() }); ok {
		b.reset = resettable.Reset
	} else {
		b.reset = nil
	}
}
