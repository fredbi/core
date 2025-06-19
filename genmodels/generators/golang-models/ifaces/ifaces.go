package ifaces

import (
	"iter"

	model "github.com/fredbi/core/genmodels/generators/golang-models/data-model"
	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/dynamic"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

type (
	// NameProvider configures the analysis of JSON schemas by a [structural.Analyzer]
	// to provide deconflicted go-friendly names for identifier, source files and packages.
	//
	// This is implemented by providers.NameProvider and is declared as an interface essentially to facilitate testing.
	NameProvider interface {
		// Schemas
		///////////////////////////////////

		// UniqueSchema provides the [structural.Analyzer] with a method to determine a unique key for a schema name.
		UniqueSchema(name string) structural.Ident

		// NameSchema provides a legit go identifier name for a schema.
		//
		// Example:
		//
		//   "$ref": "#/$defs/schema_name" would produce "SchemaName".
		NameSchema(name string, analyzed structural.AnalyzedSchema) (string, error)

		// DeconflictSchema is called back by the [structural.Analyzer] whenever a name conflicts.
		//
		// Example:
		//
		//   "$ref": "#/$defs/schemaName"
		//
		// would conflict with:
		//
		//   "$ref": "#/$defs/schema_name" (produced "SchemaName")
		//
		// Hence [NameProvider.DeconflictSchema] needs to find an alternative name that does not conflict, for example:
		//
		//   "SchemaName2"
		DeconflictSchema(name string, namespace structural.Namespace) (string, error)

		// Packages
		///////////////////////////////////

		// UniqueSchema provides the [structural.Analyzer] with a method to determine a unique key for a package path.
		UniquePath(path string) structural.Ident

		// NamePackage provides a legit go package path.
		//
		// Example:
		//
		//   "$ref": "#/$defs" would produce "" (root package),
		//
		// but:
		//
		//   "$ref": "schemas/models.json#/$defs" would produce "schema/models"
		NamePackage(path string, analyzed structural.AnalyzedPackage) (string, error)

		// DeconflictPath is called back by the [structural.Analyzer] whenever a path conflicts.
		//
		// Example:
		//   "$ref": "schemas/models.yaml#/$defs"
		//
		// would conflict with:
		//
		//   "$ref": "schemas/models.json#/$defs" (produced "schema/models")
		//
		// Hence [NameProvider.DeconflictPath] needs to find an alternative name that does not conflict, for example:
		//
		//   "schemas/modelsyaml"
		DeconflictPath(name string, namespace structural.Namespace) (string, error)

		// PackageFullName provide a fully qualified package name (e.g. to use in import)
		//
		// "analyzed" is provided for context and is optional.
		PackageFullName(name string, analyzed ...structural.AnalyzedSchema) string

		// PackageShortName provides the short package name (e.g. package alias, to use in "package ..." statements)
		//
		// "analyzed" is provided for context and is optional.
		PackageShortName(name string, analyzed ...structural.AnalyzedSchema) string

		PackageAlias(name string, part int, analyzed ...structural.AnalyzedSchema) string

		// DeconflictAlias deconflicts a package alias
		DeconflictAlias(name string, namespace structural.Namespace) (string, error)

		// Files
		///////////////////////////////////

		// FileName provides a legit go source file name for a given analyzed schema.
		FileName(name string, analyzed structural.AnalyzedSchema) string

		// FileNameForTest is similar to [NameProvider.FileName], but understands that the file must end with "_test"
		FileNameForTest(name string, analyzed structural.AnalyzedSchema) string

		// Extensions
		///////////////////////////////////

		// MapExtension validates all x-... extensions that affect naming rules
		MapExtension(directive string, extension dynamic.JSON) (any, error)

		// Other naming rules

		// NameEnumValue is used to build an identifier (e.g. constant or variable) corresponding to an enum value
		NameEnumValue(index int, enumValue json.Document, analyzed structural.AnalyzedSchema) (string, error)

		// audit and metadata
		///////////////////////////////////

		// SetAuditor equips a provider with an audit trail recording outlet, e.g. [structural.Analyzer]
		SetAuditor(Auditor)

		// SetMarker equips a provider with an extension marker, e.g. [structural.Analyzer]
		SetMarker(Marker)

		// SetAnnotator equips a provider with a metadata annotator, e.g. [structural.Analyzer]
		SetAnnotator(Annotator)
	}

	// SchemaBuilder transforms a [structural.AnalyzedSchema] into one or several [model.TargetSchema] that
	// may be consumed by a template.
	//
	// This is implemented by schema.Builder and is declared as an interface essentially to facilitate testing.
	SchemaBuilder interface {
		GenNamedSchemas(analyzed structural.AnalyzedSchema, seed model.TargetModel) iter.Seq[model.TargetSchema]
		MapExtension(directive string, extension dynamic.JSON) (any, error)
	}

	// PackageBuilder transform a [structural.AnalyzedPackage] into one or several [model.TargetPackage] that
	// may be consumed by a template.
	//
	// This is implemented by schema.Builder and is declared as an interface essentially to facilitate testing.
	PackageBuilder interface {
		GenNamedPackages(analyzed structural.AnalyzedPackage, seed model.TargetPackage) iter.Seq[model.TargetPackage]
		// NOTE: no need for now to map extensions specific to package. x-go-package is already handled by [NameProvider]
	}
)

// EnumNameProvider is a limited interface to address the naming needs of the [Builder].
//
// It is implemented by providers.NameProvider and this interface definition is mainly intended to facilitate testing.
type EnumNameProvider interface {
	// NameEnumValue is used to build an identifier (e.g. constant or variable) corresponding to an enum value
	NameEnumValue(index int, enumValue json.Document, analyzed structural.AnalyzedSchema) (string, error)
}

// Auditor is the interface for types that know how to log an audit trail of naming decisions.
//
// It is implemented by [structural.SchemaAnalyzer]
type Auditor interface {
	LogAudit(structural.AnalyzedSchema, structural.AuditTrailEntry)
	LogAuditPackage(structural.AnalyzedPackage, structural.AuditTrailEntry)
}

// Annotator is the interface for types that know how to alter the metadata of a schema.
//
// It is implemented by [structural.SchemaAnalyzer]
type Annotator interface {
	AnnotateSchema(structural.AnalyzedSchema, structural.Metadata)
}

// Marker is the interface for types that know how to enrich a schema with additional "x-*" extensions.
//
// It is implemented by [structural.SchemaAnalyzer]
type Marker interface {
	MarkSchema(structural.AnalyzedSchema, structural.Extensions)
	MarkPackage(structural.AnalyzedPackage, structural.Extensions)
}

// NameMangler is the interface for types that produce names which follow go rules and conventions.
type NameMangler interface {
	ToGoName(string) string              // produce a legit go exported identifier
	ToGoVarName(string) string           // produce a legit go unexported identifier
	ToGoFileName(string) string          // produce a legit go source code file, without special meaning for the build processs
	ToGoPackageName(string) string       // produce a legit go package name
	ToGoPackagePath(string) string       // produce a legit fully qualified package name
	ToGoPackageAlias(string, int) string // produce a legit go packacke alias formed with multiple path parts

	SpellNumber(string) string // spell numerical values in strings
}
