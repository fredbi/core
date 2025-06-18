package models

// Error is sentinel error type for all errors originated from this package.
type Error string

const (
	// ErrModel is a sentinel error that wraps any error that occurred during generation
	ErrModel Error = "error in golang genmodel"

	// ErrInit indicates an error during the initialization stage (config loading, etc)
	ErrInit Error = "error in initialization options"
)

func (e Error) Error() string {
	return string(e)
}
