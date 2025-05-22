package golang

import "fmt"

type Error string

const (
	ErrSettings Error = "errors in settings"
	ErrInit     Error = "error in initialization options"
)

func (e Error) Error() string {
	return string(e)
}

func assertValue(m any) {
	panic(fmt.Errorf("invalid value for %T: %w", m, ErrSettings))
}
