//nolint:gochecknoglobals
package light

import (
	"github.com/fredbi/core/swag/pools"
)

const defaultPathCapacity = 32

var (
	poolOfParentContexts = pools.NewRedeemable[ParentContext]()
	poolOfPaths          = pools.NewPoolSlice[stringOrInt](
		pools.WithMinimumCapacity(defaultPathCapacity),
	)
)

// BorrowParentContext borrows a [ParentContext] from the pool together with its redeem closure.
//
// Call the returned closure exactly once to return the context to the pool; calling it twice panics.
// The context is reset on borrow and on redeem, so it never pins the injected store, lexer, writer or
// path while it sits idle in the pool.
func BorrowParentContext() (*ParentContext, func()) {
	return poolOfParentContexts.BorrowWithRedeem()
}

// BorrowPath borrows a [Path] from the pool together with its redeem closure.
//
// Call the returned closure exactly once to return the underlying buffer to the pool.
func BorrowPath() (Path, func()) {
	p, redeem := poolOfPaths.BorrowWithRedeem()

	return Path(p.Slice()), redeem
}
