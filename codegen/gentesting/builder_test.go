package gentest

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	const testPackage = "generated"
	testPackagePath := filepath.Join("fixtures", testPackage)

	g := New(testPackagePath)
	t.Cleanup(g.Cleanup)

	require.NoError(t, g.Build())

	require.Equal(t, testPackage, g.Name())
	require.NotEmpty(t, g.Symbols(SymbolConst))
	require.NotEmpty(t, g.Symbols(SymbolVar))
	require.NotEmpty(t, g.Symbols(SymbolType))
	require.NotEmpty(t, g.Symbols(SymbolFunc))
	require.NotEmpty(t, g.Plugin())
	require.NotEmpty(t, g.Driver())
}
