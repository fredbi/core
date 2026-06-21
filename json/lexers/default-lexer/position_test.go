package lexer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pos struct{ line, col int }

// doc spans two lines; column/line are 1-based:
//
//	line 1: {"a": 12,
//	line 2:  "b": true}
//
// expected start positions of every token (separators included):
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
	t.Run("L exposes start line/column via methods (separators kept)", func(t *testing.T) {
		lex := NewWithBytes([]byte(posDoc), WithElideSeparator(false))

		var got []pos
		for {
			tok := lex.NextToken()
			if !lex.Ok() || tok.IsEOF() {
				break
			}
			got = append(got, pos{lex.Line(), lex.Column()})
		}
		require.NoError(t, lex.Err())
		assert.Equal(t, posWant(), got)
	})

	t.Run("L streaming with a tiny buffer reports the same positions", func(t *testing.T) {
		lex := New(strings.NewReader(posDoc), WithElideSeparator(false), WithBufferSize(4))

		var got []pos
		for {
			tok := lex.NextToken()
			if !lex.Ok() || tok.IsEOF() {
				break
			}
			got = append(got, pos{lex.Line(), lex.Column()})
		}
		require.NoError(t, lex.Err())
		assert.Equal(t, posWant(), got)
	})

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
		// VL never elides, so it yields the same token set and positions
		assert.Equal(t, posWant(), got)
	})

	t.Run("VL methods (promoted from L) agree with token fields", func(t *testing.T) {
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

	t.Run("default L (elision on) keeps correct positions for surfaced tokens", func(t *testing.T) {
		// with elision, only { "a" 12 "b" true } are surfaced
		lex := NewWithBytes([]byte(posDoc))

		var got []pos
		for {
			tok := lex.NextToken()
			if !lex.Ok() || tok.IsEOF() {
				break
			}
			got = append(got, pos{lex.Line(), lex.Column()})
		}
		require.NoError(t, lex.Err())
		assert.Equal(t, []pos{
			{1, 1},         // {
			{1, 2}, {1, 7}, // "a" 12
			{2, 2}, {2, 7}, // "b" true
			{2, 11}, // }
		}, got)
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
	lex := NewWithBytes([]byte(doc), WithElideSeparator(false))

	var got []pos
	for {
		tok := lex.NextToken()
		if !lex.Ok() || tok.IsEOF() {
			break
		}
		got = append(got, pos{lex.Line(), lex.Column()})
	}
	require.NoError(t, lex.Err())
	assert.Equal(t, []pos{
		{1, 1},         // [
		{2, 3}, {2, 4}, // 1 ,
		{3, 3}, {3, 4}, // 2 ,
		{4, 3}, // 3
		{5, 1}, // ]
	}, got)
}
