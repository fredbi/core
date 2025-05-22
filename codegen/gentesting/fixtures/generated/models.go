package generated

import (
	"context"
	"encoding/json"
	"strconv"
)

const (
	AConstant = "Constant"
)

type Enum uint8

const (
	EnumOne Enum = iota + 1
	EnumTwo
)

var (
	EnumComplex = []string{"a", "b", "c"}
	EnumValues  = []Enum{EnumOne, EnumTwo}
)

// Model for testing
type Model struct {
	A IntegerCollection `json:"a,omitempty"`
	B string            `json:"b,omitempty"`
}

// NewModel builds a fresh [Model] with default values.
func NewModel() *Model {
	return &Model{
		A: IntegerCollection([]int64{1}),
		B: "b",
	}
}

func (m *Model) Validate(context.Context) error {
	return nil
}

func (m *Model) UnmarshalBinary(data []byte) error {
	var v Model
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*m = v

	return nil
}

func (m Model) MarshalBinary() ([]byte, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return data, nil
}

type IntegerCollection []int64

func MakeIntegerCollection() IntegerCollection {
	return []int64{1}
}

func (m IntegerCollection) String() string {
	return strconv.Itoa(len(m))
}
