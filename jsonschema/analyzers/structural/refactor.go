package structural

type RefactoringAction uint8

const (
	RefactoringActionNone = iota
	RefactoringActionSplitNamed
)

type refactoringInfo struct {
	action RefactoringAction
	extra  any
}

func (r refactoringInfo) Action() RefactoringAction {
	return r.action
}
