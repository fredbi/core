package structural

import "github.com/fredbi/core/jsonschema/analyzers"

// Ident is a unique string representing a non-conflicting identifier.
//
// If Ident(a) == Ident(b), a name conflict is detected.
type Ident string

// ConflictMeta brings additional contextual information about a unique identifier in a namespace.
//
// The ID refers to the unique ID of the [AnalyzedPackage] or [AnalyzedSchema] that holds the identifier.
// The Metadata section is free for external callbacks to use to evaluate their name deconflicting resolution (e.g. ranking etc).
type ConflictMeta struct {
	Ident Ident
	Name  string
	ID    analyzers.UniqueID

	Schema  *AnalyzedSchema
	Package *AnalyzedPackage

	Metadata any
}

// Namespace represents a set of unique identifiers under the same package path.
type Namespace interface {
	// Path yields the package path of this namespace
	Path() string

	// CheckNoConflict returns false if the [Ident] identifier is in conflict, and true if this namespace slot is free.
	CheckNoConflict(ident Ident) bool

	// Meta yields the [ConflictMeta] associated to an existing [Ident] or false if there is none.
	//
	// Notice that mutating the returned structure won't have any effect on the [Namespace].
	Meta(ident Ident) (ConflictMeta, bool)
}

// BacktrackableNamespace is a [Namespace] that knows how to backtrack on previous naming decisions,
// in effect renaming schemas or packages on second-thought.
type BacktrackableNamespace interface {
	Namespace

	// Backtrack reaffects a new name to the object referred to by this id.
	//
	// It returns true if successful, false if backtracking resulted in a name conflict.
	Backtrack(resolved ConflictMeta) bool
}

type namespace struct {
	path    string
	entries map[Ident]ConflictMeta
}

func (n namespace) Path() string {
	return n.path
}

func (n namespace) Meta(ident Ident) (meta ConflictMeta, ok bool) {
	meta, ok = n.entries[ident]

	return meta, ok
}

func (n namespace) CheckNoConflict(ident Ident) (ok bool) {
	_, ok = n.entries[ident]

	return !ok
}

// register a new identifier with metadata if not conflicting.
func (n *namespace) register(meta ConflictMeta) (ok bool) {
	ident := meta.Ident
	_, ok = n.entries[ident]
	if ok {
		return false
	}

	n.entries[ident] = meta

	return true
}

func (n *namespace) backtracker(analyzer *SchemaAnalyzer) BacktrackableNamespace {
	return &backtrackable{
		namespace: n,
		analyzer:  analyzer,
	}
}

type backtrackable struct {
	*namespace
	analyzer *SchemaAnalyzer
}

func (b *backtrackable) Backtrack(resolved ConflictMeta) (ok bool) {
	if !b.register(resolved) {
		return false
	}

	return b.analyzer.rename(resolved.ID, resolved.Name)
}

func (a *SchemaAnalyzer) rename(id analyzers.UniqueID, newName string) (ok bool) {
	// rename a package or an object
	return false // TODO
}
