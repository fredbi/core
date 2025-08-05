package overlay

import (
	"fmt"
	"strings"

	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/jsonpath"
	"github.com/fredbi/core/json/nodes/light"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/stores/values"
	"github.com/fredbi/core/jsonschema/analyzers"
)

// Version is a valid version of the overlay specification.
type Version uint8

const (
	VersionUndefined Version = iota
	Version10
)

func (ov Version) String() string {
	switch ov {
	case VersionUndefined:
		return "undefined"
	case Version10:
		return "1.0.0"
	default:
		panic("unsupported overlay dialect version")
	}
}

func (ov Version) All() []Version {
	return []Version{
		VersionUndefined, Version10,
	}
}

func (ov *Version) Decode(s stores.Store, n light.Node) error {
	if !n.IsString(s) {
		return fmt.Errorf("overlay version must be a string: %w", ErrOverlay)
	}

	v, _ := n.Value(s)
	version := v.String()

	switch version {
	case Version10.String():
		*ov = Version10
		return nil
	case "":
		fallthrough
	case VersionUndefined.String():
		*ov = VersionUndefined
		return nil
	default:
		return fmt.Errorf(
			"expected overlay version to be either empty or to match a supported version (%v), but got %q: %w",
			ov.All(), version, ErrOverlay,
		)
	}
}

// Info provides metadata about a [jsonschema.Overlay].
type Info struct {
	s          stores.Store
	title      stores.Handle
	version    stores.Handle
	extensions analyzers.Extensions
}

// Reset the [Info] so it may be recycled.
func (i *Info) Reset() {
	i.s = nil
	i.title = stores.HandleZero
	i.version = stores.HandleZero
}

// Title is a human readable description of the purpose of the [Overlay].
func (i Info) Title() string {
	return i.s.Get(i.title).String()
}

// Version identifer for indicating changes to the [Overlay] document.
func (i Info) Version() string {
	return i.s.Get(i.version).String()
}

// Extensions returns a map of "x-*" extensions in this [Info] object.
func (i Info) Extensions() analyzers.Extensions {
	return i.extensions
}

var (
	titleKey       = values.MakeInternedKey("title")
	versionKey     = values.MakeInternedKey("version")
	targetKey      = values.MakeInternedKey("target")
	descriptionKey = values.MakeInternedKey("description")
	updateKey      = values.MakeInternedKey("update")
	removeKey      = values.MakeInternedKey("remove")
)

func (i *Info) Decode(s stores.Store, n light.Node) error {
	if !n.IsObject() {
		return fmt.Errorf(
			"info must be an object: %w",
			ErrOverlay,
		)
	}

	i.s = s

	for iKey, iNode := range n.Pairs() {
		switch iKey {
		case titleKey:
			if !iNode.IsString(s) {
				return fmt.Errorf("title should be a string:%w", ErrOverlay)
			}
			i.title, _ = iNode.Handle()
		case versionKey:
			if !iNode.IsString(s) {
				return fmt.Errorf("info version should be a string:%w", ErrOverlay)
			}
			i.version, _ = iNode.Handle()
		default:
			// x-* extensions
			if ext := iKey.String(); strings.HasPrefix(ext, "x-") {
				if i.extensions == nil {
					i.extensions = make(analyzers.Extensions)
				}
				doc := json.NewBuilder(s).WithRoot(iNode).Document()
				i.extensions.Add(ext, doc)
			}
		}

		// other keys remain part of the document, but uninterpreted (perhaps TODO AdditionalProperties?)
	}

	return nil
}

// Action represents one or more changes to be applied to the target document at the location defined by the target JSONPath expression.
type Action struct {
	s           stores.Store
	description stores.Handle
	target      jsonpath.Expression
	remove      bool
	update      json.Document
	extensions  analyzers.Extensions
}

// Description of the action. [CommonMark] syntax MAY be used for rich text representation.
func (a Action) Description() string {
	return a.s.Get(a.description).String()
}

func (a Action) Remove() bool {
	return a.remove
}

// Target is a JSONPath expression selecting nodes in the target document.
func (a Action) Target() jsonpath.Expression {
	return a.target
}

// Update target specification.
//
// If the target selects an object node, the value of this field MUST be an object
// with the properties and values to merge with the selected node.
//
// If the target selects an array, the value of this field MUST be an entry to append to the array.
//
// This field has no impact if the remove field of this action object is true.
func (a Action) Update() json.Document {
	return a.update
}

func (a Action) Extensions() analyzers.Extensions {
	return a.extensions
}

func (a *Action) Decode(s stores.Store, e light.Node) error {
	a.s = s
	a.update = json.EmptyDocument
	hasRequiredTarget := false

	for aKey, aNode := range e.Pairs() {
		switch aKey {
		case targetKey:
			hasRequiredTarget = true
			if !aNode.IsString(s) {
				return fmt.Errorf("action target should be a string: %w", ErrOverlay)
			}
			v, _ := aNode.Value(s)
			expr, err := jsonpath.MakeExpression(v.String())
			if err != nil {
				return fmt.Errorf("action target should be a valid JSONpath expresion: %w: %w", err, ErrOverlay)
			}
			a.target = expr
		case descriptionKey:
			if !aNode.IsString(s) {
				return fmt.Errorf("action description should be a string: %w", ErrOverlay)
			}
			a.description, _ = aNode.Handle()
		case updateKey:
			doc := json.NewBuilder(s).WithRoot(aNode).Document()
			a.update = doc
		case removeKey:
			if !aNode.IsBool(s) {
				return fmt.Errorf("action remove should be a boolean: %w", ErrOverlay)
			}
			v, _ := aNode.Value(s)
			a.remove = v.Bool()
		default:
			// x-* extensions
			if ext := aKey.String(); strings.HasPrefix(ext, "x-") {
				if a.extensions == nil {
					a.extensions = make(analyzers.Extensions)
				}
				doc := json.NewBuilder(s).WithRoot(aNode).Document()
				a.extensions.Add(ext, doc)
			}
		}

		// other keys remain part of the document, but uninterpreted
	}

	if !hasRequiredTarget {
		return fmt.Errorf("target is required in action: %w", ErrOverlay)
	}

	return nil
}
