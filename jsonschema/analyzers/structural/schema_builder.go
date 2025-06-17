package structural

// Builder builds an [AnalyzedSchema].
//
// This is used internally by the [SchemaAnalyzer].
// It may also be used by consumers of [AnalyzedSchema] s to build mocks.
type Builder struct {
}

func (b *Builder) Schema() AnalyzedSchema {
	return AnalyzedSchema{}
}
