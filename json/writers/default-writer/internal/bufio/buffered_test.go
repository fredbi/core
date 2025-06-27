package bufio

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkedBufferWriteTo(t *testing.T) {
	t.Run("without html escaping", func(t *testing.T) {
		t.Run("with bytes.Buffer", func(t *testing.T) {
			var w bytes.Buffer
			b := NewChunkedBuffer(NoEscapeHTML)

			t.Run("should write single byte", func(t *testing.T) {
				w.Reset()

				b.WriteSingleByte('{')
				b.WriteSingleByte('}')
				assert.Equal(t, int64(2), b.Size())

				n, err := b.WriteTo(&w)
				require.NoError(t, err)
				require.Equal(t, int64(2), n)
				assert.Equal(t, 2, w.Len())
				assert.Equal(t, `{}`, w.String())
			})

			t.Run("should write unescaped bytes", func(t *testing.T) {
				w.Reset()

				const s = `"xyz"`
				b.WriteBinary([]byte(s))
				assert.Equal(t, int64(5), b.Size())
				n, err := b.WriteTo(&w)
				require.NoError(t, err)
				require.Equal(t, int64(5), n)
				assert.Equal(t, 5, w.Len())
				assert.Equal(t, s, w.String())
			})

			t.Run("should write escaped bytes", func(t *testing.T) {
				t.Run("should escape double quotes", func(t *testing.T) {
					w.Reset()

					const s = `"xyz"`
					b.WriteText([]byte(s))
					assert.Equal(t, int64(7), b.Size())
					n, err := b.WriteTo(&w)
					require.NoError(t, err)
					require.Equal(t, int64(7), n)
					assert.Equal(t, 7, w.Len())
					assert.Equal(t, `\"xyz\"`, w.String())
				})
			})

			/*
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

				t.Run("Bytes() should be a stub", func(t *testing.T) {
					require.Nil(t, b.Bytes())
				})

				t.Run("WriteTo() should be a stub", func(t *testing.T) {
					var ww bytes.Buffer
					n, err := b.WriteTo(&ww)
					require.Nil(t, err)
					require.Empty(t, n)
				})
			*/
		})
	})
}

func TestChunkedBufferFragment(t *testing.T) {
	b := NewChunkedBuffer(NoEscapeHTML)
	piece := bytes.Repeat([]byte{'x'}, minSize/2)

	b.WriteBinary(piece)

	t.Run("should remain on the current buffer", func(t *testing.T) {
		assert.Empty(t, b.bufs)
		assert.Len(t, b.Buf, minSize/2)
	})

	b.WriteBinary(piece)

	t.Run("should still remain on the current buffer", func(t *testing.T) {
		assert.Empty(t, b.bufs)
		assert.Len(t, b.Buf, minSize)
	})

	b.WriteBinary(bytes.Join([][]byte{piece, piece}, []byte{';'}))

	t.Run("should switch to a new buffer, double the size of the first one", func(t *testing.T) {
		assert.Len(t, b.bufs, 1)
		assert.Len(t, b.Buf, minSize+1)
		assert.Equal(t, minSize*2, cap(b.Buf))
	})

	b.WriteBinary(bytes.Join([][]byte{piece, piece}, []byte{';'}))

	t.Run("should switch to a new buffer, double the size of the first one", func(t *testing.T) {
		assert.Len(t, b.bufs, 2)
		assert.Equal(t, minSize*2, cap(b.Buf))
	})

	t.Log(b.Size())
}
