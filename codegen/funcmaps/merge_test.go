package funcmaps

import (
	"reflect"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
)

func TestMerge(t *testing.T) {
	x := func(string) string { return "" }
	y := func(string) (string, error) { return "", nil }
	z := func(any) (string, error) { return "", nil }

	a := template.FuncMap{
		"x": x,
	}
	b := template.FuncMap{
		"x": y,
		"y": y,
		"z": z,
	}

	t.Run("should merge maps", func(t *testing.T) {
		c := Merge(a, b)

		require.Len(t, c, 3)
		require.Contains(t, c, "x")
		require.Contains(t, c, "y")
		require.Contains(t, c, "z")

		require.Equal(t, reflect.ValueOf(y).UnsafePointer(), reflect.ValueOf(c["x"]).UnsafePointer())
		require.Equal(t, reflect.ValueOf(y).UnsafePointer(), reflect.ValueOf(c["y"]).UnsafePointer())
		require.Equal(t, reflect.ValueOf(z).UnsafePointer(), reflect.ValueOf(c["z"]).UnsafePointer())
	})
}
