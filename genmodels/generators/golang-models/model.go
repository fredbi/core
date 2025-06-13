package models

import (
	"iter"

	settings "github.com/fredbi/core/genmodels/generators/common-settings"
	model "github.com/fredbi/core/genmodels/generators/golang-models/data-model"
	"github.com/fredbi/core/jsonschema/analyzers"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

type targetModelContext struct {
	model.TargetModel

	genContext
}

type genContext struct {
	// currently unused.
	// TODO: not sure we need this wrapper
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
	const (
		// list of possibly generated files from a single analyzed schema
		typeDefinitionModel int = iota
		typeValidationModel
		typeDefinitionTest
		typeValidationTest
	)

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

		stashed, foundInStash := g.stashedSchemas[analyzed.ID]
		if foundInStash {
			genModel.MergeSchemas(stashed.Schemas)
			genModel.StdImports.Merge(stashed.StdImports)
			genModel.Imports.Merge(stashed.Imports)
		}

		// stash this model for later generation
		g.stashedSchemas[parent] = genModel

		return nil // skip this generation target
	}

	g.stashMx.RLock()
	_, foundInStash := g.stashedSchemas[analyzed.ID]
	g.stashMx.RUnlock()

	// 4. determine if we should merge previously stashed models
	if foundInStash {
		// previous schemas have stashed their results for reuse by the current schema.
		//
		// We may safely assume that by now, all dependency schemas have been processed, so the
		// stash for this key is no longer updated.
		g.stashMx.Lock()
		stashed, _ := g.stashedSchemas[analyzed.ID]
		genModel.MergeSchemas(stashed.Schemas) // TODO: should sort dependencies to reflect nice code, with dependencies last
		genModel.StdImports.Merge(stashed.StdImports)
		genModel.Imports.Merge(stashed.Imports)
		delete(g.stashedSchemas, analyzed.ID)
		g.stashMx.Unlock()
	}

	// 5. yields the iterator over model variants based on the same genModel.
	//
	// Produced models will differ only by their target file name and layout flags.
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

			case typeValidationTest:
				// we want to produce a test for the Validate method only
				if !layout.SchemaWantsSplitValidation || !layout.SchemaWantsTest || !targetModel.NeedsValidation {
					// no specific file: skip
					continue
				}

				// yield a version of this [TargetModel] to generate {name}_validation_test.go, with only tests for validation code
				targetModel = genModel
				targetModel.File = g.nameProvider.FileNameForTest(targetModel.File+"_validation_test", analyzed)
				targetModel.TargetCodeFlags = applyLayoutOptions(genModel.TargetCodeFlags, flagsForValidationTest)
				targetModel.Schemas = applyLayoutOptionsToSchemas(targetModel.Schemas, flagsForValidationTest)
			}

			if !yield(targetModel) {
				return
			}
		}
	}
}

func (g *Generator) makeGenModel(analyzed structural.AnalyzedSchema) targetModelContext {
	const sensibleAllocs = 10

	genModel := model.TargetModel{
		GenModelOptions: model.GenModelOptions{
			GenOptions: &g.GenOptions,
		},
		ID:   analyzed.ID,
		Name: analyzed.OriginalName(),
		LocationInfo: model.LocationInfo{
			Package:         g.nameProvider.PackageShortName(analyzed.Path(), analyzed),
			PackageLocation: analyzed.Path(),                                           // this is relative to the codegen path
			FullPackage:     g.nameProvider.PackageFullName(analyzed.Path(), analyzed), // fully qualified package name (e.g. "github.com/fredbi/core/models")
			File:            g.nameProvider.FileName(analyzed.Name(), analyzed),        // deconflicted file name, safe for go
		},
		StdImports: make(model.ImportsMap, 0, sensibleAllocs),
		Imports:    make(model.ImportsMap, 0, sensibleAllocs),
		Schemas:    make([]model.TargetSchema, 0, sensibleAllocs),
	}

	switch {
	case g.ModelLayout.Is(settings.ModelLayoutRelatedModelsOneFile):
		// in this layout (the default), a single source file is produced for a type and the related ones.
		// (e.g. types that define the elements of a slice or a map).
		//
		// When a RelatedParent is defined, the model will be stashed for later merge and rendering in the same file
		// as its RelatedParent.
		if !analyzed.IsRoot() && analyzed.WasAnonymous() {
			parentID := analyzed.Parent().ID
			genModel.RelatedParent = parentID
		}
	case g.ModelLayout.Is(settings.ModelLayoutAllModelsOneFile):
		// in this layout, all models are packed in a single source file
		//
		// When an UltimateParent is defined, the model with be stashed for later merge and rendering in a single file.
		parentID := analyzed.UltimateParent().ID
		genModel.UltimateParent = parentID
	}

	// resolve all go type definitions that we want to produce from this [structural.AnalyzedSchema],
	// possibly with validation setup.
	for schema := range g.makeGenSchema(analyzed, genModel) {
		genModel.Schemas = append(genModel.Schemas, schema)
		genModel.StdImports.Merge(schema.StdImports)
		genModel.Imports.Merge(schema.Imports)
	}

	return targetModelContext{
		TargetModel: genModel,
	}
}

// makeGenSchema produces the data model to generate a go type for a schema.
//
// In some situations, we may have several type definitions to assemble: e.g. enums, interfaces with concrete types...
func (g *Generator) makeGenSchema(analyzed structural.AnalyzedSchema, seed model.TargetModel) iter.Seq[model.TargetSchema] {
	return g.schemaBuilder.GenNamedSchemas(analyzed, seed)
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
