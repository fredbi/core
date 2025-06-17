package genapp

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	repo "github.com/fredbi/core/codegen/templates-repo"
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

func TestGoGenAppRender(t *testing.T) {
	const (
		folder        = "generated"
		testGenerated = "output.go"
	)

	t.Run("should generate from embedded templates", func(t *testing.T) {
		templatesFS := templatesFixture(t)
		testFS := afero.NewMemMapFs()
		require.NoError(t, testFS.MkdirAll(folder, 0750))

		g := New(
			templatesFS,
			WithOutputAferoFS(testFS),
			WithOutputPath(folder),
		)

		check := t.Name()
		data := makeTestData(check)

		t.Run("should render test template", func(t *testing.T) {
			require.NoError(t, g.Render("example", testGenerated, data))
		})

		t.Run("a target file should be present", func(t *testing.T) {
			result, err := afero.ReadFile(testFS, filepath.Join(folder, testGenerated))
			require.NoError(t, err)

			assertExample(t, check, result)
		})
	})

	t.Run("should generate from local templates", func(t *testing.T) {
		localFS := fsutils.NewReadOnlyOsFS()
		templatesLocation := filepath.Join("fixtures", "templates")
		templatesFS, err := fs.Sub(localFS, templatesLocation)
		require.NoError(t, err)

		tmpDir := makeTestDir(t)
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, folder), 0755))

		g := New(
			templatesFS,
			WithOutputPath(tmpDir),
		)

		target := filepath.Join(folder, testGenerated)
		check := t.Name()
		data := makeTestData(check)

		t.Run("should render test template", func(t *testing.T) {
			require.NoError(t, g.Render("example", target, data))
		})

		t.Run("a target file should be present", func(t *testing.T) {
			expectedTarget := filepath.Join(tmpDir, target)
			require.FileExists(t, expectedTarget)

			result, err := os.ReadFile(expectedTarget)
			require.NoError(t, err)

			assertExample(t, check, result)

			info, err := os.Stat(expectedTarget)
			require.NoError(t, err)

			perms := info.Mode() & os.ModePerm

			t.Run("permissions on written file should not be too lax", func(t *testing.T) {
				const (
					allWritePerms = os.FileMode(0004)
					allReadPerms  = os.FileMode(0002)
				)
				require.Less(t, perms&allWritePerms, perms)
				require.LessOrEqual(t, perms&allReadPerms, allReadPerms)
			})

			t.Run("should create go.mod file", func(t *testing.T) {
				const goVersion = "1.23"
				require.NoError(t, g.GoMod(WithGoVersion(goVersion)))

				t.Run("a go mod file should be created", func(t *testing.T) {
					modFile := filepath.Join(tmpDir, "go.mod")
					require.FileExists(t, modFile)

					t.Run("go mod should require expected go version", func(t *testing.T) {
						content, err := os.ReadFile(modFile)
						require.NoError(t, err)

						require.Contains(t, string(content), "go "+goVersion)
					})
				})
			})
		})
	})

	t.Run("should apply overlay template", func(t *testing.T) {
		t.Skip()
		// TODO
	})
}

func TestTemplates(t *testing.T) {
	templatesFS := templatesFixture(t)
	g := New(templatesFS)

	repo := g.Templates()
	require.NotNil(t, repo)

	require.NoError(t, repo.Dump(io.Discard))
}

func TestInvalidOptions(t *testing.T) {
	t.Run("this invalid setup should panic", func(t *testing.T) {
		require.Panics(t, func() {
			_ = New(nil)
		})
	})
	t.Run("but this one is valid and should not panic", func(t *testing.T) {
		templatesFS := templatesFixture(t)
		require.Panics(t, func() {
			_ = New(nil, WithTemplatesRepo(
				repo.New(repo.WithFS(templatesFS)),
			))
		})
	})
}

func makeTestData(args ...any) testData {
	d := testData{
		A: 1,
		B: "test",
	}

	if len(args) > 0 {
		asString, ok := args[0].(string)
		if ok {
			d.F = asString
		}
	}

	return d
}

func templatesFixture(t *testing.T) fs.FS {
	location := filepath.Join("fixtures", "templates")
	const templateName = "example.gotmpl"
	require.NotNil(t, fixturesFS)

	templatesFS, err := fs.Sub(fixturesFS, location)
	require.NoError(t, err)

	t.Run(fmt.Sprintf("embed fixtures in %q should be configured as expected", location), func(t *testing.T) {
		t.Helper()

		var found bool
		require.NoError(t, fs.WalkDir(templatesFS, ".", func(path string, _ fs.DirEntry, err error) error {
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

func assertExample(t *testing.T, check string, result []byte) {
	t.Run("generated output should match expected content", func(t *testing.T) {
		expectedCode := strings.TrimLeftFunc(strings.ReplaceAll(`
// generated file do not edit

// Package generated is a test produced by a generator
package generated

import (
	"fmt"
)

func X() string {
	a := 1
	b := test

	f := "<<check>>"

	return fmt.Sprintf("a=%d, b=%d, f=%s", a, b, f) // verify if gofmt
}
`, "<<check>>", check), trimCR)

		assert.Equal(t, expectedCode, string(result))
	})
}

func trimCR(r rune) bool {
	return r == '\n' || r == '\r'
}

func makeTestDir(t *testing.T) string {
	dir, err := os.MkdirTemp(".", "gentest") //nolint:usetesting  // cannot use t.TempDir() because we want to remain inside the go source tree
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	return dir
}
