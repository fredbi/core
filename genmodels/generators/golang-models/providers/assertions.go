package providers

import (
	"fmt"

	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

func assertNameSchema(schema structural.AnalyzedSchema) {
	if !schema.IsNamed() {
		panic("expect a named schema here")
	}
}

func assertNotInfiniteAttempts(attempts int) {
	const tooMany = 100
	if attempts > tooMany {
		panic(fmt.Errorf("could not deconflict in a reasonable number of attemps (%d)", attempts))
	}
}

func assertAnonymousInParentObject(schema structural.AnalyzedSchema) {
	panic(fmt.Errorf("an anonymous schema found in a parent object must be in a property or additionalProperty"))
}

func assertAnonymousInParentTuple(schema structural.AnalyzedSchema) {
	panic(fmt.Errorf("an anonymous schema found in a parent tuple must be in a schema array or additionalItems"))
}

func assertAnonymousInParentPolymorphic(schema structural.AnalyzedSchema) {
	panic(fmt.Errorf("an anonymous schema found in a parent polymorph must be allOf, oneOf or anyOf"))
}
