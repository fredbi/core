package json

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type pointerTestCase struct {
	PointerString string
	ExpectedJSON  string
}

func TestPointer(t *testing.T) {
	t.Run("with RFC example", func(t *testing.T) {
		// runs the example provided at https://www.rfc-editor.org/rfc/rfc6901 as a test
		const jazon = `
 {
      "foo": ["bar", "baz"],
      "": 0,
      "a/b": 1,
      "c%d": 2,
      "e^f": 3,
      "g|h": 4,
      "i\\j": 5,
      "k\"l": 6,
      " ": 7,
      "m~n": 8
   }`
		doc := Make()

		t.Run("should build RFC example document", func(t *testing.T) {
			r := bytes.NewBufferString(jazon)

			require.NoError(t, doc.Decode(r))
		})

		testCases := []pointerTestCase{
			{
				PointerString: "",
				ExpectedJSON:  jazon,
			},
			{
				PointerString: "/foo",
				ExpectedJSON:  `["bar", "baz"]`,
			},
			{
				PointerString: "/foo/0",
				ExpectedJSON:  `"bar"`,
			},
			{
				PointerString: "/",
				ExpectedJSON:  `0`,
			},
			{
				PointerString: "/a~1b",
				ExpectedJSON:  `1`,
			},
			{
				PointerString: "/c%d",
				ExpectedJSON:  `2`,
			},
			{
				PointerString: "/e^f",
				ExpectedJSON:  `3`,
			},
			{
				PointerString: "/g|h",
				ExpectedJSON:  `4`,
			},
			{
				PointerString: "/i\\j",
				ExpectedJSON:  `5`,
			},
			{
				PointerString: "/k\"l",
				ExpectedJSON:  `6`,
			},
			{
				PointerString: "/ ",
				ExpectedJSON:  `7`,
			},
			{
				PointerString: "/m~0n",
				ExpectedJSON:  `8`,
			},
		}

		for _, tc := range testCases {
			t.Run(
				fmt.Sprintf("should parse JSON pointer %q", tc.PointerString),
				func(t *testing.T) {
					p, erp := MakePointer(tc.PointerString)
					require.NoError(t, erp)

					t.Run("pointer should resolve in document", func(t *testing.T) {
						result, err := doc.GetPointer(p)
						require.NoError(t, err)

						t.Run("resolved document should match expectation", func(t *testing.T) {
							b, err := result.MarshalJSON()
							require.NoError(t, err)

							require.JSONEq(t, tc.ExpectedJSON, string(b))
						})
					})
				},
			)
		}
	})
}
