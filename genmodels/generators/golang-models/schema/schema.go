package schema

import (
	"iter"

	model "github.com/fredbi/core/genmodels/generators/golang-models/data-model"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

type genSchemaFlag uint8

const (
	genSchemaFlagIgnoreNamed genSchemaFlag = 1
	genSchemaFlagPickNamed   genSchemaFlag = 2
)

// GenNamedSchemas transforms a [structural.AnalyzedSchema] into a series of [model.TargetSchema] s to be consumed by
// templates.
//
// The {model.TargetModel] is needed to seed all options and settings.
func (g *Builder) GenNamedSchemas(
	analyzed structural.AnalyzedSchema,
	seed model.TargetModel,
) iter.Seq[model.TargetSchema] {
	assertNamedSchema(analyzed)

	return g.makeGenNamedSchema(
		analyzed,
		seed,
		genSchemaFlagPickNamed,
	) // at the topmost level, pick a named schema.
}

func (g *Builder) makeGenNamedSchema(
	analyzed structural.AnalyzedSchema,
	seed model.TargetModel,
	flag genSchemaFlag,
) iter.Seq[model.TargetSchema] {
	if !analyzed.IsAnonymous() && (flag&genSchemaFlagIgnoreNamed > 0) {
		// assume this named schema is handled independently
		return nil
	}

	switch {
	case analyzed.IsScalar():
		// a scalar type, possibly nullable. So this must be "string", "number", "integer" or a multi-type
		// definition such as ["string", "null"].
		//
		// This may return several schemas if we have an enum validation.
		return g.makeNamedScalar(analyzed, seed, flag)
	case analyzed.IsObject():
		// an object, possibly without properties, possibly with implicit properties,
		// possibly with an inner "allOf" declaration, possibly with additionalProperties
		return g.makeNamedObject(analyzed, seed, flag)
	case analyzed.IsArray():
		// an array (and not a tuple), possibly with no constraint on its items;
		// possibly with an "allOf" declaration.
		return g.makeNamedArray(analyzed, seed, flag)
	case analyzed.IsTuple():
		// a tuple (and not an array), possibly with additionalItems
		return g.makeNamedTuple(analyzed, seed, flag)
	case analyzed.IsPolymorphic():
		// a polymorphic type resulting either from a "type": [ object, array ] or from an "oneOf", "anyOf"
		// declaration.
		panic("yay")
	case analyzed.IsAnyWithoutValidation():
		// an unconstrained type, mapped to "any"
		panic("yay")
	default:
		// TODO: assertion
		panic("yay")
	}
}

// makeNamedScalar builds a go type for a scalar JSON schema.
//
// Available mapping strategies:
// - for strings:
//  1. type definition
//  2. with format validation:
//
// analyzed:
//
//	type: string
//
// =>
// type Analyzed string
func (g *Builder) makeNamedScalar(
	analyzed structural.AnalyzedSchema,
	seed model.TargetModel,
	flag genSchemaFlag,
) iter.Seq[model.TargetSchema] {
	/*
		var goType string
		if ext, isUserDefined := analyzed.GetExtension("x-go-type"); isUserDefined {
		}

		if analyzed.HasEnum() {
			// produce enum type definition
		}

		switch analyzed.ScalarKind() {
		case analyzers.ScalarKindString:
			if analyzed.HasFormatValidation() {

			}
			goType = "string"
		case analyzers.ScalarKindNumber:
			if analyzed.HasFormatValidation() {
			}
			goType = g.DefaultDecimalGoType

		case analyzers.ScalarKindInteger:
			if analyzed.HasFormatValidation() {
			}

			goType = g.DefaultIntegerGoType
		case analyzers.ScalarKindBool:
			goType = "bool"
		case analyzers.ScalarKindNull:
			// not supported
			fallthrough
		default:
			panic("yay")
		}
	*/
	if !seed.WantsValidations {
		// validations wanted
	}

	return nil // TODO
}

// initializeTargetSchemaFromModel build a baseline [TargetSchema] from the parent [TargetModel].
func (g *Builder) initializeTargetSchemaFromModel(m model.TargetModel) model.TargetSchema {
	const modelTemplate = "model" // we just need this top-level entry in the templates repository
	return model.TargetSchema{
		GenSchemaTemplateOptions: model.GenSchemaTemplateOptions{
			GenOptions: m.GenOptions,
			// NeedsSerializer bool
			// MarshalMode
			// JSONLibPath     string
			// Serializer      SerializerSelector
			TargetCodeFlags: m.TargetCodeFlags,
		},
		/*
			LocationInfo: LocationInfo{
				Template: modelTemplate,
			},
		*/
		TypeDefinition: model.TypeDefinition{
			Metadata: model.Metadata{}, // TODO
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
}

func (g *Builder) makeNamedObject(
	analyzed structural.AnalyzedSchema,
	model model.TargetModel,
	flag genSchemaFlag,
) iter.Seq[model.TargetSchema] {
	schema := g.initializeTargetSchemaFromModel(model)
	schema.Identifier = analyzed.Name()
	// schema.Metadata =  /* TODO */
	/*
		schema.Target TargetSchema{
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
				*
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
				*
			},
		}
	*/
	return nil // TODO
}

func (g *Builder) makeNamedArray(
	analyzed structural.AnalyzedSchema,
	m model.TargetModel,
	flag genSchemaFlag,
) iter.Seq[model.TargetSchema] {
	/*
		children := analyzed.Children()
		if len(children) > 1 {
			// error
			panic("yay")
		}

		items := g.makeGenSchema(children[0], m, genSchemaFlagIgnoreNamed)

		if g.SliceType != "" && g.SliceType != "[]" {
			// custom generic slice
		}

		goType := "[]" + items.GoType
	*/

	return nil // TODO
}

func (g *Builder) makeNamedTuple(
	analyzed structural.AnalyzedSchema,
	model model.TargetModel,
	flag genSchemaFlag,
) iter.Seq[model.TargetSchema] {
	return nil // TODO
}
