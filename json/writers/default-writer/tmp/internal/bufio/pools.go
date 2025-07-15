//nolint:gochecknoglobals // pools are globals
package bufio

import (
	"sync"

	"github.com/fredbi/core/swag/pools"
)

const (
	minSize       = 512
	maxSizeFactor = 6
	maxSize       = 512 << maxSizeFactor // 32k
	copyBufferCap = 2 ^ 12               // 4k
)

var (
	// buffers holds [pools.PoolSlice] s of fixed size buffers, from 512b to 32k.
	buffers map[int]*pools.PoolSlice[byte]

	// copyBuffers is a pool of 32k buffers for ReadFrom operations
	copyBuffers = pools.NewPoolSlice[byte](pools.WithMinimumCapacity(copyBufferCap))

	// copyPad is allocated once to pad copy buffers with zeros.
	// this is a lazy initialization: memory won't be allocated if we never use the WriteFrom methods.
	copyPad = make([]byte, copyBufferCap)

	poolOfReaders = pools.NewRedeemable[readCloser]()

	initBuffersOnce sync.Once
)

func initializeBuffers() {
	// this is executed once on the first call to NewChunkedBuffer.

	buffers = make(map[int]*pools.PoolSlice[byte], maxSizeFactor+1)
	for sizeFactor := range maxSizeFactor + 1 {
		size := minSize << sizeFactor
		buffers[size] = pools.NewPoolSlice[byte](pools.WithMinimumCapacity(size))
	}
}

// borrowBuf recycles a chunk from the pool
func borrowBuf(size int) ([]byte, func()) {
	c, ok := buffers[size]
	if !ok {
		return make([]byte, 0, size), nil
	}

	v, redeem := c.BorrowWithRedeem()

	return v.Slice(), redeem
}
