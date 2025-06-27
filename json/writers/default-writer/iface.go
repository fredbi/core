package writer

import (
	"io"

	"github.com/fredbi/core/json/types"
)

// Buffer exposes an abstract buffer that supports writing and escaping from multiple inputs.
//
// At this moment, we provide two implementations of the [Buffer]: [bufio.Unbuffered] and [bufio.ChunkedBuffer].
type Buffer interface {
	BufferRawWriter
	BufferEscapedWriter

	Size() int64

	types.Resettable
	types.WithErrState
}

// BufferRawWriter knows how to write raw bytes, without any escaping.
//
// It is used for example to write separators and numbers, as we know no escaping applies there.
type BufferRawWriter interface {
	WriteSingleByte(data byte)
	WriteBinary(data []byte)
	WriteBinaryFrom(io.Reader)
}

// BufferEscapedWriter knows how to write strings with escaping.
//
// By default, standard JSON escaping applies.
type BufferEscapedWriter interface {
	WriteText(data []byte)
	WriteString(data string)
	WriteTextFrom(io.Reader)
}
