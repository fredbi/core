package structural

// NameProvider is a callback, which knows how to name a schema, given a suggested name and the schema to be named for more context.
type NameProvider func(name string, analyzed AnalyzedSchema) (string, error)

// NameSchema names a schema. It may return an error if the naming operation is impossible.
func (p NameProvider) NameSchema(name string, analyzed AnalyzedSchema) (string, error) {
	return p(name, analyzed)
}

// PackageNameProvider is a callback, which knows how to name a package, given a suggested name and the package to be named for more context.
type PackageNameProvider func(name string, analyzed AnalyzedPackage) (string, error)

// NamePackage names a package. It may return an error if the naming operation is impossible.
func (p PackageNameProvider) NamePackage(name string, analyzed AnalyzedPackage) (string, error) {
	return p(name, analyzed)
}

// UniqueIdentifier knows how to determine a unique [Ident] for a name.
type UniqueIdentifier func(name string) Ident

// UniqueIdentifier yields a unique [Ident] for a name.
func (i UniqueIdentifier) Hash(name string) Ident {
	return i(name)
}

// Deconflicter is a callback, which knows how to find a deconflicted name.
//
// The provided [Namespace] is provided as context.
//
// The [Deconflicter] may return an error if it fails to find a deconflicted solution.
//
// If the [WithBacktrackOnConflicts] option is enabled, the provided [Namespace] implements
// [BacktrackableNamespace] and the callback may backtrack on previous naming or decisions
// to resolve a naming conflict.
type Deconflicter func(name string, namespace Namespace) (string, error)

// SchemaMarker is a callback to inject new extensions into the schema (e.g. "x-go-*" marks)
type SchemaMarker func(analyzed AnalyzedSchema) Extensions

// MarkSchema returns extensions to be merged to the current analyzed schema.
func (m SchemaMarker) MarkSchema(analyzed AnalyzedSchema) Extensions {
	return m(analyzed)
}
