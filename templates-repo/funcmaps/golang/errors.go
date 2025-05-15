package golang

type Error string

func (e Error) Error() string {
	return string(e)
}

const ErrGolangFuncMap Error = "golang funcmap error"
