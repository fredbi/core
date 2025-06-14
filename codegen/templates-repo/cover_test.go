package repo

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstrumentation(t *testing.T) {
	t.Run("with repository from local FS", func(t *testing.T) {
		r := New(
			WithFuncMap(testFuncMap()),
			WithCoverProfile(true),
		)

		cwd, _ := os.Getwd()
		fixtures := filepath.Join(cwd, "fixtures/templates")

		t.Run("should load templates", func(t *testing.T) {
			require.NoError(t, r.Load(fixtures))
		})

		t.Run("instrumented template should execute", func(t *testing.T) {
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
			t.Log(output)

			require.NoError(t, r.FlushCoverProfile("covertmpl.out"))
		})
	})
}
