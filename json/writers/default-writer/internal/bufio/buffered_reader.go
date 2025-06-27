package bufio

import "io"

// ReadCloser creates an [io.ReadCloser] that pulls all the buffer's content.
//
// The buffer is reset upon closing the reader.
func (b *ChunkedBuffer) ReadCloser() io.ReadCloser {
	r, redeem := poolOfReaders.BorrowWithRedeem()

	r.buffer = b
	r.redeemer = redeem

	return r
}

type readCloser struct {
	currentBuffer int
	offset        int
	buffer        *ChunkedBuffer
	redeemer      func()
}

func (r *readCloser) Reset() {
	r.currentBuffer = 0
	r.offset = 0
	r.buffer = nil
	r.redeemer = nil
}

func (r *readCloser) Read(p []byte) (int, error) {
	if len(p) == 0 {
		if len(r.buffer.Buf) == 0 && len(r.buffer.bufs) == 0 {
			return 0, io.EOF
		}

		return 0, io.ErrShortBuffer
	}

	n := 0
	if r.currentBuffer < len(r.buffer.bufs) {
		// drain buf list
		for ; r.currentBuffer < len(r.buffer.bufs); r.currentBuffer++ {
			buf := r.buffer.bufs[r.currentBuffer]

			for n < len(p) {
				copied := r.drain(p[n:], buf)
				if copied == 0 {
					break
				}
				n += copied
			}

			if n == len(p) {
				return n, nil
			}
		}
	}

	// drain current Buf
	buf := r.buffer.Buf
	for n < len(p) {
		copied := r.drain(p[n:], buf)
		if copied == 0 {
			break
		}
		n += copied
	}

	return n, nil
}

func (r *readCloser) Close() error {
	r.buffer.Reset()

	r.redeemer()

	return nil
}

func (r *readCloser) drain(p []byte, buf []byte) int {
	if r.offset == len(buf) {
		return 0
	}
	copied := copy(p, buf[r.offset:])
	r.offset += copied

	return copied
}
