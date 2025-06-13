package genapp

// Error is a sentinel error type to wrap all errors raised by the genapp package.
type Error string

func (e Error) Error() string {
	return string(e)
}

// ErrGenApp is a sentinel error for all errors raised by the genapp package.
const ErrGenApp Error = "code generation app error"
