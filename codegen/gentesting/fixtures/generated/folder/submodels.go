package folder

type SubEnum string

const (
	SubConstantX SubEnum = "x"
	SubConstantY SubEnum = "y"
)

var SubEnumValues []SubEnum

func init() {
	SubEnumValues = []SubEnum{
		SubConstantX,
		SubConstantY,
	}
}

type SubModel struct {
	X  int
	Y  Content
	XY SubEnum
}

func NewSubModel() *SubModel { return &SubModel{} }

type Content struct {
	internal int
	Z        string
}

func NewContent() *Content { return &Content{} }
