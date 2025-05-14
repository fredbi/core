package store

import (
	"bytes"
	"compress/flate"
	"io"
	"sync"

	"github.com/fredbi/core/swag/pools"
)

const (
	maxTemporarySliceCapacity = maxInlineBytes * digitsPerByte
)

var (
	poolOfStores       = pools.New[Store]()
	poolOfBytesBuffers = pools.NewRedeemable[bytes.Buffer]()
	poolOfBytes        = pools.NewPoolSlice[byte](pools.WithMinimumCapacity(maxTemporarySliceCapacity))
)

// BorrowStore borrows a new or recycled [Store] from the pool.
func BorrowStore(opts ...Option) *Store {
	s := poolOfStores.Borrow()
	if len(opts) > 0 {
		s.options = applyOptionsWithDefaults(opts)
	}

	return s
}

// RedeemStore redeems a previously borrowed [Store] to the pool.
func RedeemStore(s *Store) {
	poolOfStores.Redeem(s)
}

func borrowBufferWithRedeem(size int) (*bytes.Buffer, func()) {
	b, redeem := poolOfBytesBuffers.BorrowWithRedeem() // bytes.Buffer knows how to Reset
	if size > b.Cap() {
		b.Grow(size)
	}

	return b, redeem
}

func borrowBytesWithRedeem(size int) ([]byte, func()) {
	b, redeem := poolOfBytes.BorrowWithSizeAndRedeem(size)

	return b.Slice(), redeem
}

// flateReadersPool is a memory pool of [flate.Reader] s
type flateReadersPool struct {
	sync.Pool
}

type flateReaderRedeemable struct {
	inner    flateReader
	redeemer func()
}

var poolOfFlateReaders = newFlateReadersPool()

func newFlateReadersPool() *flateReadersPool {
	p := &flateReadersPool{}
	p.Pool = sync.Pool{
		New: func() any {
			var dummyReader bytes.Buffer
			r := &flateReaderRedeemable{
				inner: flate.NewReader(&dummyReader).(flateReader),
			}
			r.redeemer = func() { p.Put(r) }

			return r
		},
	}

	return p
}

func borrowFlateReaderWithRedeem(rdr io.Reader, dict []byte) (flateReader, func()) {
	raw := poolOfFlateReaders.Get()
	container := raw.(*flateReaderRedeemable)
	reader := container.inner
	_ = reader.Reset(rdr, dict)

	return reader, container.redeemer
}
