package lexer

import (
	"bytes"

	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/lexers/token"
)

func (in *Input) consumeBoolean(start byte) token.T {
	var buf [3]byte
	if err := in.consumeN(buf[:]); err != nil {
		in.err = codes.ErrInvalidToken

		return token.None
	}

	var value bool

	// t[rue] | f[als][e]
	switch {
	case start == startOfTrue && bytes.Equal(buf[:], []byte("rue")):
		value = true
	case start == startOfFalse && bytes.Equal(buf[:], []byte("als")):
		if in.consumed >= in.bufferized {
			if err := in.readMore(); err != nil {
				in.err = codes.ErrInvalidToken

				return token.None
			}
		}
		next := in.buffer[in.consumed]
		in.consumed++
		in.offset++

		if next != 'e' {
			in.err = codes.ErrInvalidToken

			return token.None
		}

		value = false
	default:
		in.err = codes.ErrInvalidToken

		return token.None
	}

	return token.MakeBoolean(value)
}
