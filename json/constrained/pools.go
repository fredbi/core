//nolint:gochecknoglobals // pools are global variables
package constrained

import "github.com/fredbi/core/swag/pools"

var (
	poolOfObjectContexts                 = pools.New[objectContext]()
	poolOfArrayContexts                  = pools.New[arrayContext]()
	poolOfStringOrArrayContexts          = pools.New[stringOfArrayContext]()
	poolOfBoolOrObjectContexts           = pools.New[boolOrObjectContext]()
	poolOfObjectOrArrayOfObjectsContexts = pools.New[objectOrArrayOfObjectsContext]()
)
