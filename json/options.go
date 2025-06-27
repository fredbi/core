package json

import (
	"io"

	"github.com/fredbi/core/json/lexers"
	lexer "github.com/fredbi/core/json/lexers/default-lexer"
	"github.com/fredbi/core/json/nodes/light"
	"github.com/fredbi/core/json/stores"
	store "github.com/fredbi/core/json/stores/default-store"
	"github.com/fredbi/core/json/writers"
	writer "github.com/fredbi/core/json/writers/default-writer"
)

type Option func(*options)

type options struct {
	store                  stores.Store
	lexerFactory           func([]byte) (lexers.Lexer, func())
	lexerFromReaderFactory func(io.Reader) (lexers.Lexer, func())
	writerFactory          func() (writers.Writer, func())
	writerToWriterFactory  func(io.Writer) (writers.Writer, func())

	// for light nodes
	light.DecodeOptions
	light.EncodeOptions
}

func (o options) LexerFactory() func([]byte) (lexers.Lexer, func()) {
	return o.lexerFactory
}

func (o options) LexerFromReaderFactory() func(io.Reader) (lexers.Lexer, func()) {
	return o.lexerFromReaderFactory
}

func (o options) WriterFactory() func() (writers.Writer, func()) {
	return o.writerFactory
}

func (o options) WriterToWriterFactory() func(io.Writer) (writers.Writer, func()) {
	return o.writerToWriterFactory
}

func defaultLexerFactory(data []byte) (lexers.Lexer, func()) {
	// using default lexer from pool
	l := lexer.BorrowLexerWithBytes(data)

	return l, func() { lexer.RedeemLexer(l) } // TODO: use redeemable lexer to avoid the alloc of the closure
}

func defaultLexerFromReaderFactory(r io.Reader) (lexers.Lexer, func()) {
	// using default lexer from pool
	jl := lexer.BorrowLexerWithReader(r)

	return jl, func() { lexer.RedeemLexer(jl) } // TODO: use redeemable lexer to avoid the alloc of the closure
}

func defaultWriterToWriterFactory(w io.Writer) (writers.Writer, func()) {
	// using default writer from pool
	jw := writer.BorrowWriter(w)

	return jw, func() { writer.RedeemWriter(jw) } // TODO: use redeemable writer to avoid the alloc of the closure
}

func defaultWriterFactory() (writers.Writer, func()) {
	// using default writer from pool
	w := poolOfBuffers.Borrow()
	jw := writer.BorrowWriter(w)

	return jw, func() {
		poolOfBuffers.Redeem(w)
		writer.RedeemWriter(jw)
	} // TODO: use redeemable writer to avoid the alloc of the closure
}

func optionsWithDefaults(opts []Option) options {
	var o options

	for _, apply := range opts {
		apply(&o)
	}

	if o.store == nil {
		o.store = store.New()
	}

	if o.lexerFactory == nil {
		o.lexerFactory = defaultLexerFactory
	}

	if o.lexerFromReaderFactory == nil {
		o.lexerFromReaderFactory = defaultLexerFromReaderFactory
	}

	if o.writerFactory == nil {
		o.writerFactory = defaultWriterFactory
	}

	if o.writerToWriterFactory == nil {
		o.writerToWriterFactory = defaultWriterToWriterFactory
	}

	return o
}

func WithStore(s stores.Store) Option {
	return func(o *options) {
		o.store = s
	}
}

func noop() {
	// no operation func
}

func WithLexer(l lexers.Lexer) Option {
	return func(o *options) {
		o.lexerFactory = func(_ []byte) (lexers.Lexer, func()) {
			return l, noop
		}
		o.lexerFromReaderFactory = func(_ io.Reader) (lexers.Lexer, func()) {
			return l, noop
		}
	}
}

func WithWriter(w writers.Writer) Option {
	return func(o *options) {
		o.writerFactory = func() (writers.Writer, func()) {
			return w, noop
		}
		o.writerToWriterFactory = func(_ io.Writer) (writers.Writer, func()) {
			return w, noop
		}
	}
}
