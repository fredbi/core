package golang

import (
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

// makeGenSchemas builds target models from a single analyzed schema.
//
// Each returned [TargetModel] will produce one source file.
//
// Sometimes, we want to split the rendering of a single analyzed schema into several source files.
// This is when makeGenSchemas returns several [TargetModels].
func (g *Generator) makeGenSchemas(_ structural.AnalyzedSchema) []TargetModel {
	// reuses the outcome of package planning (perhaps we could just make it on the fly?)
	// TODO

	// if named: push
	// if anonymous and !root: hold and stack
	return []TargetModel{}
}
