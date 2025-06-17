package analyzers

// InformationReport contains the audit trail of decisions made by the tools.
type InformationReport struct {
	DecisionType string
	Decision     string
	Originator   string   // program/function signature
	Sources      []string // json pointers to source schema or config
}

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
	SchemaKindNull
)

type PolymorphismKind uint8

const (
	PolymorphismNone PolymorphismKind = iota
	PolymorphismOneOf
	PolymorphismAnyOf
	PolymorphismBaseType
	// TODO: not sure about this: notice that the regular allOf (excl. base type special case) is not polymorphic
)

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
