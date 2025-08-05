package differ

// Severity qualifies the impact of a change, from cosmetic to breaking change.
type Severity uint8

// CategoryMode qualifies the nature of a schema change, such as version of jsonchema, metadata only, validations only etc.
//
// Several categories may apply to a single change.
type CategoryMode uint8

// ValidationCategory indicates more precisely the type of validation to which a change with ValidationCategory is eligible.
type ValidationCategory uint8

// Type of change, e.g. update, addition or deletion.
type Type uint

const (
	SeverityNone Severity = iota
	SeverityCosmetic
	SeverityDocOnly
	SeverityPatch
	SeverityMinor // compatible
	SeverityBreaking
)

func (s Severity) Less(other Severity) bool {
	return int(s) < int(other)
}

const (
	CategoryNone    CategoryMode = 1 << iota
	CategoryVersion              // JSON schema version
	CategoryMetadata
	CategoryLocation // change in $ref
	CategoryValidation
	CategoryDataType
)

const (
	NumberValidation ValidationCategory = iota
	StringValidation
	ObjectValidation
	ArrayValidation
	EnumValidation
)

const (
	TypeNoChange Type = iota
	Deleted
	Added
	Updated
)
