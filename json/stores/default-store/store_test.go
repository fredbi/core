package store

import (
	"bytes"
	"compress/flate"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/stores/values"
	"github.com/fredbi/core/json/types"
	writer "github.com/fredbi/core/json/writers/default-writer"
)

func TestStores(t *testing.T) {
	t.Run("with Store", testGetPutValue(New()))
	t.Run("with ConcurrentStore", testGetPutValue(NewConcurrent()))
}

func TestEdgeCases(t *testing.T) {
	t.Run("with Store", testEdgeCases(New()))
	t.Run("with ConcurrentStore", testEdgeCases(NewConcurrent()))
}

func testGetPutValue(s stores.Store) func(*testing.T) {
	return func(t *testing.T) {
		t.Run("with null", func(t *testing.T) {
			l := s.Len()

			t.Run("with null value", checkNull(s))
			t.Run("with null token", checkNullToken(s))
			t.Run("null should not add to arena", func(t *testing.T) {
				assert.Equal(t, s.Len(), l)
			})
		})

		t.Run("with bool", func(t *testing.T) {
			l := s.Len()

			t.Run("with true", checkBool(s, true))
			t.Run("with true token", checkBoolToken(s, true))

			t.Run("with false", checkBool(s, false))
			t.Run("with false token", checkBoolToken(s, false))

			t.Run("bool should not add to arena", func(t *testing.T) {
				assert.Equal(t, s.Len(), l)
			})
		})

		t.Run("with inlined numbers", func(t *testing.T) {
			t.Run("with integer", func(t *testing.T) {
				const n = "123"
				l := s.Len()
				t.Run("should retrieve original value", checkNumber(s, n))
				t.Run("inlined should not add to arena", func(t *testing.T) {
					assert.Equal(t, s.Len(), l)
				})
				t.Run("should retrieve original value (token)", checkNumberToken(s, n))
			})

			t.Run("with decimal", func(t *testing.T) {
				const n = "-123.456"
				l := s.Len()
				t.Run("should retrieve original value", checkNumber(s, n))
				t.Run("inlined should not add to arena", func(t *testing.T) {
					assert.Equal(t, s.Len(), l)
				})
				t.Run("should retrieve original value (token)", checkNumberToken(s, n))
			})

			t.Run("with scientific notation", func(t *testing.T) {
				const n = "-123.456E-4"
				l := s.Len()
				t.Run("should retrieve original value", checkNumber(s, n))
				t.Run("inlined should not add to arena", func(t *testing.T) {
					assert.Equal(t, s.Len(), l)
				})
				t.Run("should retrieve original value (token)", checkNumberToken(s, n))
			})
			t.Run("with zero", func(t *testing.T) {
				const n = "0"
				l := s.Len()
				t.Run("should retrieve original value", checkNumber(s, n))
				t.Run("inlined should not add to arena", func(t *testing.T) {
					assert.Equal(t, s.Len(), l)
				})
				t.Run("should retrieve original value (token)", checkNumberToken(s, n))
			})
		})

		t.Run("with in-arena number (len=15)", func(t *testing.T) {
			const n = "123456789012345"
			l := s.Len()
			t.Run("should retrieve original value", checkNumber(s, n))
			t.Run("should add to arena", func(t *testing.T) {
				assert.Equal(t, s.Len(), l+len(n)/2+len(n)%2)
			})
			t.Run("should retrieve original value (token)", checkNumberToken(s, n))
		})

		t.Run("with inlined string (len=1)", func(t *testing.T) {
			const str = "a"
			l := s.Len()
			t.Run("should retrieve original value", checkString(s, str))
			t.Run("inlined should not add to arena", func(t *testing.T) {
				assert.Equal(t, s.Len(), l)
			})
			t.Run("should retrieve original value (token)", checkStringToken(s, str))
		})

		t.Run("with inlined string (len=7)", func(t *testing.T) {
			const str = "abcdefg"
			l := s.Len()
			t.Run("should retrieve original value", checkString(s, str))
			t.Run("inlined should not add to arena", func(t *testing.T) {
				assert.Equal(t, s.Len(), l)
			})
			t.Run("should retrieve original value (token)", checkStringToken(s, str))
		})

		t.Run("with empty string", func(t *testing.T) {
			const str = ""
			l := s.Len()
			t.Run("should retrieve original value", checkString(s, str))
			t.Run("inlined should not add to arena", func(t *testing.T) {
				assert.Equal(t, s.Len(), l)
			})
			t.Run("should retrieve original value (token)", checkStringToken(s, str))
		})

		t.Run("with inlined ASCII-only string (len=8)", func(t *testing.T) {
			const str = "abcdefgh"
			require.True(t, isOnlyASCII([]byte(str)))
			l := s.Len()
			t.Run("should retrieve original value", checkString(s, str))
			t.Run("inlined should not add to arena", func(t *testing.T) {
				assert.Equal(t, s.Len(), l)
			})
			t.Run("should retrieve original value (token)", checkStringToken(s, str))
		})

		t.Run("with in-arena string (len=9)", func(t *testing.T) {
			const str = "abcdefghi"
			l := s.Len()
			t.Run("should retrieve original value", checkString(s, str))
			t.Run("should add to arena", func(t *testing.T) {
				assert.Equal(t, s.Len(), l+len(str))
			})
			t.Run("should retrieve original value (token)", checkStringToken(s, str))
		})

		t.Run("with compressed string (len=500)", func(t *testing.T) {
			str := strings.Repeat("abcdefghij", 50)
			l := s.Len()
			t.Run("should retrieve original value", checkString(s, str))
			t.Run("should add less than original string to arena", func(t *testing.T) {
				assert.Greater(t, s.Len(), l)
				assert.Less(t, s.Len(), l+len(str))
			})
			t.Run("handle header should be large compressed string", func(t *testing.T) {
				input := values.MakeStringValue(str)
				h := s.PutValue(input)
				assert.Equal(t, stores.Handle(headerCompressedString), h&headerMask)
			})
			t.Run("should retrieve original value (token)", checkStringToken(s, str))
		})

		t.Run("with compressed string (len=129)", func(t *testing.T) {
			str := strings.Repeat("a", 129)
			l := s.Len()
			t.Run("should retrieve original value", checkString(s, str))
			t.Run("should add less than original string to arena", func(t *testing.T) {
				assert.Greater(t, s.Len(), l)
				assert.Less(t, s.Len(), l+len(str))
			})
			t.Run("handle header should be large compressed string", func(t *testing.T) {
				input := values.MakeStringValue(str)
				h := s.PutValue(input)
				assert.Equal(t, stores.Handle(headerCompressedString), h&headerMask)
			})
			t.Run("should retrieve original value (token)", checkStringToken(s, str))
		})

		t.Run("with Store and compression options", func(t *testing.T) {
			s := New(
				WithCompressionLevel(flate.BestCompression),
				WithCompressionThreshold(10))

			t.Run("with compressed string (len=150)", func(t *testing.T) {
				str := strings.Repeat("xyz", 50)
				l := s.Len()
				t.Run("should retrieve original value", checkString(s, str))
				t.Run("should add less than original string to arena", func(t *testing.T) {
					assert.Greater(t, s.Len(), l)
					assert.Less(t, s.Len(), l+len(str))
				})

				t.Run("handle header should be large compressed string", func(t *testing.T) {
					input := values.MakeStringValue(str)
					h := s.PutValue(input)
					assert.Equal(t, stores.Handle(headerCompressedString), h&headerMask)
				})

				t.Run("should retrieve original value (token)", checkStringToken(s, str))
			})

			/*
				-- can't enable this: flate's minimum size is 9 bytes
				t.Run("with inlined compressed strings", func(t *testing.T) {
					str := strings.Repeat("a", 11)
					l := len(s.arena)
					t.Run("should retrieve original value", checkString(s, str))
					t.Run("should add less than original string to arena", func(t *testing.T) {
						assert.Greater(t, len(s.arena), l)
						assert.Less(t, len(s.arena), l+len(str))
					})
					t.Run("handle header should be inlined compressed string", func(t *testing.T) {
						input := values.MakeStringValue(str)
						h := s.PutValue(input)
						assert.Equal(t, stores.Handle(headerInlinedCompressedString), h&headerMask)
					})
				})
			*/
		})
	}
}

func checkNull(s stores.Store) func(*testing.T) {
	return func(t *testing.T) {
		input := values.NullValue
		h := s.PutValue(input)
		outcome := s.Get(h)
		assert.Equal(t, input, outcome)
	}
}

func checkNullToken(s stores.Store) func(*testing.T) {
	return func(t *testing.T) {
		input := token.NullToken
		h := s.PutToken(input)
		outcome := s.Get(h)
		assert.Equal(t, values.NullValue, outcome)
	}
}

func checkBool(s stores.Store, b bool) func(*testing.T) {
	return func(t *testing.T) {
		input := values.MakeBoolValue(b)
		h := s.PutValue(input)
		outcome := s.Get(h)
		assert.Equal(t, input, outcome)
	}
}

func checkBoolToken(s stores.Store, b bool) func(*testing.T) {
	return func(t *testing.T) {
		input := token.MakeBoolean(b)
		h := s.PutToken(input)
		outcome := s.Get(h)
		assert.Equal(t, values.MakeBoolValue(b), outcome)
	}
}

func checkNumber(s stores.Store, n string) func(*testing.T) {
	return func(t *testing.T) {
		input := values.MakeNumberValue(types.Number{Value: []byte(n)})
		h := s.PutValue(input)
		require.NotEmpty(t, h)
		outcome := s.Get(h)

		assert.Equal(t, input, outcome)
	}
}

func checkNumberToken(s stores.Store, n string) func(*testing.T) {
	return func(t *testing.T) {
		input := token.MakeWithValue(token.Number, []byte(n))
		h := s.PutToken(input)
		require.NotEmpty(t, h)
		outcome := s.Get(h)

		expected := values.MakeNumberValue(types.Number{Value: []byte(n)})
		assert.Equal(t, expected, outcome)
	}
}

func checkString(s stores.Store, str string) func(*testing.T) {
	return func(t *testing.T) {
		input := values.MakeStringValue(str)
		h := s.PutValue(input)
		outcome := s.Get(h)
		expected := values.MakeStringValue(str)
		assert.Equal(t, expected, outcome)
	}
}

func checkStringToken(s stores.Store, str string) func(*testing.T) {
	return func(t *testing.T) {
		input := token.MakeWithValue(token.String, []byte(str))
		h := s.PutToken(input)
		outcome := s.Get(h)
		expected := values.MakeStringValue(str)
		assert.Equal(t, expected, outcome)
	}
}

func testEdgeCases(s stores.Store) func(*testing.T) {
	return func(t *testing.T) {
		t.Run("with Put", func(t *testing.T) {
			t.Run("PutToken: providing an invalid token should panic", func(t *testing.T) {
				assert.Panics(t, func() {
					s.PutToken(
						token.MakeDelimiter(token.ClosingBracket),
					) // invalid token for a value
				})
			})
		})
		t.Run("with Get", func(t *testing.T) {
			t.Run("providing an invalid handle should panic", func(t *testing.T) {
				assert.Panics(t, func() {
					h := stores.Handle(0xf)
					s.Get(h)
				})
			})

			t.Run(
				"providing a number handle that refer to an uncharted arena part should panic",
				testOutOfRangeHandle(s, headerNumber),
			)
			t.Run(
				"providing a string handle that refer to an uncharted arena part should panic",
				testOutOfRangeHandle(s, headerString),
			)
			t.Run(
				"providing a compressed string handle that refer to an uncharted arena part should panic",
				testOutOfRangeHandle(s, headerCompressedString),
			)
		})
	}
}

// TestWriteToCorruptHandle covers the writer-driven path (E5): unlike Get, WriteTo has the writer as an
// error sink, so a corrupted handle surfaces as an error via the writer instead of panicking.
func TestWriteToCorruptHandle(t *testing.T) {
	s := New()
	for name, headerPart := range map[string]uint8{
		"number":           headerNumber,
		"string":           headerString,
		"compressedString": headerCompressedString,
	} {
		t.Run("out of range "+name+" handle errors (no panic)", func(t *testing.T) {
			const (
				dummySize        = uint64(10)
				outOfRangeOffset = uint64(100)
			)
			h := stores.Handle(
				uint64(headerPart) |
					(dummySize << headerBits) |
					(outOfRangeOffset << (headerBits + lengthBits)),
			)

			var buf bytes.Buffer
			w := writer.NewUnbuffered(&buf)

			require.NotPanics(t, func() { s.WriteTo(w, h) })
			require.Error(t, w.Err())
			assert.ErrorContains(t, w.Err(), "out of range offset")
		})
	}
}

func testOutOfRangeHandle(s stores.Store, headerPart uint8) func(*testing.T) {
	return func(t *testing.T) {
		const (
			dummySize        = uint64(10)
			outOfRangeOffset = uint64(100)
		)

		assert.Panics(t, func() {
			h := stores.Handle(
				uint64(
					headerPart,
				) | (dummySize << headerBits) | (outOfRangeOffset << (headerBits + lengthBits)),
			)
			_ = s.Get(h)
		})
	}
}
