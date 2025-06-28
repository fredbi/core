package constrained

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConstrained(t *testing.T) {
	obj := rdr(`{"a":[1,2],"b":true,"c":{"x":[1,2,3]}}`)
	arr := rdr(`["a",[1,2],"b",true]`)
	boolv := rdr(`true`)
	str := rdr(`"test"`)
	num := rdr(`123`)
	null := rdr(`null`)
	strArr := rdr(`["a","b","c"]`)
	emptyArr := rdr(`[]`)
	emptyObj := rdr(`{}`)
	objArr := rdr(`[{"a":1},{"b": [1,2]},{}]`)

	t.Run("with object document", func(t *testing.T) {
		object := MakeObject()

		require.NoError(t, object.Decode(obj()))
		require.Error(t, object.Decode(arr()))
		require.Error(t, object.Decode(boolv()))
		require.Error(t, object.Decode(str()))
		require.Error(t, object.Decode(num()))
		require.Error(t, object.Decode(null()))
		require.Error(t, object.Decode(strArr()))
		require.Error(t, object.Decode(emptyArr()))
		require.NoError(t, object.Decode(emptyObj()))
		require.Error(t, object.Decode(objArr()))
	})

	t.Run("with array document", func(t *testing.T) {
		array := MakeArray()

		require.Error(t, array.Decode(obj()))
		require.NoError(t, array.Decode(arr()))
		require.Error(t, array.Decode(boolv()))
		require.Error(t, array.Decode(str()))
		require.Error(t, array.Decode(num()))
		require.Error(t, array.Decode(null()))
		require.NoError(t, array.Decode(strArr()))
		require.NoError(t, array.Decode(emptyArr()))
		require.Error(t, array.Decode(emptyObj()))
		require.NoError(t, array.Decode(objArr()))
	})

	t.Run("with string or array of strings document", func(t *testing.T) {
		strOrArray := MakeStringOrArrayOfStrings()

		require.Error(t, strOrArray.Decode(obj()))
		require.Error(t, strOrArray.Decode(arr()))
		require.Error(t, strOrArray.Decode(boolv()))
		require.NoError(t, strOrArray.Decode(str()))
		require.Error(t, strOrArray.Decode(num()))
		require.Error(t, strOrArray.Decode(null()))
		require.NoError(t, strOrArray.Decode(strArr()))
		require.NoError(t, strOrArray.Decode(emptyArr()))
		require.Error(t, strOrArray.Decode(emptyObj()))
		require.Error(t, strOrArray.Decode(objArr()))
	})

	t.Run("with bool or object document", func(t *testing.T) {
		boolOrObject := MakeBoolOrObject()

		require.NoError(t, boolOrObject.Decode(obj()))
		require.Error(t, boolOrObject.Decode(arr()))
		require.NoError(t, boolOrObject.Decode(boolv()))
		require.Error(t, boolOrObject.Decode(str()))
		require.Error(t, boolOrObject.Decode(num()))
		require.Error(t, boolOrObject.Decode(null()))
		require.Error(t, boolOrObject.Decode(strArr()))
		require.Error(t, boolOrObject.Decode(emptyArr()))
		require.NoError(t, boolOrObject.Decode(emptyObj()))
		require.Error(t, boolOrObject.Decode(objArr()))
	})

	t.Run("with object or array of objects document", func(t *testing.T) {
		ooa := MakeObjectOrArrayOfObjects()

		require.NoError(t, ooa.Decode(obj()))
		require.Error(t, ooa.Decode(arr()))
		require.Error(t, ooa.Decode(boolv()))
		require.Error(t, ooa.Decode(str()))
		require.Error(t, ooa.Decode(num()))
		require.Error(t, ooa.Decode(null()))
		require.Error(t, ooa.Decode(strArr()))
		require.NoError(t, ooa.Decode(emptyArr()))
		require.NoError(t, ooa.Decode(emptyObj()))
		require.NoError(t, ooa.Decode(objArr()))
	})
}

func rdr(s string) func() io.Reader {
	return func() io.Reader {
		return bytes.NewBufferString(s)
	}
}
