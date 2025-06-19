package model

import "fmt"

type Error string

const (
	ErrSettings Error = "error in settings"

	// ErrInternal indicates an internal error raised by a guard or code assertion, indicating most likely a bug.
	ErrInternal Error = "internal error detected by models generator in data-model"
)

func (e Error) Error() string {
	return string(e)
}

func assertValue(m any) {
	panic(fmt.Errorf("invalid value for %T: %w", m, ErrSettings))
}
