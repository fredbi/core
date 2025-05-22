package gentest

import (
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	const testPackage = "generated"
	testPackagePath := filepath.Join("fixtures", testPackage)
	p := newParser(testPackagePath)

	t.Run("should parse test package", func(t *testing.T) {
		require.NoError(t, p.Parse())
	})

	assert.Equal(t, testPackage, p.PackageName())
	expectedPackagePath := path.Join("github.com", "fredbi", "core", "codegen", "gentesting", "fixtures", testPackage)
	assert.Equal(t, expectedPackagePath, p.PackageImportPath())

	t.Run("should identify exported symbols by kind", func(t *testing.T) {
		s := p.Symbols()

		require.NotEmpty(t, s)
		require.NotEmpty(t, s[SymbolConst])
		require.NotEmpty(t, s[SymbolVar])
		require.NotEmpty(t, s[SymbolType])
		require.NotEmpty(t, s[SymbolFunc])

		t.Logf("%v", s)
	})
}
