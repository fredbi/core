package generated

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/fredbi/core/codegen/gentesting/fixtures/generated/folder"
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
	concreteBaseType

	isDefined bool
	A         IntegerCollection `json:"a,omitempty"`
	B         string            `json:"b,omitempty"`
	C         *string           `json:"c,omitempty"`
	D         folder.SubModel   `json:"d"`
}

// NewModel builds a fresh [Model] with default values.
func NewModel() *Model {
	return &Model{
		A: IntegerCollection([]int64{1}),
		B: "b",
	}
}

func (m Model) IsDefined() bool {
	return m.isDefined
}

func (m *Model) SetUndefined() {
	m.isDefined = false
}

func (m *Model) SetDefined() {
	m.isDefined = true
}

func (m *Model) Validate(context.Context) error {
	return nil
}

func (m *Model) UnmarshalBinary(data []byte) error {
	var v Model
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	v.isDefined = true

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

type ArrayOfModels [3]Model

type ComposedModel struct {
	Model

	folder.SubModel
}

type ModelKind string

const (
	ModelKindNone  ModelKind = ""
	ModelKindModel ModelKind = "Model"
	ModelKindOther ModelKind = "Other"
)

type BaseType interface {
	ModelKind() ModelKind
	CommonProperty() string
	SetCommonProperty(string)
}

type concreteBaseType struct {
	discriminator  ModelKind
	commonProperty string
}

func (c concreteBaseType) ModelKind() ModelKind { return c.discriminator }

func (c concreteBaseType) CommonProperty() string {
	return c.commonProperty
}

func (c *concreteBaseType) SetCommonProperty(commonProperty string) {
	c.commonProperty = commonProperty
}

func UnmarshalJSONBaseType(data []byte) (BaseType, error) {
	return nil, errors.New("not imlemented")
}
