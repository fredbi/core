package repo

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	r := New(
		WithFuncMap(testFuncMap()),
		WithParseComments(true),
	)

	cwd, _ := os.Getwd()
	fixtures := filepath.Join(cwd, "templates/fixtures")
	require.NoError(t, r.Load(fixtures))

	buf := bytes.NewBuffer(nil)
	r.Dump(buf)
	t.Log(buf.String())
}

func testFuncMap() template.FuncMap {
	return template.FuncMap{
		"comment":   func(s string) string { return "// " + s },
		"imports":   func() string { return "" },
		"pascalize": strings.ToTitle,
		"humanize":  strings.ToLower,
	}
}
