package models

import (
	"errors"
	"fmt"

	settings "github.com/fredbi/core/genmodels/generators/common-settings"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
	"github.com/fredbi/core/jsonschema/analyzers/structural/bundle"
)

// planLayout extracts the named elements from the analyzer and prepares a package layout.
//
// Available strategies:
// * PackageLayoutFlat: all models in a single package, with name deconfliction
// * PackageLayoutHierarchicalEager: generate as many packages as we can
// * PackageLayoutHierarchicalLazy: generate only packages when a name conflict occurs
//
// Package layout options:
//   - PackageLayoutEnums: applicable to all strategies. Enum types and constants are separated
//   - PackageLayoutRefBased: for Hierarchical strategies. Decide packages from the path in $ref
//   - PackageLayoutTagBased: for Hierarchical strategies. Decide packages from a tag injected by the analyzer.
//     The tag may come from an overlay (e.g. x-go-tag) or inferred from an analysis of an OpenAPI spec
//
// Package override: x-go-package-override (alias x-go-package)
func (g *Generator) planPackageLayout() error {
	bundleOptions, err := g.makeBundleOptions()
	if err != nil {
		return errors.Join(err, ErrModel)
	}

	// reorganize the schema with proper names and layout
	bundled, err := g.analyzer.Bundle(bundleOptions...)
	if err != nil {
		return errors.Join(err, ErrModel)
	}

	// prepare the output folders according to plan
	for pkg := range bundled.Namespaces(structural.OnlyLeaves()) {
		if err := g.baseFS.MkdirAll(pkg, 0755); err != nil {
			return errors.Join(err, ErrModel)
		}
	}

	g.analyzer = bundled

	return nil
}

// makeBundleOptions configures all the naming and package layout for the analyzer to produce a bundled schema.
func (g *Generator) makeBundleOptions() ([]structural.BundleOption, error) {
	bundlingOptions := []structural.BundleOption{
		// configure naming callbacks
		structural.WithBundleNameProvider(structural.NameProvider(g.nameProvider.NameSchema)),
		structural.WithBundleNameEqualOperator(structural.EqualOperator(g.nameProvider.EqualName)),
		structural.WithBundlePathProvider(structural.NameProvider(g.nameProvider.NamePackage)),
		structural.WithBundlePathEqualOperator(structural.EqualOperator(g.nameProvider.EqualPath)),
		structural.WithBundleMarker(structural.SchemaMarker(g.nameProvider.Mark)),
		// other layout options that affect schema bundling
		structural.WithBundleSingleRoot(g.ModelLayout.Is(settings.ModelLayoutAllModelsOneFile)),
		structural.WithBundleEnums(g.PackageLayoutMode.Has(settings.PackageLayoutEnums)), // allow enums to be defined in their own package,
		structural.WithBundleEnumsPackage(g.EnumPackageName),
	}

	// configure bundling layout strategy
	switch {
	case g.PackageLayout.Is(settings.PackageLayoutFlat):
		// flat layout
		bundlingOptions = append(bundlingOptions, structural.WithBundleStragegy(bundle.Flat))
	case g.PackageLayout.Is(settings.PackageLayoutHierarchical):
		// hierarchical layout
		switch g.PackageLayoutOptions {
		case settings.PackageLayoutRefBased:
			// $ref-based layout
			bundlingOptions = append(bundlingOptions, structural.WithBundleStragegy(bundle.Hierarchical))
		case settings.PackageLayoutTagBased:
			// tag-based layout
			return nil, fmt.Errorf("unsupported package layout option: %v: %w", g.PackageLayout, ErrInit)
		}
	default:
		return nil, fmt.Errorf("unsupported package layout option: %v: %w", g.PackageLayout, ErrInit)
	}

	switch {
	case g.PackageLayoutMode.Has(settings.PackageLayoutEager):
		bundlingOptions = append(bundlingOptions, structural.WithBundleAggressiveness(bundle.Eager))
	case g.PackageLayoutMode.Has(settings.PackageLayoutLazy):
		bundlingOptions = append(bundlingOptions, structural.WithBundleAggressiveness(bundle.Lazy))
	default:
		return nil, fmt.Errorf("unsupported package layout mode: %8b: %w", g.PackageLayoutMode, ErrInit)
	}

	// probably a lot of other configurable stuff to add here

	return bundlingOptions, nil
}
