package providers

import "fmt"

// Error is sentinel error type for all errors originated from this package.
type Error string

const (
	// ErrNameProvider is a sentinel error that wraps any error that occurred during naming decisions
	ErrNameProvider Error = "error in name provider"

	// ErrInternal indicates an internal error raised by a guard or code assertion, indicating most likely a bug.
	ErrInternal Error = "internal error detected by name provider"

	// ErrNotImplemented indicates we hit a code path for which the feature is not supported yet.
	ErrNotImplemented Error = "feature not implemented by the name provider"

	// ErrHint provides supplementary information about the possible cause of another error.
	ErrHint Error = "hint"
)

func (e Error) Error() string {
	return string(e)
}

func errHint(hint string) error {
	msg := "hint: " + hint + ": %w"

	return fmt.Errorf(msg, ErrHint) //nolint:err113
}
