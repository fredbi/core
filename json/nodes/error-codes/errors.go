package codes

// errors as immutable values

// NodeError is the error type returned by nodes.
type NodeError string

// Error implements the error interface.
func (e NodeError) Error() string {
	return string(e)
}

const (
	// ErrNode is the sentinel error for node-related errors.
	ErrNode NodeError = "node error"

	// ErrBuilder is the sentinel error for node-builder errors
	ErrBuilder NodeError = "node builder error"
)
