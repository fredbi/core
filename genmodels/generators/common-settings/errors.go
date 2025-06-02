package settings

import "fmt"

type Error string

const ErrSettings Error = "errors in settings"

func (e Error) Error() string {
	return string(e)
}

func assertValue(m any) {
	panic(fmt.Errorf("invalid value for %T: %w", m, ErrSettings))
}

func invalidMode(m any, s string) error {
	return fmt.Errorf("invalid string %q for %T: %w", s, m, ErrSettings)
}
