package providers

import (
	"fmt"
	"strconv"

	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/nodes"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

// NameEnumValue provides a legit go name for a constant or variable corresponding to a value in an enum.
//
// Examples:
//
// With user-provided names:
//
//	  analyzed:
//		   enum:
//			   - x
//			   - y
//		   x-go-enums:
//		     - x-axis    -> XAxis
//		     - y-axis    -> YAxis
//
// The intent is to be able to generate go constructs like:
//
//		  type Analyzed string
//
//			const (
//				XAxis Analyzed = "x-axis"
//				YAxis Analyzed = "y-axis"
//			)
//
//	   var AnalyzedEnumValues = []Analyzed{
//	     XAxis,
//	     YAxis,
//	   }
//
// With numerical values
//
//	  analyzed:
//		   enum:
//			   - 1          -> One
//			   - 2.5        -> TwoPointFive
//			   - 1.23e2     -> OneHundredTwentyThree
//			   - -1         -> MinusOne
//
// The intent is to be able to generate go constructs like:
//
//		  type Analyzed float64
//
//			const (
//				One Analyzed = 1.0
//				TwoPointFive Analyzed = 2.5
//				OneHundredTwentyThree Analyzed = 123
//				MinusOne Analyzed = -1.00
//			)
//
//	   var AnalyzedEnumValues = []Analyzed{
//	     One,
//	     TwoPointFive,
//	     OneHundredTwentyThree,
//	     MinusOne,
//	   }
//
// With anonymous objects:
//
//	   analyzed:
//		   enum:
//			   - {1,2}      -> AnalyzedEnum0
//			   - [x,y]      -> AnalyzedEnum1
//
// Here, go does not allow us to use constants. We would generate variables instead:
//
//		 type Analyzed any
//
//	   var (
//	     AnalyzedEnum0 Analyzed
//	     AnalyzedEnum1 Analyzed
//	   )
//
//	   func init() {
//	     if err := AnalyzedEnum0.UnmarshalJSON(`{1,2}`) ; err != nil {
//	       panic(err)
//	     }
//
//	     if err := AnalyzedEnum1.UnmarshalJSON(`["a","b"]`) ; err != nil {
//	       panic(err)
//	     }
//	   }
//
//	   var AnalyzedEnumValues = []Analyzed{
//	     AnalyzedEnum0,
//	     AnalyzedEnum1,
//	   }
//
// With incomplete user-provided name:
//
//	  analyzed:
//		   enum:
//			   - {1,2}      -> First
//			   - [x,y]      -> AnalyzedEnum1
//		   x-go-enums:
//		     - First
//
// TODO: the user may opt-in to make (some of) these unexported.
// TODO: needs a deconflicter too
func (p NameProvider) NameEnumValue(index int, enumValue json.Document, analyzed structural.AnalyzedSchema) (string, error) {
	const directive = "x-go-enums"

	if ext, isUserDefined := analyzed.GetExtension(directive); isUserDefined {
		names := ext.([]string)

		if index < len(names) {
			return p.mangler.ToGoName(names[index]), nil
		}
	}

	switch enumValue.Kind() {
	case nodes.KindScalar:
		v, _ := enumValue.Value()
		switch v.Kind() {
		case token.String, token.Boolean, token.Null:
			return p.mangler.ToGoName(v.String()), nil // may be deconflicted later
		case token.Number:
			return p.mangler.ToGoName(p.mangler.SpellNumber(enumValue.String())), nil
		}
	default:
		// enum value is a complex schema, not a scalar
		// determine a name for an anonymous schema, which is not a root
		parent := analyzed.Parent()

		// walk up dependencies until we find a named schema
		parentName, err := p.NameSchema(parent.Name(), parent)
		if err != nil {
			return "", fmt.Errorf("unable to determine a name for this schema: %v", analyzed)
		}

		if parentName == "" {
			return "", fmt.Errorf("unable to determine a name for this schema: %v", analyzed)
		}

		return p.mangler.ToGoName(parentName + " enum" + strconv.Itoa(index)), nil
	}

	return "", nil
}
