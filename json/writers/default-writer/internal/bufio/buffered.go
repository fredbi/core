package bufio

import (
	"io"
)

// ChunkedBuffer is a buffer made of recyclable chunks.
//
// The difference between a [ChunkedBuffer] and a regular slice is that we are not constrained
// to maintain data as contiguous block of memory.
//
// The chunked buffer works with fixed-size chunks and therefore, no internal data moving is incurred
// like when a slice has to grow.
type ChunkedBuffer struct {
	Buf []byte // Buf is the current chunk for writing

	bufs      [][]byte // past allocated buffers
	redeemers []func()
	escape    EscapeFlags
}

func NewChunkedBuffer(flags EscapeFlags) *ChunkedBuffer {
	initBuffersOnce.Do(initializeBuffers)
	const sensibleChunks = 8
	return &ChunkedBuffer{
		escape:    flags,
		bufs:      make([][]byte, 0, sensibleChunks),
		redeemers: make([]func(), 0, sensibleChunks+1),
	}
}

// WriteSingleByte appends a single byte to buffer.
func (b *ChunkedBuffer) WriteSingleByte(c byte) {
	_ = b.writeByte(c)
}

// WriteText appends a byte slice to buffer, with escaping.
func (b *ChunkedBuffer) WriteText(data []byte) {
	_, _ = writeEscapedBytes(data, b.escape, b.writeByte, b.writeBinary)
}

// WriteString appends a string to buffer, with escaping.
func (b *ChunkedBuffer) WriteString(data string) {
	_, _ = writeEscapedBytes(data, b.escape, b.writeByte, b.writeBinary)
}

// WriteBinary appends a raw byte slice to buffer, without any escaping.
func (b *ChunkedBuffer) WriteBinary(data []byte) {
	_, _ = b.writeBinary(data)
}

// Size computes the size of a buffer by adding sizes of every chunk.
func (b *ChunkedBuffer) Size() int64 {
	size := int64(len(b.Buf))
	for _, buf := range b.bufs {
		size += int64(len(buf))
	}

	return size
}

// WriteTo drains the contents of the buffer to a writer then resets the buffer.
func (b *ChunkedBuffer) WriteTo(w io.Writer) (written int64, err error) {
	var n int
	for i, buf := range b.bufs {
		n, err = w.Write(buf)
		if err != nil {
			break
		}
		written += int64(n)

		if redeem := b.redeemers[i+1]; redeem != nil {
			redeem()
		}
	}

	if err == nil {
		n, err = w.Write(b.Buf)
		written += int64(n)
		if err == nil {
			if redeem := b.redeemers[0]; redeem != nil {
				redeem()
			}
		}
	}

	b.Reset()

	return
}

func (b *ChunkedBuffer) Reset() {
	for _, redeem := range b.redeemers {
		if redeem != nil {
			redeem()
		}
	}

	b.reset()
}

// TODO: option to provide buffer
func (b *ChunkedBuffer) Bytes() []byte {
	ret := make([]byte, 0, int(b.Size()))

	for i, buf := range b.bufs {
		ret = append(ret, buf...)
		if redeem := b.redeemers[i+1]; redeem != nil {
			redeem()
		}
	}

	ret = append(ret, b.Buf...)
	if redeem := b.redeemers[0]; redeem != nil {
		redeem()
	}

	b.reset()

	return ret
}

/*
// BuildBytes creates a single byte slice with all the contents of the buffer. Data is
// copied if it does not fit in a single chunk. You can optionally provide one byte
// slice as argument that it will try to reuse.
func (b *ChunkedBuffer) BuildBytes(reuse ...[]byte) []byte {
	if len(b.bufs) == 0 {
		ret := b.Buf
		b.toPool = nil
		b.Buf = nil
		return ret
	}

	var ret []byte
	size := int(b.Size())

	// If we got a buffer as argument and it is big enought, reuse it.
	if len(reuse) == 1 && cap(reuse[0]) >= size {
		ret = reuse[0][:0]
	} else {
		ret = make([]byte, 0, size)
	}
	for _, buf := range b.bufs {
		ret = append(ret, buf...)
		redeemBuf(buf)
	}

	ret = append(ret, b.Buf...)
	redeemBuf(b.toPool)

	b.bufs = nil
	b.toPool = nil
	b.Buf = nil

	return ret
}
*/

func (b *ChunkedBuffer) Err() error {
	return nil
}

func (b *ChunkedBuffer) Ok() bool {
	return true
}

func (b *ChunkedBuffer) WriteTextFrom(r io.Reader) {
	pooledBuf, redeem := copyBuffers.BorrowWithRedeem()
	defer redeem()

	pooledBuf.Append(copyPad...)
	buf := pooledBuf.Slice()

	for {
		nr, er := r.Read(buf)
		if nr > 0 {
			_, _ = writeEscapedBytes(buf[0:nr], b.escape, b.writeByte, b.writeBinary)
		}
		if er != nil {
			break
		}
	}
}

func (b *ChunkedBuffer) WriteBinaryFrom(r io.Reader) {
	pooledBuf, redeem := copyBuffers.BorrowWithRedeem()
	defer redeem()

	pooledBuf.Append(copyPad...)
	buf := pooledBuf.Slice()

	for {
		nr, er := r.Read(buf)
		if nr > 0 {
			b.WriteBinary(buf[0:nr])
		}
		if er != nil {
			break
		}
	}
}

// moreBuffers creates a new chunk because the current buffer is full.
func (b *ChunkedBuffer) moreBuffers(requested int) {
	// heuristic to allocate larger buffers aggressively
	requested = max(requested, minSize)
	requested = max(requested, len(b.bufs)*cap(b.Buf))
	requested = min(requested, maxSize)

	chunkSize := max(cap(b.Buf), minSize)
	for chunkSize < requested {
		chunkSize <<= 1
	}

	if cap(b.Buf) == 0 {
		// initial buffer
		buf, redeem := borrowBuf(chunkSize)
		b.Buf = buf
		b.redeemers = append(b.redeemers, redeem)

		return
	}

	// swap buffers
	b.bufs = append(b.bufs, b.Buf)
	b.redeemers = append(b.redeemers, b.redeemers[0])
	// borrow a new buffer from the pool
	b.Buf, b.redeemers[0] = borrowBuf(chunkSize)
}

func (b *ChunkedBuffer) writeByte(c byte) error {
	if cap(b.Buf) == len(b.Buf) {
		b.moreBuffers(1)
	}

	b.Buf = append(b.Buf, c)

	return nil
}

func (b *ChunkedBuffer) writeBinary(data []byte) (int, error) {
	for len(data) > 0 {
		if cap(b.Buf) == len(b.Buf) {
			b.moreBuffers(len(data))
		}

		sz := min(cap(b.Buf)-len(b.Buf), len(data))
		b.Buf = append(b.Buf, data[:sz]...)
		data = data[sz:]
	}

	return 0, nil
}

func (b *ChunkedBuffer) reset() {
	b.bufs = b.bufs[:0]
	b.redeemers = b.redeemers[:0]
	b.Buf = nil
}
