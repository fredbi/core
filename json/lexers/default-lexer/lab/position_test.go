package lab

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Line/column accounting lives ONLY in the verbatim lexer now. The semantic lexer
// L deliberately exposes no Line()/Column() (tracking them is costly on
// whitespace-heavy input and cannot be recovered lazily on a streaming buffer —
// see the note in lexer.go). These tests pin the verbatim position contract on
// [VL] and [token.VT]. Byte position for the semantic lexer is [L.Offset].

type pos struct{ line, col int }

// doc spans two lines; column/line are 1-based:
//
//	line 1: {"a": 12,
//	line 2:  "b": true}
//
// expected start positions of every token (separators included; VL never elides):
//
//	{    (1,1)
//	"a"  (1,2)   :  (1,5)   12  (1,7)   ,  (1,9)
//	"b"  (2,2)   :  (2,5)   true(2,7)   } (2,11)
const posDoc = "{\"a\": 12,\n \"b\": true}"

func posWant() []pos {
	return []pos{
		{1, 1},
		{1, 2}, {1, 5}, {1, 7}, {1, 9},
		{2, 2}, {2, 5}, {2, 7}, {2, 11},
	}
}

func TestLinePosition(t *testing.T) {
	t.Run("VL carries start line/column in the token", func(t *testing.T) {
		vl := NewVerbatimWithBytes([]byte(posDoc))

		var got []pos
		for {
			tok := vl.NextToken()
			if !vl.Ok() || tok.IsEOF() {
				break
			}
			got = append(got, pos{tok.Line(), tok.Col()})
		}
		require.NoError(t, vl.Err())
		assert.Equal(t, posWant(), got)
	})

	t.Run("VL methods agree with the token fields", func(t *testing.T) {
		vl := NewVerbatimWithBytes([]byte(posDoc))
		for {
			tok := vl.NextToken()
			if !vl.Ok() || tok.IsEOF() {
				break
			}
			assert.Equal(t, tok.Line(), vl.Line())
			assert.Equal(t, tok.Col(), vl.Column())
		}
		require.NoError(t, vl.Err())
	})

	t.Run("VL streaming with a tiny buffer reports the same positions", func(t *testing.T) {
		vl := NewVerbatim(strings.NewReader(posDoc), WithBufferSize(4))

		var got []pos
		for {
			tok := vl.NextToken()
			if !vl.Ok() || tok.IsEOF() {
				break
			}
			got = append(got, pos{tok.Line(), tok.Col()})
		}
		require.NoError(t, vl.Err())
		assert.Equal(t, posWant(), got)
	})
}

func TestLinePositionMultiline(t *testing.T) {
	// values spread across several lines, with a CRLF line ending mixed in
	doc := "[\n  1,\r\n  2,\n  3\n]"
	//	line1: [
	//	line2:   1,
	//	line3:   2,
	//	line4:   3
	//	line5: ]
	vl := NewVerbatimWithBytes([]byte(doc))

	var got []pos
	for {
		tok := vl.NextToken()
		if !vl.Ok() || tok.IsEOF() {
			break
		}
		got = append(got, pos{tok.Line(), tok.Col()})
	}
	require.NoError(t, vl.Err())
	assert.Equal(t, []pos{
		{1, 1},         // [
		{2, 3}, {2, 4}, // 1 ,
		{3, 3}, {3, 4}, // 2 ,
		{4, 3}, // 3
		{5, 1}, // ]
	}, got)
}
