package store

// The header of th uint64 that a [stores.Handle] gives us
// is encoded on 4 bits. This leaves 16 possibilities to encode your piece of data.

const (
	headerNull uint8 = iota
	headerFalse
	headerTrue
	headerInlinedNumber
	headerInlinedASCII
	headerInlinedString
	headerNumber
	headerString
	headerCompressedString
	headerInlinedBlank
	headerCompressedBlank
	headerInlinedCompressedString // unused for now: DEFLATE produces strings of at least 9 bytes
)
