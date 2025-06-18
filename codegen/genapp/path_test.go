package genapp

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackagePath(t *testing.T) {
	templatesFS := templatesFixture(t)

	t.Run("without module root", func(t *testing.T) {
		t.Run("with absolute output path", func(t *testing.T) {
			cwd, err := os.Getwd()
			require.NoError(t, err)

			t.Run("should determine package path from new folder", func(t *testing.T) {
				const tgt = "target"
				expected := fmt.Sprintf("github.com/fredbi/core/codegen/genapp/%s/ok", tgt)
				wipeTarget(tgt)()
				t.Cleanup(wipeTarget(tgt))

				target := filepath.Join(cwd, tgt, "ok")
				g := New(templatesFS, WithOutputPath(target))

				needsGoMod, err := g.IsGoModRequired()
				require.NoError(t, err)
				require.False(t, needsGoMod)
				pth, err := g.PackagePath()
				require.NoError(t, err)
				assert.Equal(t, expected, pth)

				t.Run(
					"called function should not leave simulated folder behind",
					func(t *testing.T) {
						assert.NoDirExists(t, target)
					},
				)
			})

			t.Run("should determine package path from existing folder", func(t *testing.T) {
				const tgt = "target2"
				expected := fmt.Sprintf("github.com/fredbi/core/codegen/genapp/%s/ok", tgt)
				wipeTarget(tgt)()
				t.Cleanup(wipeTarget(tgt))

				target := filepath.Join(cwd, tgt, "ok")
				require.NoError(t, os.MkdirAll(target, 0o755))

				g := New(templatesFS, WithOutputPath(target))
				needsGoMod, err := g.IsGoModRequired()
				require.NoError(t, err)
				require.False(t, needsGoMod)
				pth, err := g.PackagePath()
				require.NoError(t, err)
				assert.Equal(t, expected, pth)

				t.Run("called function should not remove preexisting folder", func(t *testing.T) {
					assert.DirExists(t, target)
				})
			})
		})

		t.Run("with relative output path", func(t *testing.T) {
			t.Run("should determine package path from new (relative) folder", func(t *testing.T) {
				const tgt = "target3"
				expected := fmt.Sprintf("github.com/fredbi/core/codegen/genapp/%s/ok", tgt)
				wipeTarget(tgt)()
				t.Cleanup(wipeTarget(tgt))

				target := filepath.Join(tgt, "ok")
				g := New(templatesFS, WithOutputPath(target))
				needsGoMod, err := g.IsGoModRequired()
				require.NoError(t, err)
				require.False(t, needsGoMod)
				pth, err := g.PackagePath()
				require.NoError(t, err)
				assert.Equal(t, expected, pth)

				t.Run(
					"called function should not leave simulated folder behind",
					func(t *testing.T) {
						assert.NoDirExists(t, target)
					},
				)
			})

			t.Run("should determine package path from existing folder", func(t *testing.T) {
				const tgt = "target4"
				expected := fmt.Sprintf("github.com/fredbi/core/codegen/genapp/%s/ok", tgt)
				t.Cleanup(wipeTarget(tgt))

				target := filepath.Join(tgt, "ok")
				require.NoError(t, os.MkdirAll(target, 0o755))

				g := New(templatesFS, WithOutputPath(target))
				needsGoMod, err := g.IsGoModRequired()
				require.NoError(t, err)
				require.False(t, needsGoMod)
				pth, err := g.PackagePath()
				require.NoError(t, err)
				assert.Equal(t, expected, pth)

				t.Run("called function should not remove preexisting folder", func(t *testing.T) {
					assert.DirExists(t, target)
				})
			})

			t.Run("should determine package path from relative parent folder", func(t *testing.T) {
				const tgt = "target5"
				expected := fmt.Sprintf("github.com/fredbi/core/%s/ok", tgt)

				target := filepath.Join("..", "..", tgt, "ok")
				t.Cleanup(wipeTarget(target))
				g := New(templatesFS, WithOutputPath(target))
				needsGoMod, err := g.IsGoModRequired()
				require.NoError(t, err)
				require.False(t, needsGoMod)
				pth, err := g.PackagePath()
				require.NoError(t, err)
				assert.Equal(t, expected, pth)

				t.Run(
					"called function should not leave simulated folder behind",
					func(t *testing.T) {
						assert.NoDirExists(t, target)
					},
				)
			})
		})
	})

	t.Run("with module root", func(t *testing.T) {
		const moduleRoot = "goswagger.io/go-openapi"

		t.Run("with module root inside go build tree", func(t *testing.T) {
			t.Run("should determine package path from relative parent folder", func(t *testing.T) {
				const tgt = "target6"
				expected := moduleRoot + "/ok"
				t.Cleanup(wipeTarget(tgt))

				target := filepath.Join("..", "..", tgt, "ok")
				g := New(templatesFS, WithOutputPath(target))
				needsGoMod, err := g.IsGoModRequired()
				require.NoError(t, err)
				require.False(t, needsGoMod)
				pth, err := g.PackagePath(WithModulePath(moduleRoot))
				require.NoError(t, err)
				assert.Equal(t, expected, pth)

				t.Run(
					"called function should not leave simulated folder behind",
					func(t *testing.T) {
						assert.NoDirExists(t, target)
					},
				)
			})
		})

		t.Run("with module root outside go build tree", func(q *testing.T) {
			tmp := os.TempDir()
			root := filepath.Join(tmp, "generated")

			q.Run("should determine package path from new folder", func(t *testing.T) {
				const tgt = "target7"
				expected := moduleRoot + "/ok"
				t.Cleanup(wipeTarget(tgt))

				target := filepath.Join(root, tgt, "ok")
				g := New(templatesFS, WithOutputPath(target))
				needsGoMod, err := g.IsGoModRequired()
				require.NoError(t, err)
				require.True(t, needsGoMod)
				pth, err := g.PackagePath(WithModulePath(moduleRoot))
				require.NoError(t, err)
				assert.Equal(t, expected, pth)

				t.Run(
					"called function should not leave simulated folder behind",
					func(t *testing.T) {
						assert.NoDirExists(t, target)
					},
				)
			})
		})
	})
}

func wipeTarget(target string) func() {
	return func() {
		_ = os.RemoveAll(target)
	}
}
