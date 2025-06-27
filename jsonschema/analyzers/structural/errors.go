package structural

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrSchemaBuilder = "error in building an analyzed schema"
)
