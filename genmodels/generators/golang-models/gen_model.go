package models

import (
	"iter"
	"slices"

	settings "github.com/fredbi/core/genmodels/generators/common-settings"
	"github.com/fredbi/core/jsonschema/analyzers"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

type layoutOptions struct {
	SchemaWantsValidation      bool
	SchemaWantsSplitValidation bool
	SchemaWantsTest            bool
}

type targetModelContext struct {
	genContext
	TargetModel
}

type genContext struct {
	// TODO: not sure we need this wrapper
}

// makeGenModels builds target models from a single analyzed schema.
//
// Each returned [TargetModel] will produce one source file.
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
func (g *Generator) makeGenModels(analyzed structural.AnalyzedSchema) iter.Seq[TargetModel] {
	const (
		// list of possibly generated files from a single analyzed schema
		typeDefinitionModel int = iota
		typeValidationModel
		typeDefinitionTest
		typeValidationTest
	)

	// base data model for this schema
	genModelWithContext := g.makeGenModel(analyzed)
	layout := g.layoutOptionsForSchema(analyzed)
	genModel := genModelWithContext.TargetModel

	if parent, shouldStash := g.shouldStashModel(genModel); shouldStash {
		// case 1: when related models are defined in a single file, merge their schema(s) and imports
		// and stash these. The stash will be collected by the parent.
		//
		// case 2: when all models are defined in a single file, merge. The stash will be collected by the
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

	if foundInStash {
		// previous schemas have stashed their results for reuse by the current schema.
		//
		// We may safely assume that by now, all dependency schemas have been processed, so the
		// stash for this key is no longer updated.
		g.stashMx.Lock()
		stashed, _ := g.stashedSchemas[analyzed.ID]
		genModel.MergeSchemas(stashed.Schemas)
		genModel.StdImports.Merge(stashed.StdImports)
		genModel.Imports.Merge(stashed.Imports)
		delete(g.stashedSchemas, analyzed.ID)
		g.stashMx.Unlock()
	}

	// iterate over possible model variants based on the same genModel.
	// Only layout flags will differ.
	return func(yield func(TargetModel) bool) {
		var targetModel TargetModel
		for _, targetCode := range []int{
			typeDefinitionModel,
			typeValidationModel,
			typeDefinitionTest,
			typeValidationTest,
		} {
			switch targetCode {

			case typeDefinitionModel:
				// we want to produce a file with the type definition
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

// shouldStashModel determines if the layout options tell that we should merge several (named) schemas into one model.
func (g *Generator) shouldStashModel(genModel TargetModel) (analyzers.UniqueID, bool) {
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

func (g *Generator) makeGenModel(analyzed structural.AnalyzedSchema) targetModelContext {
	genModel := TargetModel{
		GenModelOptions: GenModelOptions{
			GenOptions: &g.GenOptions,
		},
		ID:              analyzed.ID,
		Name:            analyzed.Name,
		Package:         g.nameProvider.PackageShortName(analyzed.Path, analyzed),
		PackageLocation: analyzed.Path,                                           // this is relative to the codegen path
		FullPackage:     g.nameProvider.PackageFullName(analyzed.Path, analyzed), // fully qualified package name (e.g. "github.com/fredbi/core/models")
		File:            g.nameProvider.FileName(analyzed.Name, analyzed),
		//StdImports      ImportsMap         // imports from the standard library TODO
		//Imports         ImportsMap         // non-standard imports TODO
		//Schemas []TargetSchema // all the schemas to produce in a single source model file
	}

	switch {
	case g.ModelLayout.Is(settings.ModelLayoutRelatedModelsOneFile):
		if !analyzed.IsRoot() && analyzed.WasAnonymous() {
			parentID := analyzed.Parent().ID
			genModel.RelatedParent = parentID
		}
	case g.ModelLayout.Is(settings.ModelLayoutAllModelsOneFile):
		parentID := analyzed.UltimateParent().ID
		genModel.UltimateParent = parentID
	}

	genModel.Schemas = slices.AppendSeq(genModel.Schemas, g.makeGenSchema(analyzed, genModel))

	// TODO: not sure we need to put this at this level
	if !g.WantsValidations {
		return targetModelContext{
			TargetModel: genModel,
		}
	}

	return g.makeGenModelValidation(analyzed, genModel) // TODO
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

func applyLayoutOptions(input TargetCodeFlags, merge TargetCodeFlags) (output TargetCodeFlags) {
	output.NeedsType = merge.NeedsType
	output.NeedsValidation = input.NeedsValidation && merge.NeedsValidation
	output.NeedsOnlyValidation = merge.NeedsOnlyValidation
	output.NeedsTest = merge.NeedsTest

	return
}

func applyLayoutOptionsToSchemas(schemas []TargetSchema, merge TargetCodeFlags) []TargetSchema {
	output := make([]TargetSchema, 0, len(schemas))

	for _, schema := range schemas {
		sch := schema
		sch.TargetCodeFlags = applyLayoutOptions(schema.TargetCodeFlags, merge)
		output = append(output, sch)
	}

	return output
}
