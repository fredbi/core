package writer

const (
	comma                = ','
	colon                = ':'
	closingBracket       = '}'
	closingSquareBracket = ']'
	openingBracket       = '{'
	openingSquareBracket = '['
	quote                = '"'
	newline              = '\n'
	space                = ' '
)

var (
	trueBytes  = []byte("true")
	falseBytes = []byte("false")
	nullToken  = []byte("null")
)
