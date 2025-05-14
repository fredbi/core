package nodes

// Kind describes the kind of node in a JSON document
type Kind uint8

const (
	// KindNull is a node of type null
	KindNull Kind = iota
	// KindScalar is a node with a scalar value (not null)
	KindScalar
	// KindObject is an object container node
	KindObject
	// KindArray is an array container node
	KindArray
)
