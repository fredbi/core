package writer

import (
	"io"
)

func trueBytes() []byte  { return []byte("true") }
func falseBytes() []byte { return []byte("false") }

// Bool writes a boolean value as JSON.
func (w *W) Bool(v bool) {
	if w.err != nil {
		return
	}

	if v {
		w.buffer.WriteBinary(trueBytes())
		w.err = w.buffer.Err()

		return
	}

	w.buffer.WriteBinary(falseBytes())
	w.err = w.buffer.Err()
}

// Raw appends raw bytes to the buffer, without quotes and without escaping.
func (w *W) Raw(data []byte) {
	if w.err != nil || len(data) == 0 {
		return
	}

	w.buffer.WriteBinary(data)
	w.err = w.buffer.Err()
}

// String writes a string as a JSON string value enclosed by double quotes, with escaping.
//
// The empty string is a legit input.
func (w *W) String(s string) {
	if w.err != nil {
		return
	}

	w.buffer.WriteSingleByte('"')
	w.buffer.WriteString(s)
	w.buffer.WriteSingleByte('"')
	w.err = w.buffer.Err()
}

// StringBytes writes a slice of bytes as a JSON string enclosed by double quotes ('"'), with escaping.
//
// An empty slice is a legit input.
func (w *W) StringBytes(data []byte) {
	if w.err != nil || data == nil {
		return
	}

	w.buffer.WriteSingleByte('"')
	w.buffer.WriteText(data)
	w.buffer.WriteSingleByte('"')
	w.err = w.buffer.Err()
}

// StringRunes writes a slice of bytes as a JSON string enclosed by double quotes ('"'), with escaping.
//
// An empty slice is a legit input.
func (w *W) StringRunes(data []rune) {
	if w.err != nil || data == nil {
		return
	}
	s := string(data)
	w.buffer.WriteSingleByte('"')
	w.buffer.WriteString(s)
	w.buffer.WriteSingleByte('"')
	w.err = w.buffer.Err()
}

// NumberBytes writes a slice of bytes as a JSON number.
//
// No check is carried out.
func (w *W) NumberBytes(data []byte) {
	if w.err != nil || len(data) == 0 {
		return
	}

	w.buffer.WriteBinary(data)
	w.err = w.buffer.Err()
}

// StringCopy writes the bytes consumed from an [io.Reader] as a JSON string enclosed by double quotes ('"'), with escaping.
func (w *W) StringCopy(r io.Reader) {
	if w.err != nil {
		return
	}

	w.buffer.WriteSingleByte('"')
	w.buffer.WriteTextFrom(r)
	w.buffer.WriteSingleByte('"')
	w.err = w.buffer.Err()
}

// NumberCopy writes the bytes consumed from an [io.Reader] as a JSON number.
//
// No check is carried out.
func (w *W) NumberCopy(r io.Reader) {
	w.RawCopy(r)
}

// RawCopy writes the bytes consumed from an [io.Reader], without quotes and without escaping.
func (w *W) RawCopy(r io.Reader) {
	if w.err != nil {
		return
	}

	w.buffer.WriteBinaryFrom(r)
	w.err = w.buffer.Err()
}
