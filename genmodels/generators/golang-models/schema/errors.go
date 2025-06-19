package schema

// Error is sentinel error type for all errors originated from this package.
type Error string

const (
	// ErrSchema is a sentinel error that wraps any error that occurred during generation
	ErrSchema Error = "error in schema generator"

	// ErrInternal indicates an internal error raised by a guard or code assertion, indicating most likely a bug.
	ErrInternal Error = "internal error detected by models schema generator"

	// ErrHint provides supplementary information about the possible cause of another error.
	ErrHint Error = "hint"
)

func (e Error) Error() string {
	return string(e)
}

/*
func errHint(hint string) error {
	msg := "hint: " + hint + ": %w"

	return fmt.Errorf(msg, ErrHint) //nolint:err113
}
*/
