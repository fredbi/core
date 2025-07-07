package light

import (
	"github.com/fredbi/core/json/lexers"
	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/writers"
)

// Context holds the lexer's offset for every decoded token.
//
// Context does not apply to a [Node] built programmatically with a [Builder].
type Context struct {
	offset uint64
}

// Offset of this node in the original JSON stream.
func (c Context) Offset() uint64 {
	return c.offset
}

// ParentContext injects all the dependencies needed to operate with a [Node].
//
// The [ParentContext] is typically held by the root document, and propagated down to the hierarchy of nodes.
type ParentContext struct {
	S  stores.Store
	L  lexers.Lexer
	W  writers.StoreWriter
	DO DecodeOptions
	EO EncodeOptions
	C  *codes.ErrContext
}

func (p *ParentContext) Reset() {
	p.S = nil
	p.L = nil
	p.W = nil
	p.DO = DecodeOptions{}
	p.EO = EncodeOptions{}
	p.C = nil
}
