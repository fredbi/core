//nolint:gochecknoglobals  // pools are globals
package writer

import (
	"io"

	"github.com/fredbi/core/json/writers/default-writer/internal/bufio"
	"github.com/fredbi/core/swag/pools"
)

const sensibleCapacityForNumbers = 20

var (
	poolOfWriters        = pools.New[W]()
	poolOfChunkedBuffers = pools.NewRedeemable[bufio.ChunkedBuffer]()
	poolOfUnbuffered     = pools.NewRedeemable[bufio.Unbuffered]()
	poolOfOptions        = pools.New[options]()
	poolOfNumberBuffers  = pools.NewPoolSlice[byte](
		pools.WithMinimumCapacity(sensibleCapacityForNumbers),
	)
)

// BorrowWriter recycles a writer [W] from the global pool.
//
// [BorrowWriter] is equivalent to [New], but may save the allocation of new resources if
// they are readily available in the pool.
//
// The caller is responsible for calling [RedeemWriter] after the work is done, and relinquish resources to the pool.
func BorrowWriter(writer io.Writer, opts ...Option) *W {
	w := poolOfWriters.Borrow()
	w.options = optionsWithDefaults(writer, opts)

	return w
}

// RedeemWriter relinquishes a borrowed writer [W] back to the global pool.
//
// Inner resources are relinquished too.
func RedeemWriter(w *W) {
	w.redeem() // redeem inner resources
	poolOfWriters.Redeem(w)
}
