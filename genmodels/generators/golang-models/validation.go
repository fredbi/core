package models

import "github.com/fredbi/core/jsonschema/analyzers/structural"

// enrich the TargetModel with validation generation instructions
//
// TODO
func (g *Generator) makeGenModelValidation(analyzed structural.AnalyzedSchema, in TargetModel) targetModelContext {
	return targetModelContext{
		TargetModel: in, // TODO
	}
}
