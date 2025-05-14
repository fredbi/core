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
		assert.NotEmpty(t, o.cw)
		assert.NotNil(t, o.dict)

		t.Run("should reset to defaults", func(t *testing.T) {
			o.Reset()

			assert.Equal(t, defaultCompressionLevel, o.compressionLevel)
			assert.Equal(t, defaultCompressionThreshold, o.compressionThreshold)
			assert.NotEmpty(t, o.cw)
			assert.NotNil(t, o.dict)
		})
	})
}
