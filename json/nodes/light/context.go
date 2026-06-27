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

//nolint:gochecknoglobals // private immutable replacer
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
// The [ParentContext] is typically held by the root document and propagated down the hierarchy of
// nodes as it is decoded or encoded.
//
// It is NOT safe for concurrent use. The node machinery is single-goroutine by design: decoding pulls
// from a stateful [lexers.Lexer] and encoding pushes into a stateful [writers.StoreWriter], and the
// context itself mutates as it descends. Decoding several documents in parallel requires one
// [ParentContext] per goroutine.
//
// The set of fields is still in flux.
type ParentContext struct {
	S  stores.Store        // value store backing the decoded/encoded handles
	L  lexers.Lexer        // token source (decode)
	W  writers.StoreWriter // token sink (encode)
	DO DecodeOptions       // decode hooks and options
	EO EncodeOptions       // encode options
	C  *codes.ErrContext   // error context, populated when decoding or encoding fails
	X  any                 // caller scratch space, opaque to the machinery
	P  Path                // JSON Pointer to the current node; valid only during a callback (overwritten on the next sibling, truncated on level exit)
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
