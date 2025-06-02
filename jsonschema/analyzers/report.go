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
