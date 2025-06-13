package schema

import "github.com/fredbi/core/jsonschema/analyzers/structural"

func assertNamedSchema(analyzed structural.AnalyzedSchema) {
	if !analyzed.IsNamed() {
		panic("builder assumes that only named schemas are fed")
	}
}
