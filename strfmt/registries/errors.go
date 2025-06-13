package registries

import "fmt"

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrFormat         Error = "string format error"
	ErrFormatNotFound Error = "this registry doesn't support format"
)

func ErrNotFound(format string) error {
	return fmt.Errorf("format %q: %w", format, ErrFormatNotFound)
}
