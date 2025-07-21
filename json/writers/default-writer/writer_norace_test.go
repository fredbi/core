//go:build !race
// +build !race

package writer

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllocs(t *testing.T) {
	const epsilon = 1e-6
	stuff := prepareStuff()

	t.Run("with unbuffered", func(t *testing.T) {
		var tw bytes.Buffer

		t.Run(
			"all allocations should be amortized (excluding math/big values)",
			func(t *testing.T) {
				allocs := testing.AllocsPerRun(100, func() {
					tw.Reset()
					jw := BorrowUnbuffered(&tw)
					defer func() {
						RedeemUnbuffered(jw)
					}()
					writeStuffWithoutBig(t, jw, stuff)
					require.NoError(t, jw.Err())
				})
				assert.InDelta(t, 0, allocs, epsilon)
			},
		)

		t.Run(
			"most but not all allocations should be amortized (including math/big values)",
			func(t *testing.T) {
				allocs := testing.AllocsPerRun(100, func() {
					tw.Reset()
					jw := BorrowUnbuffered(&tw)
					defer func() {
						RedeemUnbuffered(jw)
					}()
					writeStuff(t, jw, stuff)
					require.NoError(t, jw.Err())
					require.NoError(t, jw.Err())
				})
				assert.InDelta(
					t,
					10,
					allocs,
					epsilon,
				) // this assertion is sensitive to the math/big package
			},
		)
	})

	t.Run("with pooled buffered", func(t *testing.T) {
		var tw bytes.Buffer

		t.Run(
			"all allocations should be amortized (excluding math/big values)",
			func(t *testing.T) {
				allocs := testing.AllocsPerRun(10000, func() {
					tw.Reset()
					jw := BorrowBuffered(&tw)
					defer func() {
						RedeemBuffered(jw)
					}()
					writeStuffWithoutBig(t, jw, stuff)
					require.NoError(t, jw.Err())
					require.NoError(t, jw.Flush())
				})
				assert.InDelta(t, 0, allocs, epsilon)
			},
		)

		t.Run(
			"most but not all allocations should be amortized (including math/big values)",
			func(t *testing.T) {
				allocs := testing.AllocsPerRun(100, func() {
					tw.Reset()
					jw := BorrowBuffered(&tw)
					defer func() {
						RedeemBuffered(jw)
					}()
					writeStuff(t, jw, stuff)
					require.NoError(t, jw.Err())
					require.NoError(t, jw.Flush())
				})
				assert.InDelta(
					t,
					10,
					allocs,
					epsilon,
				) // this assertion is sensitive to the math/big package
			},
		)
	})

	t.Run("with pooled indented", func(t *testing.T) {
		var tw bytes.Buffer

		t.Run(
			"all allocations should be amortized (excluding math/big values)",
			func(t *testing.T) {
				allocs := testing.AllocsPerRun(10000, func() {
					tw.Reset()
					jw := BorrowIndented(&tw)
					defer func() {
						RedeemIndented(jw)
					}()
					writeStuffWithoutBig(t, jw, stuff)
					require.NoError(t, jw.Err())
					require.NoError(t, jw.Flush())
				})
				assert.InDelta(t, 0, allocs, epsilon)
			},
		)

		t.Run(
			"most but not all allocations should be amortized (including math/big values)",
			func(t *testing.T) {
				allocs := testing.AllocsPerRun(100, func() {
					tw.Reset()
					jw := BorrowIndented(&tw)
					defer func() {
						RedeemIndented(jw)
					}()
					writeStuff(t, jw, stuff)
					require.NoError(t, jw.Err())
					require.NoError(t, jw.Flush())
				})
				assert.InDelta(
					t,
					10,
					allocs,
					epsilon,
				) // this assertion is sensitive to the math/big package
			},
		)
	})
}
