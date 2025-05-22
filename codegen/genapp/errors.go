package genapp

type Error string

func (e Error) Error() string {
	return string(e)
}

const ErrGenApp Error = "code generation app error"
