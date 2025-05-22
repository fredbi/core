package gentest

import (
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBootstrap(t *testing.T) {
	const testPackage = "generated"
	index, imported, alias := testParsed(t, testPackage)

	const pluginFile = "plugin.go"
	target := filepath.Join("fixtures", "bootstrapper", pluginFile)
	targetDir := filepath.Dir(target)
	require.NoError(t, os.MkdirAll(targetDir, 0755))
	t.Cleanup(func() {
		_ = os.RemoveAll(targetDir)
	})

	t.Run("should generate bootstrap plugin code", func(t *testing.T) {
		b := newBootStrapper(index, imported, alias, target)

		require.NoError(t, b.Generate())
	})

	t.Run("bootstrap plugin should compile", func(t *testing.T) {
		const objectFile = "plugin.so"
		targetObject := filepath.Join(targetDir, objectFile)
		builder := exec.Command(
			"go",
			"build",
			"-buildmode=plugin",
			"-o", objectFile,
			pluginFile,
		)
		builder.Dir = targetDir
		builder.Stderr = os.Stderr
		builder.Stdout = os.Stdout

		require.NoError(t, builder.Run())
		require.FileExists(t, targetObject)

		t.Run("build should produce a plugin object", func(t *testing.T) {
			info, err := os.Stat(targetObject)
			require.NoError(t, err)
			require.Greater(t, info.Size(), int64(0))
		})

		t.Run("plugin object should load", func(t *testing.T) {
			p, err := plugin.Open(targetObject)
			require.NoError(t, err)

			_, err = p.Lookup("NewModel")
			require.NoError(t, err)
		})
	})
}

func testParsed(t *testing.T, testPackage string) (symbolsIndex, string, string) {
	t.Helper()

	testPackagePath := filepath.Join("fixtures", testPackage)
	p := newParser(testPackagePath)

	require.NoError(t, p.Parse())

	index := p.Symbols()
	require.NotEmpty(t, index)

	imported := p.PackageImportPath()
	require.NotEmpty(t, imported)

	alias := p.PackageName()
	require.NotEmpty(t, alias)

	return index, imported, alias
}
