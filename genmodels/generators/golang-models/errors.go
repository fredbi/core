package models

// Error is sentinel error type for all errors originated from this package.
type Error string

const (
	ErrModel Error = "error in golang genmodel"
	ErrInit  Error = "error in initialization options"
)

func (e Error) Error() string {
	return string(e)
}
