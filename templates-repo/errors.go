package repo

type Error string

func (e Error) Error() string {
	return string(e)
}

const ErrTemplateRepo Error = "template repo error"
