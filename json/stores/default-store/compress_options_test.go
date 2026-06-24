package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	t.Run("should panic on invalid compression level setting", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = DefaultOptions().WithCompressionLevel(12)
		})
	})

	t.Run("should apply options", func(t *testing.T) {
		o := DefaultOptions().
			WithCompressionLevel(4).
			WithCompressionThreshold(45).
			resolved

		assert.Equal(t, 4, o.compressionLevel)
		assert.Equal(t, 45, o.compressionThreshold)
		assert.Nil(t, o.cw, "the compression writer is built lazily, not at option time")
		assert.Nil(t, o.dict, "no preset dictionary by default")

		t.Run("should build the writer lazily and cache it", func(t *testing.T) {
			w := o.compressWriter()
			assert.NotNil(t, w)
			assert.Same(t, w, o.compressWriter(), "cw is cached after first build")
		})
	})

	t.Run("builder is immutable: each With* returns a fresh copy", func(t *testing.T) {
		base := DefaultOptions()
		derived := base.WithCompressionLevel(9)

		assert.Equal(t, defaultCompressionLevel, base.resolved.compressionLevel, "the base must be unchanged")
		assert.Equal(t, 9, derived.resolved.compressionLevel)
	})
}
