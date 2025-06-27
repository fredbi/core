package bufio

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnbuffered(t *testing.T) {
	t.Run("without html escaping", func(t *testing.T) {
		t.Run("with bytes.Buffer", func(t *testing.T) {
			var w bytes.Buffer
			b := NewUnbuffered(&w, NoEscapeHTML)

			t.Run("should write single byte", func(t *testing.T) {
				w.Reset()

				b.WriteSingleByte('{')
				b.WriteSingleByte('}')

				assert.Equal(t, 2, w.Len())
				assert.Equal(t, `{}`, w.String())
				assert.Equal(t, int64(2), b.Size())
			})

			t.Run("should write unescaped bytes", func(t *testing.T) {
				w.Reset()

				const s = `"xyz"`
				b.WriteBinary([]byte(s))
				assert.Equal(t, 5, w.Len())
				assert.Equal(t, s, w.String())
				assert.Equal(t, int64(7), b.Size())
			})

			t.Run("should write escaped bytes", func(t *testing.T) {
				t.Run("should escape double quotes", func(t *testing.T) {
					w.Reset()

					const s = `"xyz"`
					b.WriteText([]byte(s))
					assert.Equal(t, 7, w.Len())
					assert.Equal(t, `\"xyz\"`, w.String())
					assert.Equal(t, int64(14), b.Size())
				})

				t.Run("should escape line feeds", func(t *testing.T) {
					w.Reset()

					const s = "xyz\nabc"
					b.WriteText([]byte(s))
					assert.Equal(t, len(s)+1, w.Len())
					assert.Equal(t, `xyz\nabc`, w.String())
					assert.Equal(t, int64(14+len(s)+1), b.Size())
				})

				t.Run("should escape carriage returns", func(t *testing.T) {
					w.Reset()

					const s = "xyz\rabc\rdef"
					b.WriteText([]byte(s))
					assert.Equal(t, len(s)+2, w.Len())
					assert.Equal(t, `xyz\rabc\rdef`, w.String())
				})

				t.Run("should escape tabs", func(t *testing.T) {
					w.Reset()

					const s = "xyz\tabc\tdef"
					b.WriteText([]byte(s))
					assert.Equal(t, len(s)+2, w.Len())
					assert.Equal(t, `xyz\tabc\tdef`, w.String())
				})
			})

			t.Run("should write escaped string", func(t *testing.T) {
				t.Run("should escape double quotes", func(t *testing.T) {
					w.Reset()

					const s = `"xyz"`
					b.WriteString(s)
					assert.Equal(t, 7, w.Len())
					assert.Equal(t, `\"xyz\"`, w.String())
				})
			})

			t.Run("should copy from reader unescaped", func(t *testing.T) {
				w.Reset()

				const s = `"xyz"`
				r := bytes.NewReader([]byte(s))
				b.WriteBinaryFrom(r)

				assert.Equal(t, len(s), w.Len())
				assert.Equal(t, s, w.String())
			})

			t.Run("should copy from reader with escape", func(t *testing.T) {
				w.Reset()

				const s = `"xyz"`
				r := bytes.NewReader([]byte(s))
				b.WriteTextFrom(r)

				assert.Equal(t, len(s)+2, w.Len())
				assert.Equal(t, `\"xyz\"`, w.String())
			})
		})
	})
}
