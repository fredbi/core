package nodes

// Kind describes the kind of node in a JSON document.
//
// A JSON document is organized as a hierarchy of nodes of 4 kinds: null, scalar, object and array.
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

func (k Kind) String() string {
	switch k {
	case KindObject:
		return "object"
	case KindArray:
		return "array"
	case KindScalar:
		return "scalar"
	case KindNull:
		fallthrough
	default:
		return "null"
	}
}
