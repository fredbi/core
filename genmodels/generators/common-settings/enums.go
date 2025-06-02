package settings

import types "github.com/fredbi/core/genmodels/generators/extra-types"

var (
	_ types.Setting = ConstructorOptionSelector(0)
	_ types.Setting = PackageLayoutSelector(0)
	_ types.Setting = PackageLayoutOptionSelector(0)
	_ types.Setting = ModelLayoutSelector(0)
	_ types.Setting = ValidationLayoutSelector(0)

	_ types.Setting = ConstructorMode(1)
	_ types.Setting = ExtraMethodsMode(1)
)

// ConstructorMode describe the methods to add to models as constructors.
type ConstructorMode uint8

const (
	ConstructorModeNone ConstructorMode = 1 << iota // no constructor
	ConstructorModeMake                             // constructor that returns a value
	ConstructorModeNew                              // constructor that returns a pointer
)

func (m ConstructorMode) String() string {
	switch m {
	case ConstructorModeNone:
		return "no-constructor"
	case ConstructorModeMake:
		return "make-constructor"
	case ConstructorModeNew:
		return "new-constructor"
	default:
		assertValue(m)
		return ""
	}
}

func (m ConstructorMode) FromString(s string) (ConstructorMode, error) {
	switch s {
	case "no-constructor":
		return ConstructorModeNone, nil
	case "make-constructor":
		return ConstructorModeMake, nil
	case "new-constructor":
		return ConstructorModeNew, nil
	default:
		return ConstructorModeNone, invalidMode(m, s)
	}
}

func (m ConstructorMode) DocString() string {
	switch m {
	case ConstructorModeNone:
		return "do not add any constructor for models"
	case ConstructorModeMake:
		return "add a Make constructor to defined models, that returns a value"
	case ConstructorModeNew:
		return "add a New constructor to defined models, that returns a pointer"
	default:
		assertValue(m)
		return ""
	}
}

func (m ConstructorMode) Has(modes ...ConstructorMode) bool {
	return types.HasMode(m, modes...)
}

func (m ConstructorMode) HasString(shorts ...string) bool {
	return types.HasString(m, m.FromString, shorts...)
}

type ConstructorOptionSelector uint8

const (
	ConstructorAllTypes ConstructorOptionSelector = iota
	ConstructorOnlyWithDefault
)

func (m ConstructorOptionSelector) String() string {
	switch m {
	case ConstructorAllTypes:
		return "all-types"
	case ConstructorOnlyWithDefault:
		return "only-with-default"
	default:
		assertValue(m)
		return ""
	}
}

func (m ConstructorOptionSelector) DocString() string {
	switch m {
	case ConstructorAllTypes:
		return "generates constructors for all type definitions"
	case ConstructorOnlyWithDefault:
		return "generates constructors only for types with a default value"
	default:
		assertValue(m)
		return ""
	}
}

func (m ConstructorOptionSelector) Is(selected ...ConstructorOptionSelector) bool {
	return types.IsSelector(m, selected...)
}

// PackageLayoutSelector describes the method to construct a layout for the packages that define models.
type PackageLayoutSelector uint8

const (
	PackageLayoutFlat PackageLayoutSelector = iota
	PackageLayoutHierarchical
)

func (m PackageLayoutSelector) String() string {
	switch m {
	case PackageLayoutFlat:
		return "flat"
	case PackageLayoutHierarchical:
		return "hierarchical"
	default:
		assertValue(m)
		return ""
	}
}

func (m PackageLayoutSelector) DocString() string {
	switch m {
	case PackageLayoutFlat:
		return "flat package layout (all models in one package)"
	case PackageLayoutHierarchical:
		return "hierarchical package layout (split models in separate packages)"
	default:
		assertValue(m)
		return ""
	}
}

func (m PackageLayoutSelector) Is(selected ...PackageLayoutSelector) bool {
	return types.IsSelector(m, selected...)
}

type PackageLayoutOptionSelector uint8

const (
	PackageLayoutRefBased PackageLayoutOptionSelector = iota
	PackageLayoutTagBased
)

func (m PackageLayoutOptionSelector) String() string {
	switch m {
	case PackageLayoutRefBased:
		return "ref-based"
	case PackageLayoutTagBased:
		return "tag-based"
	default:
		assertValue(m)
		return ""
	}
}

func (m PackageLayoutOptionSelector) DocString() string {
	switch m {
	case PackageLayoutRefBased:
		return "hierarchical model layout (sub-packages organized as $ref paths)"
	case PackageLayoutTagBased:
		return "hierarchical model layout (sub-packages organized according to x-go-tag metadata)"
	default:
		assertValue(m)
		return ""
	}
}

func (m PackageLayoutOptionSelector) Is(selected ...PackageLayoutOptionSelector) bool {
	return types.IsSelector(m, selected...)
}

type PackageLayoutOptionMode uint8

const (
	PackageLayoutEager PackageLayoutOptionMode = 1 << iota
	PackageLayoutLazy                          // TODO: should be selector
	PackageLayoutEnums                         // should be mode
)

func (m PackageLayoutOptionMode) String() string {
	switch m {
	case PackageLayoutEager:
		return "eager"
	case PackageLayoutLazy:
		return "lazy"
	case PackageLayoutEnums:
		return "enums"
	default:
		assertValue(m)
		return ""
	}
}

func (m PackageLayoutOptionMode) DocString() string {
	switch m {
	case PackageLayoutEager:
		return "hierarchical model layout with as many packages as we may find"
	case PackageLayoutLazy:
		return "hierarchical model layout with only packages on clusters of schemas that produce name conflicts"
	case PackageLayoutEnums:
		return "enum types and constants are defined in a separate sub-package"
	default:
		assertValue(m)
		return ""
	}
}

func (m PackageLayoutOptionMode) Has(modes ...PackageLayoutOptionMode) bool {
	return types.HasMode(m, modes...)
}

type ModelLayoutSelector uint8

const (
	ModelLayoutRelatedModelsOneFile ModelLayoutSelector = iota // default
	ModelLayoutOneModelPerFile
	ModelLayoutAllModelsOneFile
)

func (m ModelLayoutSelector) String() string {
	switch m {
	case ModelLayoutOneModelPerFile:
		return "one-per-file"
	case ModelLayoutAllModelsOneFile:
		return "single-file"
	case ModelLayoutRelatedModelsOneFile:
		return "related-same-file"
	default:
		assertValue(m)
		return ""
	}
}

func (m ModelLayoutSelector) DocString() string {
	switch m {
	case ModelLayoutOneModelPerFile:
		return "generates one type per source file"
	case ModelLayoutAllModelsOneFile:
		return "generates all models in a single source file"
	case ModelLayoutRelatedModelsOneFile:
		return "generates all related models in a single source file"
	default:
		assertValue(m)
		return ""
	}
}

func (m ModelLayoutSelector) Is(selected ...ModelLayoutSelector) bool {
	return types.IsSelector(m, selected...)
}

// ValidationLayoutSelector describes whether the validation code (which may be large) is issued in the
// same file as the type definition or in a separate source file.
type ValidationLayoutSelector uint8

const (
	ValidationLayoutNone ValidationLayoutSelector = iota
	ValidationLayoutSameFile
	ValidationLayoutSeparateFile
)

func (m ValidationLayoutSelector) String() string {
	switch m {
	case ValidationLayoutNone:
		return "no-validations"
	case ValidationLayoutSameFile:
		return "same-file"
	case ValidationLayoutSeparateFile:
		return "separate-file"
	default:
		assertValue(m)
		return ""
	}
}

func (m ValidationLayoutSelector) DocString() string {
	switch m {
	case ValidationLayoutNone:
		return "no validation code is generated"
	case ValidationLayoutSameFile:
		return "validations are generated alongside type definition"
	case ValidationLayoutSeparateFile:
		return "validations are generated in a separate source file"
	default:
		assertValue(m)
		return ""
	}
}

func (m ValidationLayoutSelector) Is(selected ...ValidationLayoutSelector) bool {
	return types.IsSelector(m, selected...)
}

type ExtraMethodsMode uint8

const (
	ExtraMethodsString   ExtraMethodsMode = 1 << iota // generate a string representation method (e.g. String())
	ExtraMethodsDeepCopy                              // generates a deep copy method
)

func (m ExtraMethodsMode) String() string {
	switch m {
	case ExtraMethodsString:
		return "stringer"
	case ExtraMethodsDeepCopy:
		return "deep-copier"
	default:
		assertValue(m)
		return ""
	}
}

func (m ExtraMethodsMode) DocString() string {
	switch m {
	case ExtraMethodsString:
		return "generates a string representation method"
	case ExtraMethodsDeepCopy:
		return "generates a deep-copy method"
	default:
		assertValue(m)
		return ""
	}
}

func (m ExtraMethodsMode) Has(modes ...ExtraMethodsMode) bool {
	return types.HasMode[ExtraMethodsMode](m, modes...)
}
