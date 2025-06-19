package schema

import (
	"iter"

	model "github.com/fredbi/core/genmodels/generators/golang-models/data-model"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

// GenNamedPackages transforms a [structural.AnalyzedPackage] into a series of [model.TargetPackage] s to be consumed by
// templates.
//
// The seed is used as template to capture options and settings from the caller's context.
//
// A [model.TargetPackage] structure is used to build code that is produced once per package.
func (g *Builder) GenNamedPackages(
	analyzed structural.AnalyzedPackage,
	seed model.TargetPackage,
) iter.Seq[model.TargetPackage] {
	return nil
}
