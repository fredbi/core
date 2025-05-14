package lexer

import (
	"io"
	"slices"

	"github.com/fredbi/core/swag/pools"
)

type poolOfLexers struct {
	*pools.Pool[L]
}

var lexersPool = poolOfLexers{
	Pool: pools.New[L](),
}

// BorrowWithBytes
func (p *poolOfLexers) BorrowWithBytes(data []byte, opts ...Option) *L {
	l := p.Borrow()
	l.applyWithDefaults(opts)
	l.r = noopReader
	l.buffer = data
	l.bufferized = len(data)
	l.previousBuffer = nil
	l.keepPreviousBuffer = 0 // disabled option
	l.reset()

	return l
}

func (p *poolOfLexers) BorrowLexerWithReader(r io.Reader, opts ...Option) *L {
	l := p.Borrow()
	l.applyWithDefaults(opts)
	// TODO: in reset
	l.r = r
	l.bufferized = 0
	l.reset()

	if cap(l.buffer) < l.bufferSize {
		// reallocates an internal buffer only if options have changed
		l.buffer = slices.Grow(l.buffer, l.bufferSize-cap(l.buffer))[:l.bufferSize]
	}

	if l.keepPreviousBuffer > 0 && cap(l.previousBuffer) < l.keepPreviousBuffer {
		l.previousBuffer = slices.Grow(l.previousBuffer, l.keepPreviousBuffer-cap(l.previousBuffer))
	}

	return l
}

// BorrowLexerWithReader borrows a L(exer) from a global pool.
//
// This is equivalent to calling New(), but may recycle a previously allocated lexer if available from the pool.
//
// The internal buffer of the lexer is also reused, provided the WithBufferSize() option has not changed the size of
// / pooled buffers.
//
// To maximize the amortizing effecto of the pool, make sure that all borrowed lexers are eventually redeemed to the pool.
func BorrowLexerWithReader(r io.Reader, opts ...Option) *L {
	return lexersPool.BorrowLexerWithReader(r, opts...)
}

// BorrowLexerWithBytes borrows a L(exer) from a global pool.
//
// This is equivalent to calling NewWithBytes(), but may recycle a previously allocated lexer if available from the pool.
//
// To maximimize the amortizing effect of the pool, make sure that all borrowed lexers are eventually redeemed to the pool.
func BorrowLexerWithBytes(data []byte, opts ...Option) *L {
	return lexersPool.BorrowWithBytes(data, opts...)
}

// RedeemLexer redeems a L(exer) to the global pool, so it may be reused.
//
// Make sure that you don't redeem a lexer twice.
func RedeemLexer(l *L) { lexersPool.Redeem(l) }
