package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	t.Run("should panic on invalid compression level setting", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = New(
				WithCompressionOptions(WithCompressionLevel(12)),
			)
		})
	})

	t.Run("should apply options", func(t *testing.T) {
		opts := []CompressionOption{WithCompressionLevel(4), WithCompressionThreshold(45)}
		o := applyCompressionOptionsWithDefaults(opts)

		assert.Equal(t, 4, o.compressionLevel)
		assert.Equal(t, 45, o.compressionThreshold)
		assert.Nil(t, o.cw, "the compression writer is built lazily, not at option time")
		assert.Nil(t, o.dict, "no preset dictionary by default")

		t.Run("should build the writer lazily and cache it", func(t *testing.T) {
			w := o.compressWriter()
			assert.NotNil(t, w)
			assert.Same(t, w, o.compressWriter(), "cw is cached after first build")
		})

		t.Run("should preserve the compression configuration on reset", func(t *testing.T) {
			// Reset is a no-op on the compression configuration: level, threshold and dict are
			// frozen for the Store's lifetime so a recycled Store keeps its dictionary and stays
			// self-consistent (see compressionOptions.Reset).
			o.Reset()

			assert.Equal(t, 4, o.compressionLevel)
			assert.Equal(t, 45, o.compressionThreshold)
		})
	})
}
