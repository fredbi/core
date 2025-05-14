package writer

import (
	"io"
	"strconv"
)

// Byte writes a single byte
func (w *W) Byte(c byte) {
	w.buffer.AppendByte(c)
}

// From appends raw binary data to the buffer or sets the error if it is given.
//
// Useful for calling with results of MarshalJSON-like functions.
func (w *W) From(data []byte, err error) {
	switch {
	case w.err != nil:
		return
	case err != nil:
		w.err = err
	case len(data) > 0:
		w.buffer.AppendBytes(data)
	default:
		w.RawString("null")
	}
}

func (w *W) Uint8Str(n uint8) {
	w.buffer.EnsureSpace(3)
	w.buffer.Buf = append(w.buffer.Buf, '"')
	w.buffer.Buf = strconv.AppendUint(w.buffer.Buf, uint64(n), 10)
	w.buffer.Buf = append(w.buffer.Buf, '"')
}

func (w *W) Uint16Str(n uint16) {
	w.buffer.EnsureSpace(5)
	w.buffer.Buf = append(w.buffer.Buf, '"')
	w.buffer.Buf = strconv.AppendUint(w.buffer.Buf, uint64(n), 10)
	w.buffer.Buf = append(w.buffer.Buf, '"')
}

func (w *W) Uint32Str(n uint32) {
	w.buffer.EnsureSpace(10)
	w.buffer.Buf = append(w.buffer.Buf, '"')
	w.buffer.Buf = strconv.AppendUint(w.buffer.Buf, uint64(n), 10)
	w.buffer.Buf = append(w.buffer.Buf, '"')
}

func (w *W) UintStr(n uint) {
	w.buffer.EnsureSpace(20)
	w.buffer.Buf = append(w.buffer.Buf, '"')
	w.buffer.Buf = strconv.AppendUint(w.buffer.Buf, uint64(n), 10)
	w.buffer.Buf = append(w.buffer.Buf, '"')
}

func (w *W) Uint64Str(n uint64) {
	w.buffer.EnsureSpace(20)
	w.buffer.Buf = append(w.buffer.Buf, '"')
	w.buffer.Buf = strconv.AppendUint(w.buffer.Buf, n, 10)
	w.buffer.Buf = append(w.buffer.Buf, '"')
}

func (w *W) UintptrStr(n uintptr) {
	w.buffer.EnsureSpace(20)
	w.buffer.Buf = append(w.buffer.Buf, '"')
	w.buffer.Buf = strconv.AppendUint(w.buffer.Buf, uint64(n), 10)
	w.buffer.Buf = append(w.buffer.Buf, '"')
}

func (w *W) Int8Str(n int8) {
	w.buffer.EnsureSpace(4)
	w.buffer.Buf = append(w.buffer.Buf, '"')
	w.buffer.Buf = strconv.AppendInt(w.buffer.Buf, int64(n), 10)
	w.buffer.Buf = append(w.buffer.Buf, '"')
}

func (w *W) Int16Str(n int16) {
	w.buffer.EnsureSpace(6)
	w.buffer.Buf = append(w.buffer.Buf, '"')
	w.buffer.Buf = strconv.AppendInt(w.buffer.Buf, int64(n), 10)
	w.buffer.Buf = append(w.buffer.Buf, '"')
}

func (w *W) Int32Str(n int32) {
	w.buffer.EnsureSpace(11)
	w.buffer.Buf = append(w.buffer.Buf, '"')
	w.buffer.Buf = strconv.AppendInt(w.buffer.Buf, int64(n), 10)
	w.buffer.Buf = append(w.buffer.Buf, '"')
}

func (w *W) IntStr(n int) {
	w.buffer.EnsureSpace(21)
	w.buffer.Buf = append(w.buffer.Buf, '"')
	w.buffer.Buf = strconv.AppendInt(w.buffer.Buf, int64(n), 10)
	w.buffer.Buf = append(w.buffer.Buf, '"')
}

func (w *W) Int64Str(n int64) {
	w.buffer.EnsureSpace(21)
	w.buffer.Buf = append(w.buffer.Buf, '"')
	w.buffer.Buf = strconv.AppendInt(w.buffer.Buf, n, 10)
	w.buffer.Buf = append(w.buffer.Buf, '"')
}

func (w *W) Float32(n float32) {
	w.buffer.EnsureSpace(20)
	w.buffer.Buf = strconv.AppendFloat(w.buffer.Buf, float64(n), 'g', -1, 32)
}

func (w *W) Float32Str(n float32) {
	w.buffer.EnsureSpace(20)
	w.buffer.Buf = append(w.buffer.Buf, '"')
	w.buffer.Buf = strconv.AppendFloat(w.buffer.Buf, float64(n), 'g', -1, 32)
	w.buffer.Buf = append(w.buffer.Buf, '"')
}

func (w *W) Float64(n float64) {
	w.buffer.EnsureSpace(20)
	w.buffer.Buf = strconv.AppendFloat(w.buffer.Buf, n, 'g', -1, 64)
}

func (w *W) Float64Str(n float64) {
	w.buffer.EnsureSpace(20)
	w.buffer.Buf = append(w.buffer.Buf, '"')
	w.buffer.Buf = strconv.AppendFloat(w.buffer.Buf, float64(n), 'g', -1, 64)
	w.buffer.Buf = append(w.buffer.Buf, '"')
}

// RawText encloses raw binary data in quotes and appends in to the buffer.
// Useful for calling with results of MarshalText-like functions.
func (w *W) RawText(data []byte, err error) {
	switch {
	case w.err != nil:
		return
	case err != nil:
		w.err = err
	case len(data) > 0:
		w.String(string(data))
	default:
		w.RawString("null")
	}
}

// Base64Bytes appends data to the buffer after base64 encoding it
func (w *W) Base64Bytes(data []byte) {
	if data == nil {
		w.buffer.AppendString("null")
		return
	}
	w.buffer.AppendByte('"')
	w.base64(data)
	w.buffer.AppendByte('"')
}

func (w *W) Uint8(n uint8) {
	w.buffer.EnsureSpace(3)
	w.buffer.Buf = strconv.AppendUint(w.buffer.Buf, uint64(n), 10)
}

func (w *W) Uint16(n uint16) {
	w.buffer.EnsureSpace(5)
	w.buffer.Buf = strconv.AppendUint(w.buffer.Buf, uint64(n), 10)
}

func (w *W) Uint32(n uint32) {
	w.buffer.EnsureSpace(10)
	w.buffer.Buf = strconv.AppendUint(w.buffer.Buf, uint64(n), 10)
}

func (w *W) Uint(n uint) {
	w.buffer.EnsureSpace(20)
	w.buffer.Buf = strconv.AppendUint(w.buffer.Buf, uint64(n), 10)
}

func (w *W) Uint64(n uint64) {
	w.buffer.EnsureSpace(20)
	w.buffer.Buf = strconv.AppendUint(w.buffer.Buf, n, 10)
}

func (w *W) Int8(n int8) {
	w.buffer.EnsureSpace(4)
	w.buffer.Buf = strconv.AppendInt(w.buffer.Buf, int64(n), 10)
}

func (w *W) Int16(n int16) {
	w.buffer.EnsureSpace(6)
	w.buffer.Buf = strconv.AppendInt(w.buffer.Buf, int64(n), 10)
}

func (w *W) Int32(n int32) {
	w.buffer.EnsureSpace(11)
	w.buffer.Buf = strconv.AppendInt(w.buffer.Buf, int64(n), 10)
}

func (w *W) Int(n int) {
	w.buffer.EnsureSpace(21)
	w.buffer.Buf = strconv.AppendInt(w.buffer.Buf, int64(n), 10)
}

func (w *W) Int64(n int64) {
	w.buffer.EnsureSpace(21)
	w.buffer.Buf = strconv.AppendInt(w.buffer.Buf, n, 10)
}

const encode = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
const padChar = '='

func (w *W) base64(in []byte) {
	if len(in) == 0 {
		return
	}

	w.buffer.EnsureSpace(((len(in)-1)/3 + 1) * 4)

	si := 0
	n := (len(in) / 3) * 3

	for si < n {
		// Convert 3x 8bit source bytes into 4 bytes
		val := uint(in[si+0])<<16 | uint(in[si+1])<<8 | uint(in[si+2])

		w.buffer.Buf = append(w.buffer.Buf, encode[val>>18&0x3F], encode[val>>12&0x3F], encode[val>>6&0x3F], encode[val&0x3F])

		si += 3
	}

	remain := len(in) - si
	if remain == 0 {
		return
	}

	// Add the remaining small block
	val := uint(in[si+0]) << 16
	if remain == 2 {
		val |= uint(in[si+1]) << 8
	}

	w.buffer.Buf = append(w.buffer.Buf, encode[val>>18&0x3F], encode[val>>12&0x3F])

	switch remain {
	case 2:
		w.buffer.Buf = append(w.buffer.Buf, encode[val>>6&0x3F], byte(padChar))
	case 1:
		w.buffer.Buf = append(w.buffer.Buf, byte(padChar), byte(padChar))
	}
}

// DumpTo outputs the data to given io.Writer, resetting the buffer.
func (w *W) DumpTo(out io.Writer) (written int, err error) {
	return w.buffer.DumpTo(out)
}

// BuildBytes returns writer data as a single byte slice. You can optionally provide one byte slice
// as argument that it will try to reuse.
func (w *W) BuildBytes(reuse ...[]byte) ([]byte, error) {
	if w.err != nil {
		return nil, w.err
	}

	return w.buffer.BuildBytes(reuse...), nil
}

// ReadCloser returns an io.ReadCloser that can be used to read the data.
// ReadCloser also resets the buffer.
func (w *W) ReadCloser() (io.ReadCloser, error) {
	if w.err != nil {
		return nil, w.err
	}

	return w.buffer.ReadCloser(), nil
}

// RawByte appends raw binary data to the buffer.
func (w *W) RawString(s string) {
	w.buffer.AppendString(s)
}
