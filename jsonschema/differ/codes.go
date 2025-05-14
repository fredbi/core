package differ

type Severity uint8
type Category uint8
type ValidationCategory uint8
type Type uint

const (
	SeverityNone Severity = iota
	SeverityCosmetic
	SeverityDocOnly
	SeverityPatch
	SeverityCompatible
	SeverityBreaking
)

const (
	CategoryNone    Category = iota
	CategoryVersion          // JSON schema version
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
	Changed
)
