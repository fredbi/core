package golang

import (
	settings "github.com/fredbi/core/genmodels/generator/common-settings"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

// planLayout extracts the named elements from the analyzer and prepares a package layout.
// TODO
func (g *Generator) planPackageLayout() error {
	if g.PackageLayout.Is(settings.PackageLayoutFlat) {
		// flat layout

		return nil
	}

	g.locations = make(LocationIndex, g.analyzer.Len())
	for namedSchema := range g.analyzer.BottomUpSchemas(structural.OnlyNamedSchemas()) {
		switch g.PackageLayoutOptions {
		case settings.PackageLayoutRefBased:
			// $ref-based layout
			if namedSchema.RefLocation == "" {
				continue
			}
		case settings.PackageLayoutTagBased:
			// tag-based layout
		case settings.PackageLayoutEnums:
			// enum layout
		}
	}

	return nil
}
