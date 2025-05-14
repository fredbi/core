package lexer

import "io"

var noopReader io.Reader = eofReader{}

// eofReader is a dummy reader that returns EOF upon call to Read()
type eofReader struct {
}

func (r eofReader) Read(_ []byte) (int, error) {
	return 0, io.EOF
}
