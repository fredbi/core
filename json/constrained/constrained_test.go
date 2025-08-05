package constrained

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstrained(t *testing.T) {
	const (
		// test cases for constrained documents
		cObj      = `{"a":[1,2],"b":true,"c":{"x":[1,2,3]}}`
		cArr      = `["a",[1,2],"b",true]`
		cBool     = `true`
		cStr      = `"test"`
		cNum      = `123`
		cNull     = `null`
		cStrArr   = `["a","b","c"]`
		cEmptyArr = `[]`
		cEmptyObj = `{}`
		cObjArr   = `[{"a":1},{"b": [1,2]},{}]`
	)

	// build readers
	obj := rdr(cObj)
	arr := rdr(cArr)
	boolv := rdr(cBool)
	str := rdr(cStr)
	num := rdr(cNum)
	null := rdr(cNull)
	strArr := rdr(cStrArr)
	emptyArr := rdr(cEmptyArr)
	emptyObj := rdr(cEmptyObj)
	objArr := rdr(cObjArr)

	t.Run("with object document", func(t *testing.T) {
		object := MakeObject()

		require.NoError(t, object.Decode(obj()))
		assertEncode(cObj, object)(t)

		require.Error(t, object.Decode(arr()))
		require.Error(t, object.Decode(boolv()))
		require.Error(t, object.Decode(str()))
		require.Error(t, object.Decode(num()))
		require.Error(t, object.Decode(null()))
		require.Error(t, object.Decode(strArr()))
		require.Error(t, object.Decode(emptyArr()))
		require.NoError(t, object.Decode(emptyObj()))
		assertEncode(cEmptyObj, object)(t)

		require.Error(t, object.Decode(objArr()))
	})

	t.Run("with array document", func(t *testing.T) {
		array := MakeArray()

		require.Error(t, array.Decode(obj()))

		require.NoError(t, array.Decode(arr()))
		assertEncode(cArr, array)(t)

		require.Error(t, array.Decode(boolv()))
		require.Error(t, array.Decode(str()))
		require.Error(t, array.Decode(num()))
		require.Error(t, array.Decode(null()))

		require.NoError(t, array.Decode(strArr()))
		assertEncode(cStrArr, array)(t)

		require.NoError(t, array.Decode(emptyArr()))
		assertEncode(cEmptyArr, array)(t)

		require.Error(t, array.Decode(emptyObj()))

		require.NoError(t, array.Decode(objArr()))
		assertEncode(cObjArr, array)(t)
	})

	t.Run("with string or array of strings document", func(t *testing.T) {
		strOrArray := MakeStringOrArrayOfStrings()

		require.Error(t, strOrArray.Decode(obj()))
		require.Error(t, strOrArray.Decode(arr()))
		require.Error(t, strOrArray.Decode(boolv()))

		require.NoError(t, strOrArray.Decode(str()))
		assertEncode(cStr, strOrArray)(t)

		require.Error(t, strOrArray.Decode(num()))
		require.Error(t, strOrArray.Decode(null()))

		require.NoError(t, strOrArray.Decode(strArr()))
		assertEncode(cStrArr, strOrArray)(t)

		require.NoError(t, strOrArray.Decode(emptyArr()))
		assertEncode(cEmptyArr, strOrArray)(t)

		require.Error(t, strOrArray.Decode(emptyObj()))
		require.Error(t, strOrArray.Decode(objArr()))
	})

	t.Run("with bool or object document", func(t *testing.T) {
		boolOrObject := MakeBoolOrObject()

		require.NoError(t, boolOrObject.Decode(obj()))
		assertEncode(cObj, boolOrObject)(t)

		require.Error(t, boolOrObject.Decode(arr()))

		require.NoError(t, boolOrObject.Decode(boolv()))
		assertEncode(cBool, boolOrObject)(t)

		require.Error(t, boolOrObject.Decode(str()))
		require.Error(t, boolOrObject.Decode(num()))
		require.Error(t, boolOrObject.Decode(null()))
		require.Error(t, boolOrObject.Decode(strArr()))
		require.Error(t, boolOrObject.Decode(emptyArr()))

		require.NoError(t, boolOrObject.Decode(emptyObj()))
		assertEncode(cEmptyObj, boolOrObject)(t)

		require.Error(t, boolOrObject.Decode(objArr()))
	})

	t.Run("with object or array of objects document", func(t *testing.T) {
		ooa := MakeObjectOrArrayOfObjects()

		require.NoError(t, ooa.Decode(obj()))
		assertEncode(cObj, ooa)(t)

		require.Error(t, ooa.Decode(arr()))
		require.Error(t, ooa.Decode(boolv()))
		require.Error(t, ooa.Decode(str()))
		require.Error(t, ooa.Decode(num()))
		require.Error(t, ooa.Decode(null()))
		require.Error(t, ooa.Decode(strArr()))

		require.NoError(t, ooa.Decode(emptyArr()))
		assertEncode(cEmptyArr, ooa)(t)

		require.NoError(t, ooa.Decode(emptyObj()))
		assertEncode(cEmptyObj, ooa)(t)

		require.NoError(t, ooa.Decode(objArr()))
		assertEncode(cObjArr, ooa)(t)
	})
}

func rdr(s string) func() io.Reader {
	return func() io.Reader {
		return bytes.NewBufferString(s)
	}
}

func assertEncode(expected string, doc interface{ Encode(io.Writer) error }) func(*testing.T) {
	return func(t *testing.T) {
		t.Run("should encode as the original", func(t *testing.T) {
			var w bytes.Buffer
			require.NoError(t, doc.Encode(&w))
			assert.JSONEq(t, expected, w.String())
		})
	}
}
