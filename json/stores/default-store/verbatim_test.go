package store

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/stores"
)

func TestVerbatimStore(t *testing.T) {
	t.Run("VerbatimStore should work with regular values", testGetPutValue(NewVerbatim()))
	t.Run("verbatim values with VerbatimStore ", testGetPutVerbatimValue(NewVerbatim()))
	t.Run("verbatim tokens with VerbatimStore ", testGetPutVerbatimToken(NewVerbatim()))
}

func testGetPutVerbatimValue(s stores.VerbatimStore) func(*testing.T) {
	return func(t *testing.T) {
		t.Run("with inlined Blank", checkBlanks(s, "  \t\n\r\n\n"))
		t.Run("with max inlined Blank", checkBlanks(s, strings.Repeat("\t", maxInlineBlanks)))
		t.Run(
			"with compressed Blank",
			checkBlanks(
				s,
				strings.Repeat("\t", maxInlineBlanks)+strings.Repeat("\n", maxInlineBlanks),
			),
		)
	}
}

func testGetPutVerbatimToken(s stores.VerbatimStore) func(*testing.T) {
	return func(t *testing.T) {
		t.Run("with bool token preceded by blanks", checkVerbatimToken(s, "  \t      \n\r\n\n"))
	}
}

func checkBlanks(s stores.VerbatimStore, str string) func(*testing.T) {
	return func(t *testing.T) {
		input := []byte(str)
		expected := stores.MakeStringValue(str)
		h := s.PutBlanks(input)
		outcome := s.Get(h)
		assert.Equal(t, expected, outcome)
	}
}

func checkVerbatimToken(s stores.VerbatimStore, str string) func(*testing.T) {
	return func(t *testing.T) {
		blanks := []byte(str)
		input := token.MakeVerbatimBoolean(true, blanks)
		vh := s.PutVerbatimToken(input)
		outcomeBlanks := s.Get(vh.Blanks())
		expectedBlanks := stores.MakeStringValue(str)
		assert.Equal(t, expectedBlanks, outcomeBlanks)
		outcomeValue := s.Get(vh.Value())
		expectedValue := stores.MakeBoolValue(true)
		assert.Equal(t, expectedValue, outcomeValue)
	}
}
