package jsonschema

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrSchema Error = "error in schema"
)
