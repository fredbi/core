package models

import (
	"iter"

	model "github.com/fredbi/core/genmodels/generators/golang-models/data-model"
	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/dynamic"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

type (
	Deconflicter interface{}

	// NameProvider configures the analysis of JSON schemas by a [structural.Analyzer]
	// to provide deconflicted go-friendly names for identifier, source files and packages.
	//
	// This is implemented by [providers.NameProvider] and is declared as an interface essentially to facilitate testing.
	NameProvider interface {
		EqualName(a, b string) bool
		EqualPath(a, b string) bool

		// FileName provides a legit go source file name for a given analyzed schema.
		FileName(name string, analyzed structural.AnalyzedSchema) string

		// FileNameForTest is similar to [NameProvider.FileName], but understands that the file must end with "_test"
		FileNameForTest(name string, analyzed structural.AnalyzedSchema) string

		// MapExtension validates all x-... extensions that affect naming rules
		MapExtension(directive string, extension dynamic.JSON) (any, error)

		// Mark is used as a callback to add extensions dynamically while analyzing the input schemas.
		Mark(analyzed structural.AnalyzedSchema) structural.Extensions

		// Naming rules

		// NameEnumValue is used to build an identifier (e.g. constant or variable) corresponding to an enum value
		NameEnumValue(index int, enumValue json.Document, analyzed structural.AnalyzedSchema) (string, error)

		// NamePackage
		NamePackage(path string, analyzed structural.AnalyzedSchema) (string, error)

		// NameSchema provides a legit go identifier name for a schema.
		NameSchema(name string, analyzed structural.AnalyzedSchema) (string, error)

		// PackageFullName provide a fully qualified package name (e.g. to use in import)
		//
		// "analyzed" is provided for context and is optional.
		PackageFullName(name string, analyzed ...structural.AnalyzedSchema) string

		// PackageShortName provides the short package name (e.g. package alias, to use in "package ..." statements)
		//
		// "analyzed" is provided for context and is optional.
		PackageShortName(name string, analyzed ...structural.AnalyzedSchema) string
	}

	// SchemaBuilder transforms a [structural.AnalyzedSchema] into one or several [model.TargetSchema] that
	// may be consumed by a template.
	//
	// This is implemented by [schema.Builder] and is declared as an interface essentially to facilitate testing.
	SchemaBuilder interface {
		GenNamedSchemas(analyzed structural.AnalyzedSchema, seed model.TargetModel) iter.Seq[model.TargetSchema]
	}

	// PackageBuilder tranform a [structural.AnalyzedPackage] into one or several [model.TargetPackage] that
	// may be consumed by a template.
	//
	// This is implemented by [schema.Builder] and is declared as an interface essentially to facilitate testing.
	PackageBuilder interface {
		GenNamedPackages(analyzed structural.AnalyzedPackage, seed model.TargetPackage) iter.Seq[model.TargetPackage]
	}
)
