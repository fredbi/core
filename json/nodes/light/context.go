package light

import (
	"strconv"
	"strings"

	"github.com/fredbi/core/json/lexers"
	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/stores/values"
	"github.com/fredbi/core/json/writers"
)

// Context holds the lexer's offset for every decoded token.
//
// Context does not apply to a [Node] built programmatically with a [Builder].
type Context struct {
	offset uint64
}

type pathElemKind uint8

const (
	pathElemString pathElemKind = 1
	pathElemInt    pathElemKind = 2
)

type stringOrInt struct {
	kind pathElemKind
	s    values.InternedKey
	i    int
}

type Path []stringOrInt

var EmptyPath = []stringOrInt{}

var pthEscaper = strings.NewReplacer(
	"~",
	"~0",
	"/",
	"~1",
)

func (p Path) String() string {
	var w strings.Builder

	for _, e := range p {
		w.WriteByte('/')
		if e.kind == pathElemString {
			w.WriteString(pthEscaper.Replace(e.s.String()))

			continue
		}
		w.WriteString(strconv.Itoa(e.i))
	}

	return w.String()
}

func addKeyToPath(original, p Path, key values.InternedKey) Path {
	if len(p) == len(original) {
		return append(p, stringOrInt{
			kind: pathElemString,
			s:    key,
		})
	}

	if len(p) == 0 {
		panic("assert")
	}

	p[len(p)-1] = stringOrInt{
		kind: pathElemString,
		s:    key,
	}

	return p
}

func addElemToPath(original, p Path, idx int) Path {
	if len(p) == len(original) {
		return append(p, stringOrInt{
			kind: pathElemInt,
			i:    idx,
		})
	}

	if len(p) == 0 {
		panic("assert")
	}

	p[len(p)-1] = stringOrInt{
		kind: pathElemInt,
		i:    idx,
	}

	return p
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
	X  any
	P  Path
}

func (p *ParentContext) Reset() {
	p.S = nil
	p.L = nil
	p.W = nil
	p.DO = DecodeOptions{}
	p.EO = EncodeOptions{}
	p.C = nil
	p.X = nil
	if len(p.P) > 0 {
		p.P = p.P[:0]
	}
}
