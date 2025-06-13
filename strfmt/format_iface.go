package strfmt

import (
	"context"
	"encoding"
)

// Format is the interface for all types that implement a JSON schema string format.
type Format interface {
	String() string
	Validate(context.Context) error

	encoding.TextMarshaler
	encoding.TextUnmarshaler
}
