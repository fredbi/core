package mangling

type Inflecter struct {
	inflecterOptions
}

type NumberSpeller struct {
	numberSpellingOptions
}

type GoNameMangler struct {
	goOptions
	NameMangler
}

type inflecterOptions struct{}
type numberSpellingOptions struct{}
type goOptions struct{}
