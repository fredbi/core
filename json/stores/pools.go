package stores

import "github.com/fredbi/core/swag/pools"

var poolOfOptions = pools.New[Options]()

// BorrowOptions borrows an [Options] from the pool and resets its content to the default settings.
//
// This is useful for implementions of [Store] that want to recycle [Options] when Get is called.
func BorrowOptions() *Options {
	return poolOfOptions.Borrow()
}

// RedeemOptions redeems an allocated [Options] to the pool, so it can be recycled.
func RedeemOptions(o *Options) {
	poolOfOptions.Redeem(o)
}
