package lexer

import (
	"bytes"

	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/lexers/token"
)

func (l *L) consumeBoolean(start byte) (token.T, token.T) {
	var buf [3]byte
	if err := l.consumeN(buf[:]); err != nil {
		l.err = codes.ErrInvalidToken

		return token.None, token.None
	}

	var value bool

	// t[rue] | f[als][e]
	switch {
	case start == startOfTrue && bytes.Equal(buf[:], []byte("rue")):
		value = true
	case start == startOfFalse && bytes.Equal(buf[:], []byte("als")):
		if l.consumed >= l.bufferized {
			if err := l.readMore(); err != nil {
				l.err = codes.ErrInvalidToken

				return token.None, token.None
			}
		}
		next := l.buffer[l.consumed]
		l.consumed++
		l.offset++

		if next != 'e' {
			l.err = codes.ErrInvalidToken

			return token.None, token.None
		}

		value = false
	default:
		l.err = codes.ErrInvalidToken

		return token.None, token.None
	}

	return l.lookAhead(token.MakeBoolean(value), 0)
}
