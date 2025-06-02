package models

// TargetKind describes a kind of go type to be generated.
//
// Notice that tuples are not native go types and they are normally represented by a struct.
type TargetKind uint8

const (
	TargetKindUnknown TargetKind = iota
	TargetKindPrimitive
	// TargetKindStruct refers to a representation of a JSON object as a go struct
	TargetKindStruct
	// TargetKindSlice refers to a representation of a JSON object as a go slice, or possibly an iter.Seq[V]
	TargetKindSlice
	// TargetKindMap refers to a representation of a JSON array as a go map, or possibly an iter.Seq2[K,V]
	TargetKindMap
	TargetKindInterface
	TargetKindAny
	// TargetKindStream refers to a representation of a file or binary string that is represented as a go io.Reader or io.Writer
	TargetKindStream
	// TargetKindTuple refers to a specific representation for a tuple, e.g. as a go struct, possibly as a go array
	TargetKindTuple
	TargetKindArray
)

// SerializerKindSelector defines the kind of serializer to be generated.
type SerializerSelector uint8

const (
	SerializerNone SerializerSelector = iota
	SerializerAllOf
	SerializerOneOf
	SerializerAnyOf
	SerializerTuple
	SerializerStream
	SerializerInterface
)

func (m SerializerSelector) String() string {
	switch m {
	case SerializerNone:
		return "none"
	case SerializerAllOf:
		return "allOf"
	case SerializerOneOf:
		return "oneOf"
	case SerializerAnyOf:
		return "anyOf"
	case SerializerTuple:
		return "tuple"
	case SerializerStream:
		return "stream"
	case SerializerInterface:
		return "interface"
	default:
		assertValue(m)
		return ""
	}
}

// JSONLibSelector defines the JSON encoding/decoding we prefer to use,
// whenever JSON marshaling is enabled with.
//
// Applicable when MarshalMode includes JSON serialization.
type JSONLibSelector uint8

const (
	JSONStdLib JSONLibSelector = iota
	JSONLibGoCCY
	JSONLibJsoniter
)

func (m JSONLibSelector) String() string {
	switch m {
	case JSONStdLib:
		return "json-stdlib"
	case JSONLibGoCCY:
		return "json-goccy"
	case JSONLibJsoniter:
		return "json-jsoniter"
	default:
		assertValue(m)
		return ""
	}
}

func (m JSONLibSelector) DocString() string {
	switch m {
	case JSONStdLib:
		return "implements JSON serialization using stdlib"
	case JSONLibGoCCY:
		return "implements JSON serialization using github.com/goccy/go-json"
	case JSONLibJsoniter:
		return "implements JSON serialization using github.com/json-iterator/go"
	default:
		assertValue(m)
		return ""
	}
}

type IntegerMappingSelector uint8

const (
	IntegerMappingFixed = iota
	IntegerMappingAsNeeded
	IntegerMappingJSONType
)

func (m IntegerMappingSelector) String() string {
	switch m {
	case IntegerMappingFixed:
		return "fixed"
	case IntegerMappingAsNeeded:
		return "as-needed"
	case IntegerMappingJSONType:
		return "json-type"
	default:
		assertValue(m)
		return ""
	}
}
func (m IntegerMappingSelector) DocString() string {
	switch m {
	case IntegerMappingFixed:
		return "fixed go type for all integers"
	case IntegerMappingAsNeeded:
		return "go type is defined according to validations"
	case IntegerMappingJSONType:
		return "use go-openapi/core/json/types.Number"
	default:
		assertValue(m)
		return ""
	}
}

type DecimalMappingSelector uint8

const (
	DecimalMappingFixed = iota
	DecimalMappingAsNeeded
	DecimalMappingJSONType
)

type MethodKindSelector uint8

const (
	MethodKindGetter MethodKindSelector = iota
	MethodKindSetter
	// MethodKindStringer
	// MethodKindDeepCopier
)

type MarshalMode uint8

const (
	MarshalJSON         MarshalMode = 1 << iota // standard library JSON serialization (several options available)
	MarshalBinary                               // standard library binary serialization
	MarshalEasyJSON                             // JSON serialization using github.com/mailru/easyjson
	MarshalYAML                                 // YAML serialization (several options available)
	MarshalValidateJSON                         // go-openapi serialization with validation
	MarshalSQL                                  // standard library SQL serialization
	MarshalBSON                                 // MongoDB's BSON serialization
	GobEncode                                   // standard library binary serialization
)

func (m MarshalMode) String() string {
	switch m {
	case MarshalBinary:
		return "marshalBinary"
	case MarshalJSON:
		return "marshalJSON"
	case MarshalEasyJSON:
		return "marshalEasyJSON"
	case MarshalYAML:
		return "marshalYAML"
	case MarshalValidateJSON:
		return "marshalValidateJSON"
	case MarshalSQL:
		return "sql"
	case MarshalBSON:
		return "marshalBSON"
	case GobEncode:
		return "gobEncode"
	default:
		assertValue(m)
		return ""
	}
}

func (m MarshalMode) DocString() string {
	switch m {
	case MarshalBinary:
		return "MarshalBinary, UnmarshalBinary interfaces (encoding/binary)"
	case MarshalJSON:
		return "MarshalJSON, UnmarshalJSON interfaces (encoding/json)"
	case MarshalEasyJSON:
		return "MarshalEasyJSON, UnmarshalEasyJSON interfaces (github.com/mailru/easyjson)"
	case MarshalYAML:
		return "MarshalYAML, UnmarshalYAML interface"
	case MarshalValidateJSON:
		return "MarshalValidateJSON, UnmarshalValidateJSON interface (github.com/go-openapi/core/jsonschema/validate"
	case MarshalSQL:
		return "Scan, Value interfaces (database/sql)"
	case MarshalBSON:
		return "MarshalBSON, UnmarshalBSON interfaces (go.mongodb.org/mongo-driver/bson)"
	case GobEncode:
		return "GobEncode, GobDecode interfaces (encoding/gob)"
	default:
		assertValue(m)
		return ""
	}
}
