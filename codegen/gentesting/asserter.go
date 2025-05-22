package gentest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type Asserter struct {
	*Driver
	// TODO
	t *testing.T
}

type AsserterEntry struct {
	Entry
	t *testing.T
}

func (a *Asserter) MustExportType(name string) AsserterEntry {
	var exported Entry
	a.t.Run(fmt.Sprintf("must export type %q", name), func(t *testing.T) {
		value, ok := a.Type(name)
		require.True(t, ok)
		exported = value
	})

	return AsserterEntry{
		Entry: exported,
		t:     a.t,
	}
}
