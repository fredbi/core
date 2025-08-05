package stubs

import (
	"math/big"
	"regexp"
)

type FakerContext struct {
	// * random generator
	// * available classes for strings
}

type Faker interface {
	String(FakerContext, ...StringOption) []byte
	Number(FakerContext, ...NumberOption) []byte
	Bool(FakerContext) bool
}

type StringOption func(*stringOptions)
type NumberOption func(*numberOptions)

type stringOptions struct {
	minLength int
	maxLength int
	rex       *regexp.Regexp
}

func MaxLength(n int) StringOption {
	return func(o *stringOptions) {
		o.maxLength = n
	}
}

type numberOptions struct {
	mustBeInteger    bool
	mayBeInteger     bool
	minimum          *big.Float
	exclusiveMinimum bool
	maximum          *big.Float
	exclusiveMaximum bool
}
