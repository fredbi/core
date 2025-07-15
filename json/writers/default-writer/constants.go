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
	lowestPrintable      = byte(0x20)
)

var (
	trueBytes  = []byte("true")
	falseBytes = []byte("false")
	nullToken  = []byte("null")
)
