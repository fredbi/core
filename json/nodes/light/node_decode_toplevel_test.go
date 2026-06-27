package light

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	lexcodes "github.com/fredbi/core/json/lexers/error-codes"
)

// TestDecodeTopLevel pins the single-document contract of Decode: exactly one top-level value is
// accepted, and the node layer leans on the lexer to reject trailing data and empty input rather than
// silently overwriting the first value (the decode() loop would otherwise last-win).
func TestDecodeTopLevel(t *testing.T) {
	t.Run("accepts a single value", func(t *testing.T) {
		for name, jazon := range map[string]string{
			"object": `{"a":1}`,
			"array":  `[1,2,3]`,
			"scalar": `42`,
			"string": `"x"`,
			"bool":   `true`,
			"null":   `null`,
		} {
			t.Run(name, func(t *testing.T) {
				assert.Equal(t, jazon, decodeDump(t, jazon, DecodeOptions{}))
			})
		}
	})

	t.Run("rejects trailing data / multiple top-level values", func(t *testing.T) {
		for name, jazon := range map[string]string{
			"two scalars":   `1 2`,
			"two objects":   `{"a":1} {"b":2}`,
			"array then num": `[1,2] 3`,
			"two bools":     `true false`,
			"object garbage": `{"a":1} garbage`,
		} {
			t.Run(name, func(t *testing.T) {
				ctx, n := newDecodeCtx(jazon, DecodeOptions{})
				n.Decode(ctx)
				require.Error(t, ctx.L.Err())
			})
		}
	})

	t.Run("rejects empty or whitespace-only input", func(t *testing.T) {
		for name, jazon := range map[string]string{
			"empty":      ``,
			"spaces":     `   `,
			"whitespace": "\n\t ",
		} {
			t.Run(name, func(t *testing.T) {
				ctx, n := newDecodeCtx(jazon, DecodeOptions{})
				n.Decode(ctx)
				require.Error(t, ctx.L.Err())
				assert.ErrorIs(t, ctx.L.Err(), lexcodes.ErrNoData)
			})
		}
	})

	t.Run("tolerates trailing whitespace after a value", func(t *testing.T) {
		assert.Equal(t, `{"a":1}`, decodeDump(t, "{\"a\":1}   \n", DecodeOptions{}))
	})
}
