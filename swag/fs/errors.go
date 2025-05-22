package fs

type Error string

func (e Error) Error() string {
	return string(e)
}

const ErrUnsupportedFSFeature Error = "unsupported fs feature"
