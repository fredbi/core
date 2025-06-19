package model

import (
	"sort"

	"github.com/fredbi/core/jsonschema/analyzers"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

var _ structural.Namespace = ImportsMap{}

type AliasedImport struct {
	Alias   string
	Name    string
	Package string
	Source  *structural.AnalyzedPackage
}

type ImportsMap struct {
	list      []AliasedImport
	byAlias   map[string]int
	byPackage map[string]int
}

// MakeImportsMap creates a new map for imports, possibly with initial values.
//
// Initial values should not conflict. Any conflict will be ignored during initialization.
func MakeImportsMap(defaults ...AliasedImport) ImportsMap {
	const sensibleAlloc = 10
	allocs := max(sensibleAlloc, len(defaults))

	m := ImportsMap{
		list:      make([]AliasedImport, 0, allocs),
		byAlias:   make(map[string]int, allocs),
		byPackage: make(map[string]int, allocs),
	}

	for _, value := range defaults {
		_ = m.Set(value)
	}

	return m
}

func (m ImportsMap) List() []AliasedImport {
	sorted := make([]AliasedImport, len(m.list))
	copy(sorted, m.list)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Package < sorted[j].Package
	})

	return sorted
}

func (m ImportsMap) Path() string {
	return ""
}

// CheckNoConflict returns false if the [Ident] identifier is in conflict, and true if this namespace slot is free.
func (m ImportsMap) CheckNoConflict(ident structural.Ident) (ok bool) {
	_, found := m.findAlias(string(ident))

	return !found
}

func (m ImportsMap) Backtrack(resolved structural.ConflictMeta) (ok bool) {
	if !m.CheckNoConflict(resolved.Ident) {
		return false
	}

	oldAlias := resolved.ID.String()
	idx, found := m.byAlias[oldAlias]
	if !found {
		return false
	}

	// swap aliases for the pre-existing entry
	newAlias := string(resolved.Ident)
	entry := m.list[idx]
	entry.Alias = newAlias
	m.list[idx] = entry

	return true
}

// Meta yields the [structural.ConflictMeta] associated to an existing [structural.Ident] or false if there is none.
//
// For the namespace formed by [ImportsMap], the identifier is the package alias.
func (m ImportsMap) Meta(ident structural.Ident) (structural.ConflictMeta, bool) {
	element, found := m.findPackage(string(ident))
	if !found {
		return structural.ConflictMeta{}, false
	}

	return structural.ConflictMeta{
		ID:      analyzers.UniqueID(element.Alias),
		Ident:   ident,
		Name:    element.Package,
		Package: element.Source,
	}, true
}

func (m *ImportsMap) Set(aliased AliasedImport) bool {
	if _, found := m.byAlias[aliased.Alias]; found {
		return false
	}
	if _, found := m.byPackage[aliased.Package]; found {
		return false
	}

	m.list = append(m.list, aliased)

	index := len(m.list) - 1
	m.byAlias[aliased.Alias] = index
	m.byPackage[aliased.Package] = index

	return true
}

// MergeDeconflicted merges an [ImportsMap] into the current one.
//
// There are no redundant entries.
//
// If the two maps use a different alias for the same package, the alias remains the one in the base [ImportsMap].
//
// If an alias conflict is detected (same alias, different target), a deconflict function is called to resolve
// the conflict.
func (m ImportsMap) MergeDeconflicted(
	merged ImportsMap,
	deconflict func(string) string,
) ImportsMap {
	for _, element := range merged.List() {
		i, foundAlias := m.byAlias[element.Alias]

		if foundAlias {
			// exact same entry found: skip
			if m.list[i].Package == element.Package {
				continue
			}

			// same alias different package
			done := m.Set(AliasedImport{
				Alias:   deconflict(element.Alias),
				Package: element.Package,
			})

			// works or panic
			assertAliasConflictMustWork(done, element.Alias)

			break
		}

		_, foundPkg := m.byPackage[element.Package]
		if foundPkg {
			// same package found with a different alias: skip
			continue
		}

		// new alias, new package
		_ = m.Set(element)
	}

	return m
}

func (m ImportsMap) findPackage(pkg string) (AliasedImport, bool) {
	idx, ok := m.byPackage[pkg]

	return m.list[idx], ok
}

func (m ImportsMap) findAlias(alias string) (AliasedImport, bool) {
	idx, ok := m.byAlias[alias]

	return m.list[idx], ok
}
