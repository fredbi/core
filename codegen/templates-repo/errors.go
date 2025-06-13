package repo

// Error is a sentinel error type signaling any error raised from this package.
type Error string

func (e Error) Error() string {
	return string(e)
}

// ErrTemplateRepo is a sentinel error wrapped any error raised from this package.
const ErrTemplateRepo Error = "template repo error"
