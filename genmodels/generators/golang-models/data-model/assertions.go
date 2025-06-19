package model

import "fmt"

func assertAliasConflictMustWork(done bool, name string) {
	if !done {
		panic(fmt.Errorf(
			"the package alias deconflicter should always manage to find a deconficted alias. Failed doing so for alias %q: %w",
			name,
			ErrInternal,
		))
	}
}
