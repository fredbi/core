package lexer

import (
	"io"
	"slices"

	"github.com/fredbi/core/swag/pools"
)

type poolOfLexers struct {
	*pools.PoolRedeemable[L]
}

// lexersPool is a redeemable pool: borrowing yields a cached redeem closure (no
// per-borrow allocation), and under the poolsdebug build tag it detects
// double-redeem, foreign-redeem and leaks.
var lexersPool = poolOfLexers{
	PoolRedeemable: pools.NewRedeemable[L](),
}

func (p *poolOfLexers) borrowWithBytes(data []byte, opts ...Option) (*L, func()) {
	l, redeem := p.BorrowWithRedeem()
	l.applyWithDefaults(opts)
	l.r = noopReader
	l.buffer = data
	l.bufferized = len(data)
	l.previousBuffer = nil
	l.keepPreviousBuffer = 0 // disabled option
	l.wholeBuffer = true     // the whole input is in the buffer: values may alias it
	l.reset()

	return l, redeem
}

func (p *poolOfLexers) borrowWithReader(r io.Reader, opts ...Option) (*L, func()) {
	l, redeem := p.BorrowWithRedeem()
	l.applyWithDefaults(opts)
	l.r = r
	l.bufferized = 0
	l.wholeBuffer = false // streaming: the buffer is refilled, values must be copied
	l.reset()

	if cap(l.buffer) < l.bufferSize {
		// reallocates an internal buffer only if options have changed
		l.buffer = slices.Grow(l.buffer, l.bufferSize-cap(l.buffer))[:l.bufferSize]
	}

	if l.keepPreviousBuffer > 0 && cap(l.previousBuffer) < l.keepPreviousBuffer {
		l.previousBuffer = slices.Grow(l.previousBuffer, l.keepPreviousBuffer-cap(l.previousBuffer))
	}

	return l, redeem
}

// BorrowLexerWithReader borrows a L(exer) from a global pool, together with the
// closure that redeems it back to the pool.
//
// This is equivalent to calling [New], but may recycle a previously allocated
// lexer if available from the pool. The internal buffer of the lexer is also
// reused, provided the [WithBufferSize] option has not changed the pooled size.
//
// The redeem closure must be called exactly once when the lexer is no longer
// needed (typically via defer); after calling it, drop the reference to the
// lexer. Calling it more than once panics. To maximize the amortizing effect of
// the pool, make sure every borrowed lexer is eventually redeemed.
func BorrowLexerWithReader(r io.Reader, opts ...Option) (*L, func()) {
	return lexersPool.borrowWithReader(r, opts...)
}

// BorrowLexerWithBytes borrows a L(exer) from a global pool, together with the
// closure that redeems it back to the pool.
//
// This is equivalent to calling [NewWithBytes], but may recycle a previously
// allocated lexer if available from the pool.
//
// The redeem closure must be called exactly once when the lexer is no longer
// needed (typically via defer); after calling it, drop the reference to the
// lexer. Calling it more than once panics.
func BorrowLexerWithBytes(data []byte, opts ...Option) (*L, func()) {
	return lexersPool.borrowWithBytes(data, opts...)
}
