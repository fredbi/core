package main

// reexport all exported functions, variables from package "generated" */

import (
	"github.com/fredbi/core/codegen/gentesting/fixtures/generated"
)

var (
	// all exported constants
	AConstant = generated.AConstant
	EnumOne   = generated.EnumOne
	EnumTwo   = generated.EnumTwo
)

var (
	// all exported variables
	EnumComplex = generated.EnumComplex
	EnumValues  = generated.EnumValues
)

var (
	// all exported types
	Model             generated.Model
	IntegerCollection generated.IntegerCollection
)

var (
	// all exported functions
	NewModel              = generated.NewModel
	MakeIntegerCollection = generated.MakeIntegerCollection
)
