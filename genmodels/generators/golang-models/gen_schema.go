package models

import (
	"iter"

	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

// makeGenSchema produces the data model to generate a go type for a schema.
//
// In some situations, we may have several type definitions to assemble: e.g. enums, interfaces with concrete types...
func (g *Generator) makeGenSchema(analyzed structural.AnalyzedSchema, model TargetModel) iter.Seq[TargetSchema] {
	if analyzed.IsAnonymous() {
		// TODO
	}

	switch {
	case analyzed.IsScalar():
		return g.makeNamedScalar(analyzed, model)
	case analyzed.IsObject():
		return g.makeNamedObject(analyzed, model)
	case analyzed.IsArray():
		return g.makeNamedArray(analyzed, model)
	case analyzed.IsTuple():
		return g.makeNamedTuple(analyzed, model)
	case analyzed.IsPolymorphic():
		panic("yay")
	case analyzed.IsAnyWithoutValidation():
		panic("yay")
	default:
		panic("yay")
	}

	return nil // TODO
}

func (g *Generator) makeNamedScalar(analyzed structural.AnalyzedSchema, model TargetModel) iter.Seq[TargetSchema] {
	return nil // TODO
}

func (g *Generator) makeNamedObject(analyzed structural.AnalyzedSchema, model TargetModel) iter.Seq[TargetSchema] {
	schema := TargetSchema{
		GenSchemaTemplateOptions: GenSchemaTemplateOptions{
			GenOptions: model.GenOptions,
			//NeedsSerializer bool
			//MarshalMode     MarshalMode
			//JSONLibPath     string
			//Serializer      SerializerSelector
			TargetCodeFlags: model.TargetCodeFlags,
		},
		TypeDefinition: TypeDefinition{
			Metadata:   Metadata{}, // TODO
			Identifier: analyzed.Name,
			//GoType: ,
			/*
				ContainerFlags

				// maps
				Key *ContainerContext

				// maps and slices
				Element *ContainerContext

				// structs & tuples
				Fields []NamedContainerContext

				// interfaces
				Methods []MethodContainerContext // GetX, SetX
				DiscriminatedTypes

				DefaultValue any
			*/
		},
	}
	return nil // TODO
}

func (g *Generator) makeNamedArray(analyzed structural.AnalyzedSchema, model TargetModel) iter.Seq[TargetSchema] {
	return nil // TODO
}

func (g *Generator) makeNamedTuple(analyzed structural.AnalyzedSchema, model TargetModel) iter.Seq[TargetSchema] {
	return nil // TODO
}
