package models

import (
	"context"
	"errors"
	"fmt"
	"iter"

	"github.com/fredbi/core/codegen/genapp"
	settings "github.com/fredbi/core/genmodels/generators/common-settings"
	model "github.com/fredbi/core/genmodels/generators/golang-models/data-model"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
	"github.com/fredbi/core/jsonschema/analyzers/structural/bundle"
	"golang.org/x/sync/errgroup"
)

// planLayout extracts the named elements from the analyzer and prepares a package layout.
//
// This part that decides all the naming strategy of the generated target.
//
// # Naming strategy
//
// The naming strategy is brought to the [structural.Analyzer] by a [NameProvider] (e.g. [providers.NameProvider]).
//
// During the schemas bundling phase, all thing that the generator wish to have a name provide one using callbacks.
//
// Callbacks allow for custom extensions to alter automatically determined names.
//
// Name override: x-go-name
//
// # Layout strategies
//
// - PackageLayoutFlat: all models in a single package, with name deconfliction
// - PackageLayoutHierarchical: exploits the hierarchy of schemas to define packages
//
// Package layout options:
//
//   - PackageLayoutEnums: applicable to all strategies. Enum types and constants are separated
//   - PackageLayoutRefBased: for Hierarchical strategies. Decide packages from the path in $ref
//   - PackageLayoutTagBased: for Hierarchical strategies. Decide packages from a tag injected by the analyzer
//     (e.g. from an OpenAPI specification).
//   - PackageLayoutEager: generate as many packages as we can
//   - PackageLayoutLazy: generate only packages when a name conflict occurs
//
// Package override: x-go-package-override (alias x-go-package)
//
// The tag may come from an overlay (e.g. x-go-tag) or inferred from an analysis of an OpenAPI spec.
//
// # Layout actions
//
// The package layout consists of:
//
//   - creating the directories to hold package source files
//   - optionally initializing a go.mod file for the generated target
//   - creating specific files that are unique to a package (e.g. doc.go, other utilities)
//     This code generation may be carried out in parallel.
func (g *Generator) planPackageLayout() error {
	bundleOptions, err := g.makeBundleOptions()
	if err != nil {
		return err
	}

	// 1. reorganize the schemas hierarchy with proper names and layout
	//
	// This step invokes the configured callbacks to define if and how schemas and packages are named.
	//
	// This is also the moment when we reorganize the '$ref's in the schema collection to achieve the desired layout.
	bundled, err := g.analyzer.Bundle(bundleOptions...)
	if err != nil {
		return errors.Join(err, ErrModel)
	}
	g.l.Info("schema bundled")

	// 2. prepare the output folders according to plan
	//
	// This step only needs to retrieve the generatormost folders to create the entire source tree structure.
	var numPkg int
	for folder := range bundled.Namespaces(structural.OnlyLeaves()) {
		if err := g.baseFS.MkdirAll(folder, 0755); err != nil {
			return errors.Join(err, ErrModel)
		}
		numPkg++
	}
	g.l.Info("package folders created", "packages", numPkg)

	if g.GenOptions.WantsGoMod {
		// TODO: should warn this doesn't work with afero FS
		if err := g.generator.GoMod(
			genapp.WithModulePath(g.nameProvider.PackageFullName(g.outputPath)),
			genapp.WithGoVersion(g.GenOptions.MinGoVersion),
		); err != nil {
			return errors.Join(err, ErrModel)
		}

		g.l.Info("module initialized", "target_package", g.outputPath)
	}

	if g.GenOptions.WantsPkgArtifact() {
		// 3. generate package-level source files (e.g. doc.go, README.md, utils.go...)
		//
		// This content is typically not dependent on the generated models.
		//
		// Code generation may run concurrently.
		genGroup, _ := errgroup.WithContext(context.Background())
		genGroup.SetLimit(g.maxParallel())
		numPkg = 0

		for pkg := range bundled.Packages( /* sort option */ ) { // at this moment, package-level code doesn't require any specific ordering
			for genPkg := range g.makeGenPackage(pkg) {
				pkgTemplate := genPkg.Template

				genGroup.Go(func() error {
					return g.generator.Render(pkgTemplate, genPkg.FileName(), genPkg)
				})
				numPkg++
			}
		}
		if err := genGroup.Wait(); err != nil {
			return errors.Join(err, ErrModel)
		}

		g.l.Info("package-level artifacts created", "packages", numPkg)
	}

	// 4. supersede the analyzer with the bundled version
	g.analyzer = bundled

	return nil
}

// makeGenPackage returns an iterator over the targets to generate at the package level.
//
// Like with [model.TargetModel] and [model.TargetSchema], we may define several targets for a single package.
//
// At this moment, the configuration only produces a "doc.go" in each package.
func (g *Generator) makeGenPackage(pkg structural.AnalyzedPackage) iter.Seq[model.TargetPackage] {
	// We need:
	// TargetOptions like we have for schemas
	seed := model.TargetPackage{}

	return g.packageBuilder.GenNamedPackages(pkg, seed) // TODO
}

// makeBundleOptions configures all the naming and package layout for the analyzer to produce a bundled schema.
func (g *Generator) makeBundleOptions() ([]structural.BundleOption, error) {
	bundlingOptions := []structural.BundleOption{
		// configure naming callbacks to name packages and schemas during bundling
		structural.WithBundleNameProvider(structural.NameProvider(g.nameProvider.NameSchema)),      // callback to name schemas
		structural.WithBundleNameEqualOperator(structural.EqualOperator(g.nameProvider.EqualName)), // callback to detect name conflicts on named schemas
		structural.WithBundlePathProvider(structural.NameProvider(g.nameProvider.NamePackage)),     // callback to name package paths
		structural.WithBundlePathEqualOperator(structural.EqualOperator(g.nameProvider.EqualPath)), // callback to detect name conflicts on package paths
		structural.WithBundleMarker(structural.SchemaMarker(g.nameProvider.Mark)),                  // callback to amend the schema on-the-fly with extra extensions
		//
		// other layout options that affect schema bundling
		structural.WithBundleSingleRoot(g.ModelLayout.Is(settings.ModelLayoutAllModelsOneFile)), // enforce a single root schema, so we may pack everything into one file if desired
		structural.WithBundleEnums(g.PackageLayoutMode.Has(settings.PackageLayoutEnums)),        // allow enums to be defined in their own package
		structural.WithBundleEnumsPackage(g.EnumPackageName),                                    // if PackageLayoutEnums is enabled, the package name to define (e.g. "enums")
	}

	// configure the layout strategy for bundling
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

	// configure how aggressive the hierarchical layout should be, i.e. produce as many separate packages
	// as possible ("eager") vs produce separate packages only when a naming conflict occurs ("lazy").
	switch {
	case g.PackageLayoutMode.Has(settings.PackageLayoutEager):
		bundlingOptions = append(bundlingOptions, structural.WithBundleAggressiveness(bundle.Eager))
	case g.PackageLayoutMode.Has(settings.PackageLayoutLazy):
		bundlingOptions = append(bundlingOptions, structural.WithBundleAggressiveness(bundle.Lazy))
	default:
		return nil, fmt.Errorf("unsupported package layout mode: %8b: %w", g.PackageLayoutMode, ErrInit)
	}

	// probably a lot of other configurable stuff to add here...

	return bundlingOptions, nil
}
