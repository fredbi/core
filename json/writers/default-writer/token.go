package writer

import "github.com/fredbi/core/json/lexers/token"

// Token writes a token [token.T] from a lexer.
//
// For key tokens, you'd need to call explicitly with the following colon token.
func (w *W) Token(tok token.T) {
	if w.err != nil {
		return
	}

	switch tok.Kind() {
	case token.Delimiter:
		switch tok.Delimiter() {
		case token.OpeningBracket:
			w.StartObject()
		case token.ClosingBracket:
			w.EndObject()
		case token.OpeningSquareBracket:
			w.StartArray()
		case token.ClosingSquareBracket:
			w.EndArray()
		case token.Comma:
			w.Comma()
		case token.Colon:
			w.buffer.WriteSingleByte(':')
		default:
			// ignore
		}
	case token.String, token.Key:
		w.buffer.WriteSingleByte('"')
		w.buffer.WriteText(tok.Value())
		w.buffer.WriteSingleByte('"')
	case token.Number:
		w.buffer.WriteBinary(tok.Value())
	case token.Boolean:
		w.Bool(tok.Bool())
	case token.Null:
		w.Null()
	case token.EOF:
		fallthrough
	default:
		// ignore
	}
}

// VerbatimToken writes a verbatim token [token.VT] from a verbatim lexer.
//
// Non-significant white-space preceding each token is written to the buffer.
func (w *W) VerbatimToken(tok token.VT) {
	if w.err != nil {
		return
	}

	if tok.IsKey() {
		return
	}

	w.buffer.WriteBinary(tok.Blanks())
	w.Token(tok.T)
}
