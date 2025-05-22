package gentest

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDriver(t *testing.T) {
	const testPackage = "generated"
	testPackagePath := filepath.Join("fixtures", testPackage)

	g := New(testPackagePath)
	t.Cleanup(g.Cleanup)

	require.NoError(t, g.Build())

	d := g.Driver()
	require.NotEmpty(t, d)

	t.Run("assert Model type", func(t *testing.T) {
		model, ok := d.Type("Model")
		require.True(t, ok)

		require.True(t, model.IsStruct())
		require.True(t, model.HasField("A"))

		fieldA := model.Field("A")
		require.False(t, fieldA.IsEmpty())

		t.Logf("type A: %v", fieldA.Value().Type())
	})
}
