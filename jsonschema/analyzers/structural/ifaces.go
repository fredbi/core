package structural

import (
	"iter"

	"github.com/fredbi/core/jsonschema"
	"github.com/fredbi/core/jsonschema/analyzers"
)

// Analyzer is the interface exposed by a structural analyzer.
type Analyzer interface {
	// Analyze a single JSON schema
	Analyze(jsonschema.Schema) error

	// Analyze an entire collection of JSON schemas
	AnalyzeCollection(jsonschema.Collection) error

	// Bundle the collection of analyzed schemas, reorganizing the namespace.
	// It returns a new intance of the [Analyzer].
	//
	// [BundleOption] s may be added to alter the bundling strategy.
	//
	// [Analyzer.Bundle] may mutate schemas and packages, and invokes callbacks whenever visiting a package or a schema.
	Bundle() (Analyzer, error)

	// AnalyzedSchemas iterates over the analyzed schemas, possibly applying some [Filter]
	//
	// [Filter] s may be added to restrict the scope or defined a specific ordering.
	//
	// [Analyzer.AnalyzedSchemas] does not mutate anything and no callbacks are invoked.
	AnalyzedSchemas(...Filter) iter.Seq[AnalyzedSchema]

	// Number of analyzed schemas, with all inner sub-schemas
	Len() int

	// Namespaces iterates over the namespace of bundled package paths.
	//
	// [Filter] s may be added to restrict the scope or defined a specific ordering.
	//
	// [Analyzer.Namespaces] does not mutate anything and no callbacks are invoked.
	Namespaces(...Filter) iter.Seq[string]

	// Packages iterates over the namespace of bundled packages.
	//
	// It is similar to [Analyzer.Namespaces], but yields a detailed [AnalyzedPackage] instead of just a path.
	//
	// [Filter] s may be added to restrict the scope or defined a specific ordering.
	//
	// [Analyzer.Packages] does not mutate anything and no callbacks are invoked.
	Packages(...Filter) iter.Seq[AnalyzedPackage]

	// SchemaByID yields a single schema, given its unique key ID.
	SchemaByID(analyzers.UniqueID) (AnalyzedSchema, bool)

	// LogAudit inject an audit trail entry into this [AnalyzedSchema].
	//
	// [Analyzer.LogAudit] mutates the internal representation of the [AnalyzedSchema].
	LogAudit(AnalyzedSchema, AuditTrailEntry)
	LogAuditPackage(AnalyzedPackage, AuditTrailEntry)

	// MarkSchema injects extensions into this [AnalyzedSchema].
	//
	// [Analyzer.MarkSchema] mutates the internal representation of the [AnalyzedSchema].
	MarkSchema(AnalyzedSchema, Extensions)

	// MarkPackage injects extensions into this [AnalyzedPackage].
	//
	// [Analyzer.MarkPackage] mutates the internal representation of the [AnalyzedPackage].
	MarkPackage(AnalyzedPackage, Extensions)

	// AnnotateSchema injects metadata into this [AnalyzedSchema].
	//
	// [Analyzer.MarkSchema] mutates the internal representation of the [AnalyzedSchema].
	AnnotateSchema(AnalyzedSchema, Metadata)
}
