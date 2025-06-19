package models

import (
	"iter"

	settings "github.com/fredbi/core/genmodels/generators/common-settings"
	model "github.com/fredbi/core/genmodels/generators/golang-models/data-model"
	"github.com/fredbi/core/jsonschema/analyzers"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

const sensibleAllocs = 10

type targetModelContext struct {
	model.TargetModel

	// possibly extensible here
}

// makeGenModels builds target models from a single analyzed schema.
//
// It deals specifically with the complexity of generating several target files that pertain to a single type.
//
// Each returned [TargetModel] will produce one source file.
//
// The details of how the [structural.AnalyzedSchema] is mapped into a [model.TargetSchema] (or several ones)
// are handed over to [Generator.makeGenSchema] below.
//
// # File layout decisions
//
// One source file may contain one of several type definitions or validation code only for several types.
//
// Conversely, a single schema may produce several files:
//
//  1. at least the type definition
//  2. possibly, a separate source for the Validate method
//  3. possibly, a test file
//  4. if the Validate method is in a separate file and we want tests, these produce another file
//
// This is when makeGenModels returns several [TargetModels].
//
// # Settings that drive file layout
//
// - ModelLayout
//
// - x-go-wants-validation (x-go-validation)
// - x-go-wants-split-validation (x-go-split-validation)
// - x-go-wants-test (x-go-test)
func (g *Generator) makeGenModels(analyzed structural.AnalyzedSchema) iter.Seq[model.TargetModel] {
	// 1. construct the base data model for this schema
	genModelWithContext := g.makeGenModel(analyzed)

	// 2. infer code layout options
	layout := g.layoutOptionsForSchema(analyzed)
	genModel := genModelWithContext.TargetModel

	// 3. determine if we should stash that one for a later time
	if parent, shouldStash := g.shouldStashModel(genModel); shouldStash {
		// case 3.1: when related models are defined in a single file, merge their schema(s) and imports
		// and stash these. The stash will be collected by the parent.
		//
		// case 3.2: when all models are defined in a single file, merge. The stash will be collected by the
		// ultimate parent.
		g.stashMx.Lock()
		defer g.stashMx.Unlock()

		stashed, foundInStash := g.stashedSchemas[analyzed.ID()]
		if foundInStash {
			genModel.MergeSchemas(stashed.Schemas)
			genModel.Imports = g.deconflictedMergedImports(genModel.Imports, stashed.Imports)
		}

		// stash this model for later generation
		g.stashedSchemas[parent] = genModel

		return nil // skip this generation target
	}

	g.stashMx.RLock()
	_, foundInStash := g.stashedSchemas[analyzed.ID()]
	g.stashMx.RUnlock()

	// 4. determine if we should merge previously stashed models
	if foundInStash {
		// previous schemas have stashed their results for reuse by the current schema.
		//
		// We may safely assume that by now, all dependency schemas have been processed, so the
		// stash for this key is no longer updated.
		g.stashMx.Lock()
		stashed := g.stashedSchemas[analyzed.ID()]
		genModel.MergeSchemas(stashed.Schemas) // dependencies are sorted in reverse, to reflect nice code with dependencies last
		genModel.Imports = g.deconflictedMergedImports(genModel.Imports, stashed.Imports)
		delete(g.stashedSchemas, analyzed.ID())
		g.stashMx.Unlock()
	}

	// 5. yields the iterator over model variants based on the same genModel.
	//
	// Produced models will differ only by their target file name, their layout flags and possibly imports.
	//
	// It is possible to configure these models with different templates. At present, the "model" top-level
	// templates serves as a switch.
	const (
		// list of possibly generated files from a single analyzed schema
		typeDefinitionModel int = iota
		typeValidationModel
		typeDefinitionTest
		typeValidationTest
	)

	return func(yield func(model.TargetModel) bool) {
		var targetModel model.TargetModel
		for _, targetCode := range []int{
			typeDefinitionModel,
			typeValidationModel,
			typeDefinitionTest,
			typeValidationTest,
		} {
			switch targetCode {

			case typeDefinitionModel:
				// we want to produce a file with the type definition, {name}.go
				targetModel = genModel
				flags := flagsForTypeDefinition
				flags.NeedsValidation = !layout.SchemaWantsSplitValidation
				targetModel.TargetCodeFlags = applyLayoutOptions(genModel.TargetCodeFlags, flags)
				targetModel.Schemas = applyLayoutOptionsToSchemas(targetModel.Schemas, flags)

			case typeValidationModel:
				// we want to produce a file with only the Validate method, as it may be quite large
				if !layout.SchemaWantsSplitValidation || !targetModel.NeedsValidation {
					continue
				}

				// yield a version of this [TargetModel] to generate {name}_validation.go (possibly deconflicted)
				targetModel = genModel
				targetModel.File = g.nameProvider.FileName(targetModel.File+"_validation", analyzed)
				targetModel.TargetCodeFlags = applyLayoutOptions(genModel.TargetCodeFlags, flagsForTypeValidation)
				targetModel.Schemas = applyLayoutOptionsToSchemas(targetModel.Schemas, flagsForTypeValidation)

			case typeDefinitionTest:
				// we want to produce a test for the type definition
				if !layout.SchemaWantsTest {
					// no specific file: skip
					continue
				}

				// yield a version of this [TargetModel] to generate {name}_test.go (possibly deconflicted)
				targetModel = genModel
				targetModel.File = g.nameProvider.FileNameForTest(targetModel.File+"_test", analyzed)
				flags := flagsForTypeTest
				flags.NeedsValidation = !layout.SchemaWantsSplitValidation
				targetModel.TargetCodeFlags = applyLayoutOptions(genModel.TargetCodeFlags, flags)
				targetModel.Schemas = applyLayoutOptionsToSchemas(targetModel.Schemas, flags)

				testImports := model.MakeImportsMap(g.extraModelImports(flags)...)
				targetModel.Imports = g.deconflictedMergedImports(testImports, targetModel.Imports)

			case typeValidationTest:
				// we want to produce a test for the Validate method only
				if !layout.SchemaWantsSplitValidation || !layout.SchemaWantsTest || !targetModel.NeedsValidation {
					// no specific file: skip
					continue
				}

				// yield a version of this [TargetModel] to generate {name}_validation_test.go, with only tests for validation code
				targetModel = genModel
				targetModel.File = g.nameProvider.FileNameForTest(targetModel.File+"_validation_test", analyzed)
				flags := flagsForValidationTest
				targetModel.TargetCodeFlags = applyLayoutOptions(genModel.TargetCodeFlags, flags)
				targetModel.Schemas = applyLayoutOptionsToSchemas(targetModel.Schemas, flagsForValidationTest)

				testImports := model.MakeImportsMap(g.extraModelImports(flags)...)
				targetModel.Imports = g.deconflictedMergedImports(testImports, targetModel.Imports)
			}

			if !yield(targetModel) {
				return
			}
		}
	}
}

func (g *Generator) deconflictedMergedImports(base model.ImportsMap, merged model.ImportsMap) model.ImportsMap {
	deconflict := func(alias string) string {
		deconflicted, _ := g.nameProvider.DeconflictAlias(alias, base)

		return deconflicted
	}

	base.MergeDeconflicted(merged, deconflict)

	return base
}

// defaultModelImports defines default imports for all models.
//
// TODO(fred): imports depends on options ... etc. This will be a last minute tuning
func (g *Generator) defaultModelImports() []model.AliasedImport {
	// 1. customize import to support formats
	formatsPkg := g.FormatsImportPath
	if formatsPkg == "" {
		formatsPkg = "github.com/fredbi/core/strfmt/jsonschema-formats"
	}

	baseImports := []model.AliasedImport{
		{
			Alias:   "apierrors",
			Name:    "errors",
			Package: "github.com/fredbi/core/errors", // possible to override this with settings
		},
		{
			Alias:   "formats",
			Name:    "formats",
			Package: formatsPkg,
		},
	}

	// 2. configure JSON std lib
	if g.WantsSerializer {
		switch {
		case g.JSONLibSelector.Is(model.JSONStdLib):
			baseImports = append(baseImports, model.AliasedImport{
				Alias:   "json",
				Name:    "json",
				Package: "encoding/json",
			})
		case g.JSONLibSelector.Is(model.JSONLibGoCCY):
			baseImports = append(baseImports, model.AliasedImport{
				Alias:   "json",
				Name:    "json",
				Package: "github.com/goccy/go-json",
			})
		case g.JSONLibSelector.Is(model.JSONLibJsoniter):
			baseImports = append(baseImports, model.AliasedImport{
				Alias:   "jsoniter",
				Name:    "jsoniter",
				Package: "github.com/json-iterator/go",
			})
		}
	}

	// 3. other imports

	return baseImports
}

// extraModelImports lists supplementary imports required for alternative model targets,
// e.g test code.
//
// Extra imports are thus determined by the model.TargetCodeFlags
func (g *Generator) extraModelImports(layout model.TargetCodeFlags) []model.AliasedImport {
	imports := make([]model.AliasedImport, 0, sensibleAllocs)

	if layout.NeedsTest {
		imports = append(imports, model.AliasedImport{
			Alias:   "testing",
			Name:    "testing",
			Package: "testing",
		})
	}

	return imports
}

// makeGenModel produces a seed model from a [structural.AnalyzedSchema].
//
// It decides whether to stash it for later generation (e.g. several schemas in a single file), or to render
// it straight away.
func (g *Generator) makeGenModel(analyzed structural.AnalyzedSchema) targetModelContext {
	seed := model.TargetModel{
		GenModelOptions: model.GenModelOptions{
			GenOptions: &g.GenOptions,
		},
		ID:   analyzed.ID(),
		Name: analyzed.OriginalName(),
		LocationInfo: model.LocationInfo{
			Package:         g.nameProvider.PackageShortName(analyzed.Path(), analyzed),
			PackageLocation: analyzed.Path(),                                           // Location is a sanitized /-separated URL path. This is relative to the codegen path
			FullPackage:     g.nameProvider.PackageFullName(analyzed.Path(), analyzed), // fully qualified package name (e.g. "github.com/fredbi/core/models")
			File:            g.nameProvider.FileName(analyzed.Name(), analyzed),        // deconflicted file name, safe for go
		},
		Imports: model.MakeImportsMap(g.defaultModelImports()...),
		Schemas: make([]model.TargetSchema, 0, sensibleAllocs),
	}

	switch {
	case g.ModelLayout.Is(settings.ModelLayoutRelatedModelsOneFile):
		// in this layout (the default), a single source file is produced for a type and the related ones.
		// (e.g. types that define the elements of a slice or a map).
		//
		// When a RelatedParent is defined, the model will be stashed for later merge and rendering in the same file
		// as its RelatedParent.
		if !analyzed.IsRoot() && analyzed.WasAnonymous() {
			parent := analyzed.Parent()
			parentID := parent.ID()
			seed.RelatedParent = parentID
		}
	case g.ModelLayout.Is(settings.ModelLayoutAllModelsOneFile):
		// in this layout, all models are packed in a single source file
		//
		// When an UltimateParent is defined, the model with be stashed for later merge and rendering in a single file,
		// which naming will be decided by the root schema (or pseudo-root if the dependency graph is multi-rooted).
		parent := analyzed.UltimateParent()
		parentID := parent.ID()
		seed.UltimateParent = parentID

	default:
		// nothing to do otherwise
	}

	// resolve all go type definitions that we want to produce from this [structural.AnalyzedSchema],
	// possibly with validation setup.
	for schema := range g.makeGenSchema(analyzed, seed) {
		seed.Schemas = append(seed.Schemas, schema)
		seed.Imports = g.deconflictedMergedImports(seed.Imports, schema.Imports)
	}

	return targetModelContext{
		TargetModel: seed,
	}
}

// makeGenSchema produces the data model to generate a go type for a schema.
//
// In some situations, we may have several type definitions to assemble: e.g. enums, interfaces with concrete types...
func (g *Generator) makeGenSchema(analyzed structural.AnalyzedSchema, seed model.TargetModel) iter.Seq[model.TargetSchema] {
	return g.schemaBuilder.GenNamedSchemas(analyzed, seed) // TODO in schema.Builder
}

// shouldStashModel determines if the layout options tell that we should merge several (named) schemas
// into one model (source file).
func (g *Generator) shouldStashModel(genModel model.TargetModel) (analyzers.UniqueID, bool) {
	parent := genModel.RelatedParent
	if parent.String() != "" && g.ModelLayout.Is(settings.ModelLayoutRelatedModelsOneFile) {
		return parent, true
	}

	ultimateParent := genModel.UltimateParent
	if ultimateParent.String() != "" && g.ModelLayout.Is(settings.ModelLayoutAllModelsOneFile) {
		return ultimateParent, true
	}

	return "", false
}

// layoutOptions describes how we prefer to layout code.
type layoutOptions struct {
	SchemaWantsValidation      bool
	SchemaWantsSplitValidation bool
	SchemaWantsTest            bool
}

// layoutOptionsForSchema extracts the file layout options, from global settings
// or from extension overrides at the schema level.
func (g *Generator) layoutOptionsForSchema(analyzed structural.AnalyzedSchema) layoutOptions {
	schemaWantsValidation := g.WantsValidations
	if raw, isOverride := analyzed.GetExtension("x-go-wants-validation", "x-go-validation"); isOverride {
		flag := raw.(bool)
		schemaWantsValidation = flag
	}

	schemaWantsSplitValidation := schemaWantsValidation && g.WantsSplitValidation
	if raw, isOverride := analyzed.GetExtension("x-go-wants-split-validation", "x-go-split-validation"); isOverride {
		flag := raw.(bool)
		schemaWantsSplitValidation = flag
	}

	schemaWantsTest := g.WantsTest
	if raw, isOverride := analyzed.GetExtension("x-go-wants-test", "x-go-test"); isOverride {
		flag := raw.(bool)
		schemaWantsTest = flag
	}

	return layoutOptions{
		SchemaWantsValidation:      schemaWantsValidation,
		SchemaWantsSplitValidation: schemaWantsSplitValidation,
		SchemaWantsTest:            schemaWantsTest,
	}
}

var (
	flagsForTypeDefinition = model.TargetCodeFlags{
		NeedsType:           true,
		NeedsValidation:     true,
		NeedsOnlyValidation: false,
		NeedsTest:           false,
	}

	flagsForTypeValidation = model.TargetCodeFlags{
		NeedsType:           false,
		NeedsValidation:     true,
		NeedsOnlyValidation: true,
		NeedsTest:           false,
	}

	flagsForTypeTest = model.TargetCodeFlags{
		NeedsType:           false,
		NeedsValidation:     true,
		NeedsOnlyValidation: false,
		NeedsTest:           true,
	}

	flagsForValidationTest = model.TargetCodeFlags{
		NeedsType:           false,
		NeedsValidation:     true,
		NeedsOnlyValidation: true,
		NeedsTest:           true,
	}
)

func applyLayoutOptions(input model.TargetCodeFlags, merge model.TargetCodeFlags) (output model.TargetCodeFlags) {
	output.NeedsType = merge.NeedsType
	output.NeedsValidation = input.NeedsValidation && merge.NeedsValidation
	output.NeedsOnlyValidation = merge.NeedsOnlyValidation
	output.NeedsTest = merge.NeedsTest

	return
}

func applyLayoutOptionsToSchemas(schemas []model.TargetSchema, merge model.TargetCodeFlags) []model.TargetSchema {
	output := make([]model.TargetSchema, 0, len(schemas))

	for _, schema := range schemas {
		sch := schema
		sch.TargetCodeFlags = applyLayoutOptions(schema.TargetCodeFlags, merge)
		output = append(output, sch)
	}

	return output
}
