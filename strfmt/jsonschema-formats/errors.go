package formats

type Error string

const ErrFormat Error = "error in jsonschema formats"

func (e Error) Error() string {
	return string(e)
}
