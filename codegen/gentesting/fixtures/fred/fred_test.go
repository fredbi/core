package tmp

import (
	"encoding"
	"fmt"
	"plugin"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerated(t *testing.T) {
	lib := "../tmp/generated.so"
	pkg, err := plugin.Open(lib)
	require.NoError(t, err)

	t.Run(`package "generated" should export type "Model"`, func(t *testing.T) {
		modelSymbol, err := pkg.Lookup("Model")
		require.NoError(t, err)

		p := reflect.ValueOf(modelSymbol)
		require.Equalf(t, reflect.Pointer, p.Kind(), "%v: %T", p.Kind(), modelSymbol)
		value := reflect.Indirect(p)

		t.Run(`type "Model" should be a struct`, func(t *testing.T) {
			require.Equalf(t, reflect.Struct, value.Kind(), "%v: %T", value.Kind(), modelSymbol)

			model := value.Interface()
			modelPtr := p.Interface()

			t.Run(`type "Model" should contain a field "B" of type string`, func(t *testing.T) {
				fieldValue := value.FieldByName("B")
				require.True(t, fieldValue.IsValid())

				require.Equal(t, reflect.String, fieldValue.Kind())
			})

			t.Run(`type "Model" (value receiver) should implement "encoding.BinaryMarshaler"`, func(t *testing.T) {
				_, ok := model.(encoding.BinaryMarshaler)
				require.True(t, ok)
			})

			t.Run(`type "Model" (pointer receiver) should implement "encoding.BinaryMarshaler"`, func(t *testing.T) {
				withUnmarshalBinary, ok := modelPtr.(encoding.BinaryUnmarshaler)
				require.True(t, ok)
				testJSON := []byte(`{"a": [1,2], "b": "fred"}`)

				t.Run(`UnmarshalBinary should not error on sample input`, func(t *testing.T) {
					require.NoError(t, withUnmarshalBinary.UnmarshalBinary(testJSON))

					t.Run(`MarshalBinary should not error`, func(t *testing.T) {
						withMarshalBinary, ok := withUnmarshalBinary.(encoding.BinaryMarshaler)
						require.True(t, ok)

						data, err := withMarshalBinary.MarshalBinary()
						require.NoError(t, err)

						t.Run(`MarshalBinary should yield an equivalent representation of the original input`, func(t *testing.T) {
							require.JSONEq(t, string(testJSON), string(data))
						})
					})
				})
			})
		})
	})

	t.Run(`package "generated" should export type "IntegerCollection"`, func(t *testing.T) {
		collectionSymbol, err := pkg.Lookup("IntegerCollection")
		require.NoError(t, err)

		p := reflect.ValueOf(collectionSymbol)
		require.Equalf(t, reflect.Pointer, p.Kind(), "%v: %T", p.Kind(), collectionSymbol)
		value := reflect.Indirect(p)

		collectionType := value.Type()
		pkgPath := collectionType.PkgPath()
		assert.Contains(t, pkgPath, "core/codegen/gentesting/fixtures/generated")

		t.Run(`type "IntegerCollection" should have method String`, func(t *testing.T) {
			method, ok := collectionType.MethodByName("String")
			require.True(t, ok)

			require.Equalf(t, reflect.Func, method.Type.Kind(), "%v", method.Type.Kind())
			require.Equal(t, 1, method.Type.NumIn()) // add the receiver as first parameter
			require.Equal(t, 1, method.Type.NumOut())
		})

		t.Run(`type "IntegerCollection" should be a slice of int64`, func(t *testing.T) {
			require.Equalf(t, reflect.Slice, value.Kind(), "%v: %T", value.Kind(), collectionSymbol)

			elementType := value.Type().Elem()
			require.Equalf(t, reflect.Int64, elementType.Kind(), "%v: %T", elementType.Kind(), collectionSymbol)

			collection := value.Interface()

			_, ok := collection.([]int64)
			require.Falsef(t, ok, "collection: %T", collection)

			t.Run(`type "IntegerCollection" should implement fmt.Stringer`, func(t *testing.T) {
				_, isStringer := collection.(fmt.Stringer)
				require.True(t, isStringer)
			})
		})
	})

	t.Run(`package "generated" should export const "AConstant"`, func(t *testing.T) {
		aConstantSymbol, err := pkg.Lookup("AConstant") // actually a variable when reexported
		require.NoError(t, err)

		p := reflect.ValueOf(aConstantSymbol)
		require.Equalf(t, reflect.Pointer, p.Kind(), "%v: %T", p.Kind(), aConstantSymbol)
		value := reflect.Indirect(p)
		aConstant := value.Interface()

		_, ok := aConstant.(string)
		require.True(t, ok)
	})

	/*
		instance, err := pkg.Lookup("InstanceOfModel")
		if assert.NoError(t, err) {
			f := reflect.ValueOf(instance)
			require.Equalf(t, reflect.Pointer, f.Kind(), "%v: %T", f.Kind(), instance)

			vv := reflect.Indirect(f)

			assert.Equalf(t, reflect.Struct, vv.Kind(), "%v: %T", vv.Kind(), instance)
		}

		/*modelConstructor */
	/*
		_, err = pkg.Lookup("NewModel")
		require.NoError(t, err)


		hasMarshalBinary, ok := model.(encoding.BinaryMarshaler)
		require.True(t, ok)

	*/
}
