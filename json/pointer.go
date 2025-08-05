package json

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/fredbi/core/json/nodes"
	"github.com/fredbi/core/json/nodes/light"
	"github.com/fredbi/core/json/stores/values"
)

// EmptyPointer represents the empty JSON pointer, which matches a whole document.
var EmptyPointer = Pointer([]stringOrInt{})

var (
	// replacers that implement JSON pointer escaping and unescaping rules

	pthEscaper = strings.NewReplacer(
		"~",
		"~0",
		"/",
		"~1",
	)

	pthUnescaper = strings.NewReplacer(
		"~1",
		"/",
		"~0",
		"~",
	)
)

type pointerError string

func (e pointerError) Error() string {
	return string(e)
}

const (
	// ErrPointer is an error raised by a JSON pointer.
	ErrPointer pointerError = "JSON pointer error"

	// ErrPointerNotFound is raised when a JSON pointer cannot be resolved against a document.
	ErrPointerNotFound pointerError = "JSON pointer not found"

	// ErrInvalidStart states that a JSON pointer must start with a separator ("/"), or be the empty JSON pointer.
	ErrInvalidStart pointerError = `JSON pointer must be empty or start with a /"`
)

// GetPointer returns the JSON [Document] pointed by a JSON [Pointer] inside the current [Document],
// or an error if it is not found.
func (d Document) GetPointer(p Pointer) (Document, error) {
	if len(p) == 0 {
		return d, nil
	}

	node, err := d.getNodePointer(d.root, p)
	if err != nil {
		return EmptyDocument, errors.Join(err, ErrPointerNotFound)
	}

	return Document{
		options: d.options,
		document: document{
			root: node,
		},
	}, nil
}

func errPointerGotKey(k values.InternedKey) error {
	return fmt.Errorf("expected a numerical index to search an array, but got %q instead", k.String())
}

func errPointerNoIndex(i int) error {
	return fmt.Errorf("searching element %d in array, but was not found", i)
}

// TODO: delegate to light.Node?
func (d Document) getNodePointer(root light.Node, p Pointer) (current light.Node, err error) {
	current = root

	for _, e := range p {
		if current.Kind() == nodes.KindObject {
			if e.kind&pathElemString == 0 {
				return current, errPointerGotIndex(e.i)
			}

			n, ok := current.AtInternedKey(e.s)
			if !ok {
				return current, errPointerNoKey(e.s)
			}

			current = n

			continue
		}

		if current.Kind() != nodes.KindArray || e.kind&pathElemInt == 0 {
			return current, errPointerGotKey(e.s)
		}

		n, ok := current.Elem(e.i)
		if !ok {
			return current, errPointerNoIndex(e.i)
		}

		current = n
	}

	return current, nil
}

// TODO: delegate to light.Node?
func (d *Document) setNodePointer(root light.Node, p Pointer, value light.Node) (current light.Node, err error) {
	current = root

	for _, e := range p {
		if current.Kind() == nodes.KindObject {
			if e.kind&pathElemString == 0 {
				return current, errPointerGotIndex(e.i)
			}

			n, ok := current.AtInternedKey(e.s)
			if !ok {
				return current, errPointerNoKey(e.s)
			}

			current = n

			continue
		}

		if current.Kind() != nodes.KindArray || e.kind&pathElemInt == 0 {
			return current, errPointerGotKey(e.s)
		}

		n, ok := current.Elem(e.i)
		if !ok {
			return current, errPointerNoIndex(e.i)
		}

		current = n
	}

	return current, nil
}

// JSONLookup implements the classical [github.com/go-openapi/jsonpointer.JSONPointable] interface, so users
// of this package can resolve JSON pointers against [Document] s.
//
// The returned value is always a JSON [Document].
func (d Document) JSONLookup(pointer string) (any, error) {
	p, err := MakePointer(pointer) // TODO: borrow from pool
	if err != nil {
		return nil, err
	}

	return d.GetPointer(p)
}

// Pointer represents a JSON Pointer.
type Pointer []stringOrInt

type pathElemKind uint8

const (
	pathElemString      pathElemKind = 1
	pathElemInt         pathElemKind = 2
	pathElemStringOrInt pathElemKind = 3
)

type stringOrInt struct {
	kind pathElemKind
	s    values.InternedKey
	i    int
}

// MakePointer builds a JSON [Pointer] from its string representation.
//
// RFC6901 definition of a JSON pointer:
//
//   - may be empty
//   - if not empty, must start by "/"
//   - all tokens are separated by "/"
//   - "/" is escaped by "~1"
//   - "~" is escaped by "~0"
//   - tokens representing a numerical array index are non-negative integers
//   - an integer digit may be "0" or any integer without a leading "0"
//
// Notice that this definition of a JSON pointer does not yield a unique match:
// token "123" would both match key "123" in an object or item 123 in an array.
func MakePointer(s string) (Pointer, error) {
	if s == "" {
		return EmptyPointer, nil
	}

	unrooted, ok := strings.CutPrefix(s, "/")
	if !ok {
		return nil, errors.Join(ErrInvalidStart, ErrPointer)
	}

	tokens := strings.Split(unrooted, "/")
	p := make(Pointer, len(tokens))

	for i, token := range tokens {
		unescaped := pthUnescaper.Replace(token)
		p[i].s = values.MakeInternedKey(unescaped)
		idx := asNumber(unescaped)
		if idx < 0 {
			p[i].kind = pathElemString
			continue
		}
		p[i].i = idx
		p[i].kind = pathElemStringOrInt
	}

	return p, nil
}

func asNumber(s string) int {
	l := len(s)
	if l == 0 {
		return -1
	}
	if s[0] < '0' || s[0] > '9' {
		return -1
	}
	if s[0] == '0' && l > 1 {
		return -1
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}

	return n
}

// MakePointerFromElements buils a JSON [Pointer] from a list of elements
// that constitute the search path.
//
// Elements may be of type string, [values.InternedKey] or int.
//
// String elements are not escaped.
//
// Integer elements only apply to arrays. In this representation, we no longer have an ambiguous
// search that could match a key with the string representation of the integer element.
func MakePointerFromElements(elems ...any) (Pointer, error) {
	if len(elems) == 0 {
		return EmptyPointer, nil
	}

	p := make(Pointer, len(elems))

	for i, e := range elems {
		switch te := e.(type) {
		case string:
			idx := asNumber(te)
			if idx < 0 {
				p[i] = stringOrInt{
					kind: pathElemString,
					s:    values.MakeInternedKey(te),
				}
				continue
			}

			p[i] = stringOrInt{
				kind: pathElemStringOrInt,
				s:    values.MakeInternedKey(te),
				i:    idx,
			}
		case values.InternedKey:
			idx := asNumber(te.String())
			if idx < 0 {
				p[i] = stringOrInt{
					kind: pathElemString,
					s:    te,
				}
				continue
			}

			p[i] = stringOrInt{
				kind: pathElemStringOrInt,
				s:    te,
				i:    idx,
			}
		case int:
			p[i] = stringOrInt{
				kind: pathElemInt,
				i:    te,
			}
		default:
			return nil, ErrPointer
		}
	}

	return p, nil
}

// String representation of a JSON pointer, with escaping rules as per RFC 6901
func (p Pointer) String() string {
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

func errPointerGotIndex(i int) error {
	return fmt.Errorf("expected a path key string to search an object, but got %d instead", i)
}

func errPointerNoKey(k values.InternedKey) error {
	return fmt.Errorf("searching path key %q in object, but was not found", k.String())
}
