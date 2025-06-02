package order

type SchemaOrdering uint8

const (
	// NoOrder doesn't specify any particular ordering
	NoOrder SchemaOrdering = iota
	// TopDown iterates over the dependency graph of schemas from root nodes down to the leaves.
	TopDown
	// BottomUpiterates over the dependency graph of schemas from leave nodes up to the root nodes.
	BottomUp
)
