package types

import "slices"

// Setting is an interface for types which may be used in settings structures.
type Setting interface {
	String() string
	DocString() string
}

/*
// Selector is an interface for types which may be used as settings from a codegen template.
type Selector interface {
	//Selected(...string) bool // TODO: improve this: goal is to be able to find a selected option from a template
	Setting
	Is(...Selector) bool
}

type Mode interface {
	Setting
	Has(...Mode) bool
}
*/

// UniqueID of a generated type
type UniqueID string

type ModelsDependencies map[UniqueID][]UniqueID

// InformationReport represents additional information produced by the
// code generator to track decisions and understand how code was generated.
type InformationReport struct {
	DecisionType string
	Decision     string
	Originator   string   // program/func signature
	Sources      []string // JSON pointers to source schema or config
}

type ModeSettingConstraint interface {
	~uint8
}

type SelectorSettingConstraint interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func HasMode[T ModeSettingConstraint](m T, modes ...T) bool {
	var p T
	for _, mode := range modes {
		p |= mode
	}

	return m&p > 0
}

func IsSelector[T SelectorSettingConstraint](m T, selected ...T) bool {
	return slices.Contains(selected, m)
}

func HasString[T ModeSettingConstraint](m T, conv func(string) (T, error), shorts ...string) bool {
	modes := make([]T, 0, len(shorts))
	for _, short := range shorts {
		mode, err := conv(short)
		if err != nil {
			continue
		}
		modes = append(modes, mode)
	}

	return HasMode(m, modes...)
}
