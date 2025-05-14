package codes

// errors as immutable values

// NodeError is the error type returned by nodes.
type NodeError string

// Error implements the error interface.
func (e NodeError) Error() string {
	return string(e)
}

// ErrNode is the sentinel error for node-related errors.
const ErrNode NodeError = "node error"
