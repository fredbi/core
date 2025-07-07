package store

import (
	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/stores/values"
	"github.com/fredbi/core/json/writers"
)

const (
	headerBits     = 4
	lengthBits     = 20
	smallBits      = 4
	maxInlineBytes = 7

	headerMask = stores.Handle(uint64(1)<<headerBits - 1)

	// length field for offset in arena
	lengthMask = stores.Handle(uint64(1)<<lengthBits-1) << headerBits
	offsetMask = ^stores.Handle(0) & ^headerMask & ^lengthMask

	// length field for inlined payload
	smallMask   = stores.Handle(uint64(1)<<smallBits-1) << headerBits
	payloadMask = ^stores.Handle(0) & ^headerMask & ^smallMask // 7 bytes
)

// Store is the default implementation for [stores.Store].
//
// It acts an in-memory store for JSON values, with an emphasis on compactness.
//
// # Concurrency
//
// It safe to retrieve values concurrently with [store.Get], but it is unsafe to have
// several go routines storing content concurrently.
//
// [store.WriteTo] should not be used concurrently.
type Store struct {
	options
	arena []byte
	_     struct{}
}

var _ stores.Store = &Store{} // [Store] implements [stores.Store]

// New [Store].
//
// See [Option] to alter default settings.
func New(opts ...Option) *Store {
	s := &Store{
		options: applyOptionsWithDefaults(opts),
	}

	s.arena = make([]byte, 0, s.minArenaSize)

	return s
}

func (s *Store) Len() int {
	return len(s.arena)
}

// Get a [values.Value] from a [stores.Handle].
func (s *Store) Get(h stores.Handle) values.Value {
	header := uint8(h & headerMask) //nolint:gosec

	switch header {
	case headerNull:
		return values.NullValue
	case headerFalse:
		return values.FalseValue
	case headerTrue:
		return values.TrueValue
	case headerInlinedNumber: // small number inlined
		return s.getInlinedNumber(h)
	case headerInlinedASCII: // small ascii string inlined: 8 bytes exactly
		return s.getInlinedASCII(h)
	case headerInlinedString: // small string inlined
		return s.getInlinedString(h)
	case headerNumber: // large number
		return s.getLargeNumber(h)
	case headerString: // large string
		return s.getLargeString(h)
	case headerCompressedString: // large compressed string
		return s.getCompressedString(h)
	case headerInlinedCompressedString: // small compressed string
		// this case is not active: flate's minimum size is 9 bytes
		return s.getInlinedCompressedString(h)
	default:
		assertValidHeader(header)
		return values.NullValue
	}
}

// WriteTo writes the value pointed to be the [stores.Handle] to a JSON [writers.StoreWriter].
//
// This avoids unnessary buffering when transferring the value down to the writer.
func (s *Store) WriteTo(writer writers.StoreWriter, h stores.Handle) {
	header := uint8(h & headerMask) //nolint:gosec

	switch header {
	case headerNull:
		writer.Null()
	case headerFalse:
		writer.Bool(false)
	case headerTrue:
		writer.Bool(true)
	case headerInlinedNumber: // small number inlined
		size, payload := inlined(h)
		buffer, redeem := borrowBytesWithRedeem(
			size * digitsPerByte,
		) // amortize the allocation of this temporary buffer
		buffer = unpackBCD(size, payload, buffer)
		writer.NumberBytes(buffer) // sends the buffer directly to the writer
		redeem()
	case headerInlinedASCII: // small ascii string inlined: 8 bytes exactly
		size, payload := inlined(h) // 7 bytes
		var buffer [8]byte
		out := unpackASCII(size, payload, buffer[:])
		writer.StringBytes(out)
	case headerInlinedString: // small string inlined
		size, payload := inlined(h) // 0-7 bytes (0-8 packed characters)
		if size == 0 {
			writer.String("")
			return
		}
		var buffer [8]byte
		out := unpackString(size, payload, buffer[:])
		writer.StringBytes(out)
	case headerNumber: // large number
		size, offset := withOffset(h)
		assertOffsetInArena(offset, len(s.arena))

		nibbles := s.arena[offset : offset+size]
		buffer, redeem := borrowBytesWithRedeem(size * digitsPerByte)
		buffer = decodeBCDAsNumber(nibbles, buffer)
		writer.NumberBytes(buffer)
		redeem()
	case headerString: // large string
		size, offset := withOffset(h)
		assertOffsetInArena(offset, len(s.arena))

		strBytes := s.arena[offset : offset+size]
		writer.StringBytes(strBytes)
	case headerCompressedString: // large compressed string
		size, offset := withOffset(h)
		assertOffsetInArena(offset, len(s.arena))

		inflater, redeem := s.uncompressStringReader(s.arena[offset : offset+size])
		writer.StringCopy(inflater)
		redeem()
	case headerInlinedCompressedString: // small compressed string
		// this case is not active: flate's minimum size is 9 bytes
		size, payload := inlined(h) // 0-7 bytes
		var buffer [8]byte
		out := unpackString(size, payload, buffer[:])
		inflater, redeem := s.uncompressStringReader(out)
		writer.StringCopy(inflater)
		redeem()
	default:
		assertValidHeader(header)
	}
}

// PutToken puts a value inside a [token.T] and returns its [stores.Handle] for later retrieval.
func (s *Store) PutToken(tok token.T) stores.Handle {
	switch tok.Kind() {
	case token.Null:
		return s.PutNull()

	case token.Boolean:
		return s.PutBool(tok.Bool())

	case token.Number:
		return s.putNumber(tok.Value())

	case token.String, token.Key:
		return s.putString(tok.Value())

	default:
		assertValidToken(tok)
		return stores.Handle(headerNull)
	}
}

// PutValue puts a [values.Value] and returns its [stores.Handle] for later retrieval.
func (s *Store) PutValue(v values.Value) stores.Handle {
	switch v.Kind() {
	case token.Null:
		return s.PutNull()

	case token.Boolean:
		return s.PutBool(v.Bool())

	case token.Number:
		return s.putNumber(v.NumberValue().Value)

	case token.String, token.Key:
		return s.putString(v.StringValue().Value)

	default:
		assertValidValue(
			v,
		) // moved to guards: it is normally not possible to build an invalid values.Value
		return stores.Handle(headerNull)
	}
}

// PutNull is a shorthand for putting a null value. The returned [stores.Handle] is always 0.
func (s *Store) PutNull() stores.Handle {
	return stores.Handle(headerNull)
}

// PutNull is a shorthand for putting a bool value.
func (s *Store) PutBool(b bool) stores.Handle {
	if b {
		return stores.Handle(headerTrue)
	}
	return stores.Handle(headerFalse)
}

// Reset the [Store] to its initial state.
//
// This is useful to recycle [Store] s from a memory pool.
//
// Implements [pools.Resettable].
func (s *Store) Reset() {
	s.arena = s.arena[:0]
	s.options.Reset()
}

func (s *Store) getInlinedNumber(h stores.Handle) values.Value {
	size, payload := inlined(h)
	buffer := s.getBuffer(maxInlineBytes + 1)

	return values.MakeRawValue(token.MakeWithValue(token.Number, unpackBCD(size, payload, buffer)))
}

func (s *Store) getInlinedASCII(h stores.Handle) values.Value {
	size, payload := inlined(h) // 7 bytes (0-8 packed characters)
	// convention: in this case, size is always equal to 8
	buffer := s.getBuffer(maxInlineBytes + 1)

	return values.MakeRawValue(
		token.MakeWithValue(token.String, unpackASCII(size, payload, buffer)),
	)
}

func (s *Store) getInlinedString(h stores.Handle) values.Value {
	size, payload := inlined(h) // 0-7 bytes
	if size == 0 {
		return values.EmptyStringValue
	}
	buffer := s.getBuffer(maxInlineBytes + 1)

	return values.MakeRawValue(
		token.MakeWithValue(token.String, unpackString(size, payload, buffer)),
	)
}

func (s *Store) getLargeNumber(h stores.Handle) values.Value {
	size, offset := withOffset(h)
	assertOffsetInArena(offset, len(s.arena))
	nibbles := s.arena[offset : offset+size]
	buffer := s.getBuffer(digitsPerByte * size)

	return values.MakeRawValue(
		token.MakeWithValue(token.Number, decodeBCDAsNumber(nibbles, buffer)),
	)
}

func (s *Store) getLargeString(h stores.Handle, _ ...[]byte) values.Value {
	size, offset := withOffset(h)
	assertOffsetInArena(offset, len(s.arena))
	strBytes := s.arena[offset : offset+size]

	return values.MakeRawValue(token.MakeWithValue(token.String, strBytes))
}

func (s *Store) getInlinedCompressedString(h stores.Handle) values.Value {
	size, payload := inlined(h) // 0-7 bytes
	var buf [8]byte
	out := unpackString(size, payload, buf[:])
	uncompressed := s.uncompressString(out) // if we manage to get there some day, provide buffer

	return values.MakeRawValue(token.MakeWithValue(token.String, uncompressed))
}

func (s *Store) getCompressedString(h stores.Handle) values.Value {
	size, offset := withOffset(h)
	assertOffsetInArena(offset, len(s.arena))

	buffer := s.getBuffer(s.uncompressRatioHeuristic(size))
	uncompressed := s.uncompressString(s.arena[offset:offset+size], buffer)

	return values.MakeRawValue(token.MakeWithValue(token.String, uncompressed))
}

func (s *Store) putNumber(value []byte) stores.Handle {
	nibbles, redeem := borrowBytesWithRedeem(nibbleSize(value))
	defer redeem()
	nibbles = encodeNumberAsBCD(value, nibbles)
	if len(nibbles) <= maxInlineBytes {
		return s.putInlinedNumber(nibbles)
	}

	return s.putLargeNumber(nibbles)
}

func (s *Store) putString(value []byte) stores.Handle {
	l := len(value)

	switch {
	case l <= maxInlineBytes:
		return s.putInlinedString(value)
	case l == maxInlineBytes+1 && isOnlyASCII(value):
		return s.putInlinedASCIIString(value)
	case s.compressionThreshold > 0 && l > s.compressionThreshold:
		return s.putCompressedString(value)
	default:
		return s.putLargeString(value)
	}
}

func (s *Store) putInlinedString(value []byte) stores.Handle {
	// inlined string (up to 7 bytes)
	l := len(value)
	const headerPart = uint64(headerInlinedString)
	lengthPart := uint64(l) << headerBits
	payload := packString(value) << (headerBits + smallBits)

	return stores.Handle(headerPart | lengthPart | payload)
}

func (s *Store) putInlinedASCIIString(value []byte) stores.Handle {
	// inlined 8 bytes ASCII-only string
	const headerPart = uint64(headerInlinedASCII)
	const lengthPart = uint64(maxInlineBytes) << headerBits
	payload := packASCII(value) << (headerBits + smallBits)

	return stores.Handle(headerPart | lengthPart | payload)
}

func (s *Store) putLargeString(value []byte) stores.Handle {
	// long string put into arena
	l := len(value)
	const headerPart = uint64(headerString)
	lengthPart := uint64(l) << headerBits
	offsetPart := uint64(len(s.arena)) << (headerBits + lengthBits)
	s.arena = append(s.arena, value...)

	return stores.Handle(headerPart | lengthPart | offsetPart)
}

func (s *Store) putCompressedString(value []byte) stores.Handle {
	buffer, redeem := borrowBytesWithRedeem(len(value))
	defer redeem()
	compressed := s.compressString(value, buffer)
	l := len(compressed)

	if l > maxInlineBytes {
		const headerPart = uint64(headerCompressedString)
		lengthPart := uint64(l) << headerBits
		offsetPart := uint64(len(s.arena)) << (headerBits + lengthBits)
		s.arena = append(s.arena, compressed...)

		return stores.Handle(headerPart | lengthPart | offsetPart)
	}

	// this part is never active: min length of a compressed string is 9 bytes
	const headerPart = uint64(headerInlinedCompressedString)
	lengthPart := uint64(l) << headerBits
	payload := packString(compressed) << (headerBits + smallBits)

	return stores.Handle(headerPart | lengthPart | payload)
}

func (s *Store) putInlinedNumber(nibbles []byte) stores.Handle {
	// inlined number
	const headerPart = uint64(headerInlinedNumber)
	sizePart := uint64(len(nibbles)) << headerBits
	payload := packBCD(nibbles) << (headerBits + smallBits)

	return stores.Handle(headerPart | sizePart | payload)
}

func (s *Store) putLargeNumber(nibbles []byte) stores.Handle {
	// BCD number put into arena
	const headerPart = uint64(headerNumber)
	lengthPart := uint64(len(nibbles)) << headerBits
	offsetPart := uint64(len(s.arena)) << (headerBits + lengthBits)
	s.arena = append(s.arena, nibbles...)

	return stores.Handle(headerPart | lengthPart | offsetPart)
}

// withOffset extracts the size and offset in arena from a handle
//
//nolint:gosec
func withOffset(h stores.Handle) (size int, offset int) {
	size = int((h & lengthMask) >> headerBits)
	offset = int(h&offsetMask) >> (headerBits + lengthBits)
	assertOffsetAddressable(
		offset,
	) // impossible on 64-bit systems, theoretically possible on 32-bits systems if the handle is corrupted.

	return
}
