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
