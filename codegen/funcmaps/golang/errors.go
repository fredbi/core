package golang

// Error is a sentinel error type for all errors raised by this package.
type Error string

func (e Error) Error() string {
	return string(e)
}

// ErrFuncMap is a sentinel error that wraps all errors raised by this package.
const ErrFuncMap Error = "golang funcmap error"

// ErrTemplateAssertion is an error reported by the "assert" function in templates.
const ErrTemplateAssertion Error = "assertion failed in template"
