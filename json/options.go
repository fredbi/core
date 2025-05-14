package json

import (
	"io"

	"github.com/fredbi/core/json/lexers"
	lexer "github.com/fredbi/core/json/lexers/default-lexer"
	"github.com/fredbi/core/json/nodes/light"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/writers"
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

func defaultLexerFactory(data []byte) (lexers.Lexer, func()) {
	// using default lexer from pool
	l := lexer.BorrowLexerWithBytes(data)

	return l, func() { lexer.RedeemLexer(l) }
}

func optionsWithDefaults(opts []Option) options {
	var o options

	for _, apply := range opts {
		apply(&o)
	}

	if o.lexerFactory == nil {
		o.lexerFactory = defaultLexerFactory
	}

	return o
}
