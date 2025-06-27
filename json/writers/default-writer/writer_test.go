package writer

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/types"
)

const expected = `{
	"stores_values":[
		"\"quoted_string\"",
		false,
		123,
		null
	],
	"native_number":[15,0.9230769230769231,12,12.12,12.23],
	"native_strings":[
		"\"quoted\"",
		"escaped\nstring",
		"escaped\tstring"
	],
	"readers":[
		"from\treader",
		678
	],
	"native_bool":true,
	"native_null":null,
	"json_types":[null,true,"quote",12345],
	"raw":[45,1],
	"tokens":[
		true,
		999,
		"\t\r\n\"\\r",
	  {
			"key_token":[
				null,
				{"inner":{}}
			]
		}
	]
}
`

func TestWriter(t *testing.T) {
	t.Run("with unbuffered", func(t *testing.T) {
		t.Run("should write JSON straight-through to io.Writer", func(t *testing.T) {
			var tw bytes.Buffer

			jw := New(&tw)

			writeStuff(jw)(t)
			assert.JSONEq(t, expected, tw.String())
		})
	})
}

func TestWriterFromPool(t *testing.T) {
	t.Run("with unbuffered", func(t *testing.T) {
		var tw bytes.Buffer

		jw := BorrowWriter(&tw)
		defer func() {
			RedeemWriter(jw)
		}()
		writeStuff(jw)(t)
		assert.JSONEq(t, expected, tw.String())
	})
}

func writeStuff(jw *W) func(*testing.T) {
	return func(t *testing.T) {
		jw.StartObject()
		// stores
		jw.Key(stores.MakeInternedKey("stores_values"))
		jw.StartArray()
		jw.Value(stores.MakeStringValue(`"quoted_string"`))
		jw.Comma()
		jw.Value(stores.MakeBoolValue(false))
		jw.Comma()
		jw.Value(stores.MakeNumberValue(types.Number{Value: []byte("123")}))
		jw.Comma()
		jw.Value(stores.NullValue)
		jw.EndArray()
		jw.Comma()
		// native numbers
		jw.String("native_number")
		jw.Colon()
		jw.StartArray()
		jw.Number(15)
		jw.Comma()
		jw.Number(big.NewRat(12, 13))
		jw.Comma()
		jw.Number(uint16(12))
		jw.Comma()
		jw.Number(float32(12.12))
		jw.Comma()
		jw.Number(big.NewFloat(12.23))
		jw.EndArray()
		jw.Comma()
		// native strings
		jw.StringBytes([]byte("native_strings"))
		jw.Colon()
		jw.StartArray()
		jw.String(`"quoted"`)
		jw.Comma()
		jw.StringBytes([]byte("escaped\nstring"))
		jw.Comma()
		jw.StringRunes([]rune("escaped\tstring"))
		jw.EndArray()
		jw.Comma()
		jw.Key(stores.MakeInternedKey("readers"))
		// readers
		jw.StartArray()
		rs := bytes.NewReader([]byte("from\treader"))
		jw.StringCopy(rs)
		require.NoError(t, jw.Err())
		jw.Comma()
		rn := bytes.NewReader([]byte("678"))
		jw.NumberCopy(rn)
		require.NoError(t, jw.Err())
		jw.EndArray()
		jw.Comma()
		// native
		jw.Key(stores.MakeInternedKey("native_bool"))
		jw.Bool(true)
		jw.Comma()
		jw.Key(stores.MakeInternedKey("native_null"))
		jw.Null()
		jw.Comma()
		jw.Key(stores.MakeInternedKey("json_types"))
		// json types
		jw.StartArray()
		jw.JSONNull(types.Null)
		jw.Comma()
		jw.JSONBoolean(types.Boolean{}.With(true))
		jw.Comma()
		jw.JSONString(types.String{Value: []byte("quote")})
		jw.Comma()
		jw.JSONNumber(types.Number{Value: []byte("12345")})
		jw.EndArray()
		jw.Comma()
		jw.Key(stores.MakeInternedKey("raw"))
		jw.Raw([]byte("[45,1]"))
		jw.Comma()
		jw.Key(stores.MakeInternedKey("tokens"))
		jw.StartArray()
		jw.Token(token.MakeBoolean(true))
		jw.Token(token.MakeDelimiter(token.Comma))
		jw.Token(token.MakeWithValue(token.Number, []byte("999")))
		jw.Token(token.MakeDelimiter(token.Comma))
		jw.Token(token.MakeWithValue(token.String, []byte("\t\r\n\"\\r")))
		jw.Token(token.MakeDelimiter(token.Comma))
		jw.Token(token.MakeDelimiter(token.OpeningBracket))
		jw.Token(token.MakeWithValue(token.Key, []byte("key_token")))
		jw.Token(token.MakeDelimiter(token.Colon))
		jw.Token(token.MakeDelimiter(token.OpeningSquareBracket))
		jw.Token(token.NullToken)
		jw.Token(token.MakeDelimiter(token.Comma))
		jw.Token(token.MakeDelimiter(token.OpeningBracket))
		jw.Token(token.MakeWithValue(token.Key, []byte("inner")))
		jw.Token(token.MakeDelimiter(token.Colon))
		jw.Token(token.MakeDelimiter(token.OpeningBracket))
		jw.Token(token.MakeDelimiter(token.ClosingBracket))
		jw.Token(token.MakeDelimiter(token.ClosingBracket))
		jw.Token(token.MakeDelimiter(token.ClosingSquareBracket))
		jw.Token(token.MakeDelimiter(token.ClosingBracket))
		jw.EndArray()
		jw.EndObject()
		require.NoError(t, jw.Err())
	}
}
