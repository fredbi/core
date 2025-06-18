package providers

import (
	"errors"
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
		panic(fmt.Errorf("could not deconflict in a reasonable number of attempts (%d)", attempts))
	}
}

func assertAnonymousInParentObject(schema structural.AnalyzedSchema) {
	panic(errors.New("an anonymous schema found in a parent object must be in a property, additionalProperty, patternProperty or allOf"))
}

func assertAnonymousInParentArray(schema structural.AnalyzedSchema) {
	panic(errors.New("an anonymous schema found in a parent array must be in an items or allOf"))
}

func assertAllOfInParentArray(schema structural.AnalyzedSchema) {
	panic(errors.New("allOf in arrays should have been rewritten by the analyzer"))
}

func assertAnonymousInParentTuple(schema structural.AnalyzedSchema) {
	panic(errors.New("an anonymous schema found in a parent tuple must be in a schema array or additionalItems"))
}

func assertAnonymousInParentPolymorphic(schema structural.AnalyzedSchema) {
	panic(errors.New("an anonymous schema found in a parent polymorph must be allOf, oneOf or anyOf"))
}

func assertMustDeconflictPackageAlias(done bool, name string) {
	if !done {
		panic(fmt.Errorf("the package alias deconflicter should always manage to find a deconficted alias. Failed doing so for alias %q", name))
	}
}

func assertMustDeconflictUsingIndex(name string) {
	panic(fmt.Errorf("failed to deconflict package alias %q", name))
}
