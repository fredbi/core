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
		WithCoverProfile(true),
	)

	cwd, _ := os.Getwd()
	fixtures := filepath.Join(cwd, "templates/fixtures")
	require.NoError(t, r.Load(fixtures))

	buf := bytes.NewBuffer(nil)
	r.Dump(buf)
	t.Log(buf.String())

	buf.Reset()
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
