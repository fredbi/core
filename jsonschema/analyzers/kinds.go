package analyzers

type UniqueID string

func (u UniqueID) String() string {
	return string(u)
}

type SchemaKind uint8

const (
	SchemaKindNone SchemaKind = iota
	SchemaKindObject
	SchemaKindArray
	SchemaKindTuple
	SchemaKindPolymorphic
	SchemaKindScalar
)

func (s SchemaKind) String() string {
	switch s {
	case SchemaKindObject:
		return "object"
	case SchemaKindArray:
		return "array"
	case SchemaKindTuple:
		return "tuple"
	case SchemaKindPolymorphic:
		return "polymorphic"
	case SchemaKindScalar:
		return "scalar"
	case SchemaKindNone:
		fallthrough
	default:
		return "any"
	}
}

type PolymorphismKind uint8

const (
	PolymorphismNone PolymorphismKind = iota
	PolymorphismOneOf
	PolymorphismAnyOf
	PolymorphismBaseType
	// TODO: not sure about this: notice that the regular allOf (excl. base type special case) is not polymorphic
)

func (s PolymorphismKind) String() string {
	switch s {
	case PolymorphismOneOf:
		return "oneOf"
	case PolymorphismAnyOf:
		return "anyOf"
	case PolymorphismBaseType:
		return "base-type"
	case PolymorphismNone:
		fallthrough
	default:
		return "none"
	}
}

type ScalarKind uint8

const (
	ScalarKindString ScalarKind = iota
	ScalarKindNumber
	ScalarKindInteger
	ScalarKindBool
	ScalarKindNull
)

func (s ScalarKind) String() string {
	switch s {
	case ScalarKindString:
		return "string"
	case ScalarKindNumber:
		return "number"
	case ScalarKindInteger:
		return "integer"
	case ScalarKindBool:
		return "bool"
	case ScalarKindNull:
		fallthrough
	default:
		return "null"
	}
}
