package store

import (
	"bytes"
	"compress/flate"
	"fmt"
	"io"
)

const (
	defaultCompressionThreshold = 128
	defaultDictSize             = 512
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
		dict:                 make([]byte, 0, defaultDictSize),
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

	if o.cw == nil {
		// compression writer is initialized only once, whereas the compression reader is borrowed from a pool every
		// time we need it.
		var (
			err error
			wrt bytes.Buffer
		)
		o.cw, err = flate.NewWriterDict(&wrt, o.compressionLevel, o.dict)
		assertCompressOptionWriter(err)
	}

	return o
}

// Reset restores the default compression settings, for recycling a [Store] between documents.
//
// It deliberately does NOT rebuild the cached compression writer cw: cw still encodes the level and
// dictionary it was created with. This is consistent as long as the Store is recycled with the
// default compression configuration (the only configuration reachable today, since dict is always
// empty and no option sets it). If a Store was configured with non-default compression options, it
// must be re-borrowed with those options ([BorrowStore]) so a fresh cw is built — Reset alone would
// leave cw and these fields out of sync.
//
// This assumes a Store is never recycled mid-document; see the lifecycle note on [Store].
func (co *compressionOptions) Reset() {
	co.dict = co.dict[:0]
	co.compressionLevel = defaultCompressionLevel
	co.compressionThreshold = defaultCompressionThreshold
}

func (co *compressionOptions) isCompressionInitialized() bool {
	return co.cw != nil
}

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
