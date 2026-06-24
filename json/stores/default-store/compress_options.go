package store

import (
	"bytes"
	"compress/flate"
	"fmt"
	"io"
)

const (
	defaultCompressionThreshold = 128
	defaultCompressionLevel     = flate.DefaultCompression // i.e. level 6
	stdCompressionLevel         = 6                        // corresponds to the default in compress/flate
	maxCompressionLevel         = 9
	minCompressedSize           = 9 // flate never produces compressed strings smaller then 9 bytes
)

// CompressionOption alter default settings from string compression inside the [Store].
type CompressionOption = func(*compressionOptions)

type compressionOptions struct {
	compressionThreshold int
	compressionLevel     int
	cw                   flateWriter
	dict                 []byte
}

type flateReader interface {
	io.ReadCloser
	flate.Resetter
}

type flateWriter interface {
	io.WriteCloser
	Reset(io.Writer)
}

func applyCompressionOptionsWithDefaults(opts []CompressionOption) compressionOptions {
	o := compressionOptions{
		compressionThreshold: defaultCompressionThreshold,
		compressionLevel:     defaultCompressionLevel,
		// dict defaults to nil (no preset dictionary); cw is built lazily on first compression.
	}

	for _, apply := range opts {
		apply(&o)
	}

	if o.compressionLevel < 0 {
		o.compressionLevel = stdCompressionLevel
	}
	if o.compressionLevel > maxCompressionLevel {
		o.compressionLevel = 9
	}

	return o
}

// compressWriter returns the cached DEFLATE writer, building it on first use from the frozen
// (compressionLevel, dict).
//
// cw is a derived artifact of those two fields, which are frozen for the Store's lifetime, so the
// writer and the dict every reader is seeded with can never drift apart. Building it lazily means a
// caller that never compresses a string — compression disabled, or no value ever exceeding the
// threshold — never pays for the (sizeable) flate writer allocation.
//
// It mutates cw, so it must be called on the write path only; for a [ConcurrentStore] that is under
// the write lock (the only place that compresses).
func (co *compressionOptions) compressWriter() flateWriter {
	if co.cw == nil {
		var wrt bytes.Buffer
		cw, err := flate.NewWriterDict(&wrt, co.compressionLevel, co.dict)
		assertCompressOptionWriter(err)
		co.cw = cw
	}

	return co.cw
}

// Reset prepares the compression configuration for reuse when a [Store] is recycled between
// documents. It is intentionally a no-op.
//
// The compression level, threshold, preset dictionary and the cached writer cw are part of the
// Store's immutable configuration, not of the per-document data held in the arena. They are frozen
// for the Store's whole lifetime so that every payload compressed into the arena stays decodable
// against the same dictionary (see [WithCompressionDict]). Recycling a Store therefore preserves
// them as-is; to change the compression configuration, re-borrow the Store with new options
// ([BorrowStore]), which rebuilds the configuration (and lets cw rebuild lazily from the new
// (level, dict), see [compressionOptions.compressWriter]).
//
// This assumes a Store is never recycled mid-document; see the lifecycle note on [Store].
func (co *compressionOptions) Reset() {}

func WithCompressionLevel(level int) CompressionOption {
	return func(o *compressionOptions) {
		if level < -2 || level > 9 {
			panic(fmt.Errorf("invalid compress level: %d: %w", level, ErrStore))
		}

		o.compressionLevel = level
	}
}

func WithCompressionThreshold(threshold int) CompressionOption {
	return func(o *compressionOptions) {
		o.compressionThreshold = threshold
	}
}

// WithCompressionDict injects a preset DEFLATE dictionary used to seed both string compression and
// decompression in the [Store].
//
// A preset dictionary lifts the compression ratio of short, repetitive payloads by priming the
// DEFLATE window with frequently-seen byte sequences. It is typically trained offline from a
// representative corpus.
//
// # Lifecycle and immutability
//
// The dictionary is frozen for the whole lifetime of the Store: every value the Store compresses and
// every value it later decompresses is bound to this exact dictionary. The Store aliases (does not
// copy) the provided slice and never mutates it; the caller MUST NOT mutate dict while any Store
// seeded with it is alive, otherwise previously-compressed payloads become undecodable. A Store
// recycled from the pool keeps its dictionary (see [Store.Reset]); a gob round-trip carries it along
// so a reloaded Store stays self-consistent.
//
// Because the dictionary is caller-owned, it may outlive the Store and be shared across successive
// Store generations. This is how a compression dictionary "learns": the corpus is trained externally
// and a fresh, frozen dictionary is injected into the next generation — the dict mutates between
// Stores, never within one.
func WithCompressionDict(dict []byte) CompressionOption {
	return func(o *compressionOptions) {
		o.dict = dict
	}
}
