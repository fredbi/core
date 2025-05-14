package store

import (
	"encoding/binary"
	"slices"

	"github.com/fredbi/core/json/stores"
)

const asciiBits = 7

// inlined extracts the size and payload from a handle representing a JSON value.
//
//nolint:gosec
func inlined(h stores.Handle) (size int, payload uint64) {
	size = int((h & smallMask) >> headerBits)
	payload = uint64((h & payloadMask) >> (headerBits + smallBits))

	return
}

// inlined extracts the size and payload from a handle representing blank space.
//
//nolint:gosec
func inlinedBlanks(h stores.Handle) (size int, payload uint64) {
	size = int((h & blankMask) >> headerBits)
	payload = uint64((h & blankPayloadMask) >> (headerBits + blankBits))

	return
}

// packString packs a small string into a uint64
func packString(value []byte) uint64 {
	return packBytes(value)
}

func ensureEmptyBuffer(size int, buffer ...[]byte) []byte {
	if len(buffer) == 0 || buffer[0] == nil {
		return make([]byte, 0, size)
	}

	out := slices.Grow(buffer[0], size)

	return out[:0]
}

// unpackString retrieves the string packed in a uint64
func unpackString(size int, payload uint64, buffer ...[]byte) []byte {
	if size == 0 {
		return nil
	}
	var buf [maxInlineBytes + 1]byte
	binary.LittleEndian.PutUint64(buf[:], payload)
	out := ensureEmptyBuffer(size, buffer...)
	out = append(out, buf[:size]...)

	return out
}

// isOnlyASCII checks if a slice of bytes of length 8 contains only ASCII characters
func isOnlyASCII(value []byte) bool {
	assertInlineASCIISize(value)

	buf := (*[8]byte)(value)
	checked := binary.LittleEndian.Uint64(buf[:])
	const mask = uint64(0x7f7f7f7f7f7f7f7f)

	return checked == checked&mask
}

// packASCII packs a slice of 8 bytes containing ASCII characters (encoded on 7 bits) into a uint64
//
//nolint:mnd
func packASCII(value []byte) uint64 {
	assertInlineASCIISize(value)

	var r uint64
	r = uint64(value[0])
	r |= uint64(value[1]) << asciiBits
	r |= uint64(value[2]) << (2 * asciiBits)
	r |= uint64(value[3]) << (3 * asciiBits)
	r |= uint64(value[4]) << (4 * asciiBits)
	r |= uint64(value[5]) << (5 * asciiBits)
	r |= uint64(value[6]) << (6 * asciiBits)
	r |= uint64(value[7]) << (7 * asciiBits)

	return r
}

// unpackASCII unpacks a uint64 with encoded ASCII characters into the original 8 bytes as a slice.
//
// A preallocated buffer may be provided. Otherwise, the function allocates a slice of bytes to store the result.
//
//nolint:mnd
func unpackASCII(size int, payload uint64, buffer ...[]byte) []byte {
	assertInlineASCIIUnpackSize(size)

	var out []byte
	if len(buffer) == 0 || buffer[0] == nil {
		var buf [maxInlineBytes + 1]byte
		out = buf[:]
	} else {
		out = slices.Grow(buffer[0], maxInlineBytes+1)
		out = out[:maxInlineBytes+1]
	}

	out[0] = byte((payload & uint64(0x000000000000007f)))
	out[1] = byte((payload & uint64(0x0000000000003f80)) >> asciiBits)
	out[2] = byte((payload & uint64(0x00000000001fc000)) >> (2 * asciiBits))
	out[3] = byte((payload & uint64(0x000000000fe00000)) >> (3 * asciiBits))
	out[4] = byte((payload & uint64(0x00000007f0000000)) >> (4 * asciiBits))
	out[5] = byte((payload & uint64(0x000003f800000000)) >> (5 * asciiBits))
	out[6] = byte((payload & uint64(0x0001fc0000000000)) >> (6 * asciiBits))
	out[7] = byte((payload & uint64(0x00fe000000000000)) >> (7 * asciiBits))

	return out[:maxInlineBytes+1]
}

// packBCD packs a slice of BCD nibbles into a uint64.
func packBCD(value []byte) uint64 {
	return packBytes(value)
}

// unpackBCD unpacks a number packed as a BCD string in a uint64.
//
// A preallocated buffer may be provided. Otherwise, the function allocates a slice of bytes to store the result.
func unpackBCD(size int, payload uint64, buffer ...[]byte) []byte {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], payload)

	return decodeBCDAsNumber(buf[:size], buffer...)
}

// packBytes packs up to 7 bytes in a uint64
func packBytes(value []byte) uint64 {
	assertInlinePackBytes(value)

	var buf [8]byte
	copy(buf[:], value)

	return binary.LittleEndian.Uint64(buf[:])
}

const (
	blankEncoding = byte(iota)
	tabEncoding
	lineFeedEncoding
	carriageReturnEncoding
)

const (
	blankEncodingMask = uint64(0b11)
)

// packBlanks packs up to 28 blank characters (7*4) in a uint64
func packBlanks(value []byte) uint64 {
	assertInlinePackBlanks(value)

	var r uint64
	var offsetBits int
	for _, b := range value {
		switch b {
		case blank:
			r |= (uint64(blankEncoding) << offsetBits)
		case tab:
			r |= (uint64(tabEncoding) << offsetBits)
		case lineFeed:
			r |= (uint64(lineFeedEncoding) << offsetBits)
		case carriageReturn:
			r |= (uint64(carriageReturnEncoding) << offsetBits)
		default:
			assertVerbatimIsBlank(b)
		}
		offsetBits += bitsPerBlank
	}

	return r
}

// unpackBlanks retrieves the string of blank characters packed in a uint64.
//
// A preallocated buffer may be provided. Otherwise, the function allocates a slice of bytes to store the result.
func unpackBlanks(size int, payload uint64, buffer ...[]byte) []byte {
	var out []byte
	if len(buffer) == 0 || buffer[0] == nil {
		out = make([]byte, 0, size) // this allocation is returned to the caller. Can't recycle it.
	} else {
		out = slices.Grow(buffer[0], size)
		out = out[:0]
	}

	for offsetBits := 0; offsetBits < size*bitsPerBlank; offsetBits += bitsPerBlank {
		u := byte(payload >> offsetBits & blankEncodingMask)
		switch u {
		case blankEncoding:
			out = append(out, blank)
		case tabEncoding:
			out = append(out, tab)
		case lineFeedEncoding:
			out = append(out, lineFeed)
		case carriageReturnEncoding:
			out = append(out, carriageReturn)
		}
	}

	return out
}
