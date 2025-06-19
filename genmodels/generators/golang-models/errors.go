package models

import "fmt"

// Error is sentinel error type for all errors originated from this package.
type Error string

const (
	// ErrModel is a sentinel error that wraps any error that occurred during generation
	ErrModel Error = "error in golang genmodel"

	// ErrInit indicates an error during the initialization stage (config loading, etc)
	ErrInit Error = "error in initialization options"

	// ErrInternal indicates an internal error raised by a guard or code assertion, indicating most likely a bug.
	ErrInternal Error = "internal error detected by models generator"

	// ErrHint provides supplementary information about the possible cause of another error.
	ErrHint Error = "hint"

	// ErrNotImplemented indicates we hit a code path for which the feature is not supported yet.
	ErrNotImplemented Error = "feature not implemented by the models generator"
)

func (e Error) Error() string {
	return string(e)
}

func errHint(hint string) error {
	msg := "hint: " + hint + ": %w"

	return fmt.Errorf(msg, ErrHint) //nolint:err113
}
