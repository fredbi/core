package repo

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed fixtures/*
var fixturesFS embed.FS

func TestLoad(t *testing.T) {
	t.Run("with repository from local FS", func(t *testing.T) {
		r := New(
			WithFuncMap(testFuncMap()),
			WithParseComments(true),
			// WithCoverProfile(true),
		)

		cwd, _ := os.Getwd()
		fixtures := filepath.Join(cwd, "fixtures/templates")

		t.Run("should load templates", func(t *testing.T) {
			require.NoError(t, r.Load(fixtures))
		})

		t.Run("should dump templates structure", func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			require.NoError(t, r.Dump(buf))

			output := buf.String()
			require.NotEmpty(t, output)

			t.Run("should report structure as markdown", func(t *testing.T) {
				assert.Contains(t, output, "# Templates")
				assert.Contains(t, output, "## docstring")
				assert.Contains(t, output, "Defined in docstring.gotmpl")
				assert.Contains(t, output, "## empty")
				assert.Contains(t, output, "Defined in empty.gotmpl")
			})

			t.Run("should report dependencies", func(t *testing.T) {
				assert.Regexp(t, `#### Requires[\n\s]+\- annotations[\n\s+]\- docstring`, output)
			})

			t.Run("should have resolved comments as docstrings", func(t *testing.T) {
				assert.Contains(t, output, "docstring generate comments from a schema's Title and Description")
			})
		})

		t.Run("should fetch top-level template", func(t *testing.T) {
			t.Run("should work with Get", func(t *testing.T) {
				tpl, err := r.Get("docstring")
				require.NoError(t, err)
				require.NotNil(t, tpl)
			})

			t.Run("template should execute", func(t *testing.T) {
				buf := bytes.NewBuffer(nil)
				require.NoError(t, r.MustGet("docstring").Execute(buf, struct {
					Title         string
					Description   string
					Name          string
					Example       any
					MinProperties int
					MaxProperties int
				}{
					Title:       "x",
					Description: "This is a x",
					Name:        "a variable name",
					Example:     1.45,
				}))
				output := buf.String()
				require.NotEmpty(t, output)

				t.Run("output should match expectations", func(t *testing.T) {
					assert.Contains(t, output, `// x`)
					assert.Contains(t, output, `// This is a x`)
					assert.Contains(t, output, `// Example: 1.45`)
					assert.NotContains(t, output, `a variable`)
				})
			})
		})

		t.Run("should fetch template in folder", func(t *testing.T) {
			tpl, err := r.Get("folderHeader")
			require.NoError(t, err)
			require.NotNil(t, tpl)
		})

		t.Run("should error on non-existing template", func(t *testing.T) {
			_, err := r.Get("folderDocString")
			require.Error(t, err)
		})
	})

	t.Run("with repository from embed FS", func(t *testing.T) {
		templatesFS, err := fs.Sub(fixturesFS, "fixtures/templates")
		require.NoError(t, err)

		r := New(
			WithFS(templatesFS),
			WithFuncMap(testFuncMap()),
		)

		t.Run("should load templates", func(t *testing.T) {
			require.NoError(t, r.Load("."))
		})

		t.Run("should fetch template in folder", func(t *testing.T) {
			tpl, err := r.Get("folderHeader")
			require.NoError(t, err)
			require.NotNil(t, tpl)
		})
	})
}

func testFuncMap() template.FuncMap {
	return template.FuncMap{
		"comment":   func(s string) string { return "// " + s },
		"imports":   func() string { return "" },
		"pascalize": strings.ToTitle,
		"humanize":  strings.ToLower,
	}
}
