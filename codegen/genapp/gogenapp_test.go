package genapp

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	fsutils "github.com/fredbi/core/swag/fs"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed fixtures/templates
var fixturesFS embed.FS

type testData struct {
	A int
	B string
	F string
}

func (d testData) FileName() string {
	return d.F
}

func TestGoGenApp(t *testing.T) {
	const (
		folder        = "generated"
		testGenerated = "output.go"
	)

	t.Run("should generate from embedded templates", func(t *testing.T) {
		templatesFS := templatesFixture(t)
		testFS := afero.NewMemMapFs()
		require.NoError(t, testFS.MkdirAll(folder, 0750))

		g := New(
			WithTemplates(templatesFS),
			WithOutputAferoFS(testFS),
			WithOutputPath(folder),
		)

		data := testData{
			A: 1,
			B: "test",
			F: testGenerated,
		}

		require.NoError(t, g.Render("example", data))
		result, err := afero.ReadFile(testFS, filepath.Join(folder, testGenerated))
		require.NoError(t, err)

		assertExample(t, result)

		info, err := testFS.Stat(filepath.Join(folder, testGenerated))
		require.NoError(t, err)
		t.Logf("%v", fs.FormatFileInfo(info))
	})

	t.Run("should generate from local templates", func(t *testing.T) {
		localFS := fsutils.NewReadOnlyOsFS()
		templatesLocation := filepath.Join("fixtures", "templates")
		templatesFS, err := fs.Sub(localFS, templatesLocation)
		require.NoError(t, err)

		tmpDir := t.TempDir()
		os.MkdirAll(filepath.Join(tmpDir, folder), 0755)
		g := New(
			WithTemplates(templatesFS),
			WithOutputPath(folder),
		)

		data := testData{
			A: 1,
			B: "test",
			F: testGenerated,
		}

		require.NoError(t, g.Render("example", data))
		result, err := os.ReadFile(filepath.Join(folder, testGenerated))
		require.NoError(t, err)

		assertExample(t, result)

		info, err := os.Stat(filepath.Join(folder, testGenerated))
		require.NoError(t, err)
		t.Logf("%v", fs.FormatFileInfo(info))
	})
}

func templatesFixture(t *testing.T) fs.FS {
	location := filepath.Join("fixtures", "templates")
	const templateName = "example.gotmpl"
	require.NotNil(t, fixturesFS)
	templatesFS, err := fs.Sub(fixturesFS, location)
	require.NoError(t, err)

	t.Run(fmt.Sprintf("embed fixtures in %q should be configured as expected", location), func(t *testing.T) {
		var found bool
		require.NoError(t, fs.WalkDir(templatesFS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if path == templateName {
				found = true
			}
			return nil
		}))
		require.True(t, found)
		f, err := templatesFS.Open(templateName)
		require.NoError(t, err)
		_ = f.Close()
	})

	return templatesFS
}

func assertExample(t *testing.T, result []byte) {
	t.Run("generated output should ", func(t *testing.T) {
		assert.Equal(t, `test:
a = 1
b = test
`, string(result),
		)
	})
}
