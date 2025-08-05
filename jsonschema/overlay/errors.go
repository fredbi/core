package overlay

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrOverlay Error = "error in schema overlay"
)
