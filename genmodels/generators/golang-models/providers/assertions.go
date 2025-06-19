package providers

import (
	"fmt"

	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

/*
func assertNameSchema(schema structural.AnalyzedSchema) {
	if !schema.IsNamed() {
		panic("expect a named schema here")
	}
}
*/

/*
func assertNotInfiniteAttempts(attempts int) {
	const tooMany = 100
	if attempts > tooMany {
		panic(fmt.Errorf("could not deconflict in a reasonable number of attempts (%d)", attempts))
	}
}
*/

func assertAnonymousInParentObject(_ structural.AnalyzedSchema) {
	panic(fmt.Errorf(
		"an anonymous schema found in a parent object must be in a property, additionalProperty, patternProperty or allOf: %w",
		ErrInternal,
	))
}

func assertAnonymousInParentArray(_ structural.AnalyzedSchema) {
	panic(fmt.Errorf(
		"an anonymous schema found in a parent array must be in an items or allOf: %w",
		ErrInternal,
	))
}

func assertAllOfInParentArray(_ structural.AnalyzedSchema) {
	panic(fmt.Errorf(
		"allOf in arrays should have been rewritten by the analyzer: %w",
		ErrInternal,
	))
}

func assertAnonymousInParentTuple(_ structural.AnalyzedSchema) {
	panic(fmt.Errorf(
		"an anonymous schema found in a parent tuple must be in a schema array or additionalItems: %w",
		ErrInternal,
	))
}

func assertAnonymousInParentPolymorphic(_ structural.AnalyzedSchema) {
	panic(fmt.Errorf(
		"an anonymous schema found in a parent polymorph must be allOf, oneOf or anyOf: %w",
		ErrInternal,
	))
}

/*
func assertMustDeconflictPackageAlias(done bool, name string) {
	if !done {
		panic(
			fmt.Errorf(
				"the package alias deconflicter should always manage to find a deconficted alias. Failed doing so for alias %q",
				name,
			))
	}
}
*/

func assertMustDeconflictUsingIndex(name string) {
	panic(fmt.Errorf(
		"failed to deconflict name using an index strategy that is supposed to always work: %q: %w",
		name, ErrInternal,
	))
}

func assertConflictMetaMustHavePackage(pkg *structural.AnalyzedPackage, name string) {
	if pkg == nil {
		panic(fmt.Errorf(
			"a ConflictMeta provided by the structural.SchemaAnalyzer on a package is expected to contain the AnalyzedPackage: %q: %w",
			name,
			ErrInternal,
		))
	}
}

func assertInvalidKindScalar(v stores.Value) {
	panic(fmt.Errorf(
		"JSON token Kind unexpected: a KindScalar node can only be String, Boolean, Null or Number. Got %v: %w",
		v,
		ErrInternal,
	))
}
