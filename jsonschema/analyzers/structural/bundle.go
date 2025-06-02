package structural

// NameProvider knows how to name a schema, given a suggested name and the schema to be named for more context.
type NameProvider func(name string, analyzed AnalyzedSchema) (string, error)

// NameSchema names a schema. It may return an error if the naming operation is impossible.
func (p NameProvider) NameSchema(name string, analyzed AnalyzedSchema) (string, error) {
	return p(name, analyzed)
}

// EqualOperator compares names and yields true if they are equivalent.
type EqualOperator func(string, string) bool

// Equal yields true if two names are equivalent and would conflict.
func (o EqualOperator) Equal(a string, b string) bool {
	return o(a, b)
}

// SchemaMarker allows a callback to inject extensions to the schema (e.g. "x-go-*" marks)
type SchemaMarker func(analyzed AnalyzedSchema) Extensions

// MarkSchema returns extensions to be merged to the current analyzed schema.
func (m SchemaMarker) MarkSchema(analyzed AnalyzedSchema) Extensions {
	return m(analyzed)
}

type Namespace map[string]struct{}
