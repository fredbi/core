package writer

import "github.com/fredbi/core/swag/pools"

var poolOfWriters = pools.New[W]()

func BorrowWriter(opts ...Option) *W {
	w := poolOfWriters.Borrow()
	w.options = optionsWithDefaults(opts)

	return w
}

func RedeemWriter(w *W) {
	w.buffer.Reset() // redeem inner buffers
	poolOfWriters.Redeem(w)
}
