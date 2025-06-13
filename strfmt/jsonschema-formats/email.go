package formats

import (
	"context"
	"net/mail"

	"github.com/fredbi/core/strfmt"
)

var _ strfmt.Format = &Email{}

func IsEmail(str string) bool {
	_, err := mail.ParseAddress(str)

	return err == nil
}

type Email struct {
	mail.Address
}

func NewEmail() *Email {
	e := MakeEmail()

	return &e
}

func MakeEmail() Email {
	return Email{}
}

type IDNEmail struct {
	mail.Address
}

func (e Email) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}

func (e *Email) UnmarshalText(data []byte) error {
	addr, err := mail.ParseAddress(string(data))
	if err != nil {
		return err
	}
	e.Address = *addr

	return nil
}

func (e Email) Validate(_ context.Context) error {
	_, err := mail.ParseAddress(e.Address.Address)

	return err
}
