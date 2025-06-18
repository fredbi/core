package schema

// Builder knows how to produce a data model [model.TargetSchema] from	a [structural.AnalyzedSchema],
// to be consumed by model generation templates.
type Builder struct {
	options
}

// NewBuilder constructs a new schema [Builder].
func NewBuilder(opts ...Option) *Builder {
	return &Builder{
		options: optionsWithDefaults(opts),
	}
}
