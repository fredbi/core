package light

// I don't think we really need options here
type DecodeOptions struct {
	decodeHooks
	uniqueKey bool
}

type EncodeOptions struct {
	// omitEmpty bool // should not be needed
}
