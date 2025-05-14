package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBCD(t *testing.T) {
	t.Run("with integer", func(t *testing.T) {
		const expected = "12345678900"
		t.Run("should retrieve the original number", checkBCD(expected))
	})

	t.Run("with decimal", func(t *testing.T) {
		const expected = "0.123456789"
		t.Run("should retrieve the original number", checkBCD(expected))
	})

	t.Run("with sign", func(t *testing.T) {
		const expected = "-0.123456789"
		t.Run("should retrieve the original number", checkBCD(expected))
	})

	t.Run("with scientific notation", func(t *testing.T) {
		const expected = "1e-5"
		t.Run("should retrieve the original number", checkBCD(expected))
	})

	t.Run("with scientific notation (2)", func(t *testing.T) {
		const expected = "123.45E-5"
		t.Run("should retrieve the original number", checkBCD(expected))
	})
}

func checkBCD(expected string) func(*testing.T) {
	return func(t *testing.T) {
		input := []byte(expected)
		expectedNibbles := len(input)/2 + len(input)%2
		output := make([]byte, 0, expectedNibbles)
		nibbles := encodeNumberAsBCD(input, output)
		assert.Len(t, nibbles, expectedNibbles)

		outcome := decodeBCDAsNumber(nibbles)
		assert.Equal(t, expected, string(outcome))
	}
}
