package model

import "fmt"

type Error string

const (
	ErrSettings Error = "error in settings"
)

func (e Error) Error() string {
	return string(e)
}

func assertValue(m any) {
	panic(fmt.Errorf("invalid value for %T: %w", m, ErrSettings))
}
