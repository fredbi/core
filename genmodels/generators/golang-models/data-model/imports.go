package model

import (
	"slices"
	"strconv"
)

type ImportsMap []AliasedImport

func (m ImportsMap) Merge(merged ImportsMap) ImportsMap {
	if len(merged) == 0 {
		return m
	}

	added := 0
	aliasIndex := make(map[string]string, len(m))

	for _, existing := range m {
		aliasIndex[existing.Alias] = existing.Package
	}
	for _, candidate := range merged {
		aliased, found := aliasIndex[candidate.Alias]
		if !found {
			aliasIndex[candidate.Alias] = candidate.Package
			added++

			continue
		}

		// alias conflict
		if candidate.Package != aliased {
			newAlias := m.deconflictAlias(candidate.Package, aliasIndex)
			aliasIndex[newAlias] = candidate.Package
			added++

			continue
		}

		// identical entries: nothing to do
	}

	result := make(ImportsMap, 0, len(m)+added)
	for alias, pkg := range aliasIndex {
		result = append(result, AliasedImport{Alias: alias, Package: pkg})
	}

	slices.SortFunc(result, func(a, b AliasedImport) int {
		switch {
		case a.Package < b.Package:
			return -1
		case a.Package > b.Package:
			return 1
		default:
			return 0
		}
	})

	return result
}

func (m ImportsMap) deconflictAlias(conflicting string, index map[string]string) (deconflicted string) {
	found := true
	for i := 2; !found; i++ {
		deconflicted = conflicting + strconv.Itoa(i) // TODO: should try first by removing "-"
		_, found = index[deconflicted]
	}

	return deconflicted
}
