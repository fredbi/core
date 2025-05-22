package gentest

import (
	"fmt"
	"iter"
	"plugin"
	"reflect"
	"strings"
	"testing"
)

var EmptyEntry = Entry{}

type Driver struct {
	p              *plugin.Plugin
	symbols        map[ExportedSymbolKind][]symbol
	identIndex     map[string]symbol
	identKindIndex map[ExportedSymbolKind]map[string]symbol
}

// Symbols iterates over the exported Entries of a given kind from the source package being scrutinized.
func (d *Driver) Symbols(kind ExportedSymbolKind) iter.Seq[Entry] {
	return func(yield func(Entry) bool) {
		for _, symb := range d.symbols[kind] {
			entry := d.makeEntry(symb)
			if !yield(entry) {
				return
			}
		}
	}
}

func (d *Driver) Asserter(t *testing.T) *Asserter {
	return &Asserter{
		Driver: d,
		t:      t,
	}
}

// Lookup for any exported symbol from the tested package.
func (d *Driver) Lookup(ident string) (Entry, bool) {
	symb, ok := d.identIndex[ident]
	if !ok {
		return EmptyEntry, false
	}

	return d.makeEntry(symb), true
}

// Type looks up for an exported type.
func (d *Driver) Type(ident string) (Entry, bool) {
	symb, ok := d.identKindIndex[SymbolType][ident]
	if !ok {
		return EmptyEntry, false
	}

	return d.makeEntry(symb), true
}

// Const looks up for an exported constant.
func (d *Driver) Const(ident string) (Entry, bool) {
	symb, ok := d.identKindIndex[SymbolConst][ident]
	if !ok {
		return EmptyEntry, false
	}

	return d.makeEntry(symb), true
}

// Var looks up for an exported variable.
func (d *Driver) Var(ident string) (Entry, bool) {
	symb, ok := d.identKindIndex[SymbolVar][ident]
	if !ok {
		return EmptyEntry, false
	}

	return d.makeEntry(symb), true
}

// Func looks up for an exported function.
func (d *Driver) Func(ident string) (Entry, bool) {
	symb, ok := d.identKindIndex[SymbolFunc][ident]
	if !ok {
		return EmptyEntry, false
	}

	return d.makeEntry(symb), true
}

// makeEntry builds a new entry from the original exported symbol.
//
// This creates a new instance of the variable for types and variables, but not for constants and functions,
// which should not mutate.
func (d *Driver) makeEntry(symb symbol) Entry {
	resolved, err := d.p.Lookup(symb.Ident)
	if err != nil {
		panic(fmt.Errorf("internal error: cannot resolve symbol from plugin: %q", symb.Ident))
	}

	ptr := reflect.ValueOf(resolved)
	if ptr.Kind() != reflect.Pointer { // TODO: test when exported type is already a pointer
		panic(fmt.Errorf("internal error: resolved symbol from plugin expected to be a pointer: %q", symb.Ident))
	}

	value := reflect.Indirect(ptr)
	var iface any
	if value.CanInterface() {
		// normally, we can't access exported things that can't "interface", apart perhaps (todo: remove perhaps) for embedded fields
		iface = value.Interface() // I guess there a few types for which we can't to that. TODO : test more thoroughly for edge cases
	}

	// instantiate a new variable of the same type
	if symb.Kind != SymbolConst && symb.Kind != SymbolFunc {
		value = cloneReflectValue(value)
	}

	return Entry{
		symbol: symb,
		iface:  iface,
		rtype:  value.Type(),
		rvalue: value,
	}
}

func cloneReflectValue(value reflect.Value) reflect.Value {
	clone := reflect.New(value.Type())

	return reflect.Indirect(clone)
}

type Entry struct {
	symbol

	iface  any
	rtype  reflect.Type
	rvalue reflect.Value
}

func (e Entry) IsStruct() bool    { return e.rtype.Kind() == reflect.Struct }
func (e Entry) IsSlice() bool     { return e.rtype.Kind() == reflect.Slice }
func (e Entry) IsMap() bool       { return e.rtype.Kind() == reflect.Map }
func (e Entry) IsArray() bool     { return e.rtype.Kind() == reflect.Array }
func (e Entry) IsInterface() bool { return e.rtype.Kind() == reflect.Interface }
func (e Entry) IsFunction() bool  { return e.rtype.Kind() == reflect.Func }
func (e Entry) IsEmpty() bool     { return e.Kind == noSymbol }
func (e Entry) IsOfType(name string) bool {
	return false // TODO
}

func (e Entry) HasField(name string) bool {
	if !e.IsStruct() {
		return false
	}

	v := e.rvalue.FieldByName(name)

	return v.IsValid()
}

// Field returns an Entry for a struct field.
//
// TODO: test when field comes from another (nested) package.
func (e Entry) Field(name string) Entry {
	if !e.HasField(name) {
		return EmptyEntry
	}

	value := e.rvalue.FieldByName(name)
	var iface any
	if value.CanInterface() {
		// normally, we can't access exported things that can't "interface", apart perhaps (todo: remove perhaps) for embedded or anonymous fields
		iface = value.Interface() // I guess there a few types for which we can't to that. TODO : test more thoroughly for edge cases
	}

	// instantiate a new variable of the same type
	if value.Kind() != reflect.Func {
		value = cloneReflectValue(value)
	}

	fieldType := value.Type().Name()
	nameParts := strings.Split(fieldType, ".")
	var fieldPkg string
	var fieldTypeName string
	if len(nameParts) > 1 {
		fieldPkg = nameParts[0]
		fieldTypeName = nameParts[1]
	} else {
		fieldPkg = e.Pkg
		fieldTypeName = fieldType
	}
	return Entry{
		symbol: symbol{
			Kind:  SymbolType,
			Pkg:   fieldPkg,
			Ident: fieldTypeName,
		},
		iface:  iface,
		rtype:  value.Type(),
		rvalue: value,
	}
}

func (e Entry) Interface() any       { return e.iface }
func (e Entry) Value() reflect.Value { return e.rvalue }

func (e Entry) HasMethod(name string) bool {
	v := e.rvalue.MethodByName(name)

	return v.IsValid()
}

// TODO
func (e Entry) ElementType() Entry { return Entry{} }
func (e Entry) KeyType() Entry     { return Entry{} }
