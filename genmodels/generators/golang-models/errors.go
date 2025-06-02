package models

import "fmt"

type Error string

const (
	ErrModel    Error = "error in golang genmodel"
	ErrSettings Error = "error in settings"
	ErrInit     Error = "error in initialization options"
)

func (e Error) Error() string {
	return string(e)
}

func assertValue(m any) {
	panic(fmt.Errorf("invalid value for %T: %w", m, ErrSettings))
}
