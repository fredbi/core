package lexer

import "github.com/fredbi/core/json/lexers/token"

// NextToken returns the next JSON token consumed from the stream or slice of bytes.
//
// The last token is of Kind EOF.
//
// If the lexer is in an errored status, it will keep responding tokens with Kind Unknown.
//
// By default the structural separators "," and ":" are validated but not emitted
// (see [WithElideSeparator]); pass WithElideSeparator(false) to receive them.
//
// Tokens are expected to have a short lifespan: when [L.NextToken] is called again,
// the memory allocated to support the value of the previously returned token is reused for the next token.
//
// If you want to keep a token for later reuse, you should clone it using its [T.Clone] method.
func (l *L) NextToken() token.T {
	// devirtualized pull core.
	//
	// Dispatch once per token on wholeBuffer: the whole-buffer lane (local cursor, no readMore, zero-copy blanks)
	// is our fastest core.
	//
	// The stream lane is optimized separately.
	// The generic scanTokenBufferG / scanTokenStreamG are lexgen's source-of-truth.
	if l.in.WholeBuffer {
		return scanTokenBufferSemantic(l, semanticPolicy{})
	}

	if l.in.NeedFirstFill {
		l.in.FirstFill()
		if l.in.WholeBuffer {
			return scanTokenBufferSemantic(l, semanticPolicy{})
		}
	}

	return scanTokenStreamSemantic(l, semanticPolicy{})
}

// NextToken returns the next token consumed from the stream or slice of bytes.
//
// The returned [token.T] with string/number values kept RAW (escapes intact — decode on demand with [token.Unescape]).
//
// The last token is of Kind EOF; in an errored state it keeps returning tokens of Kind Unknown.
//
// The verbatim feature — the whitespace run preceding the token and its line-based source position —
// is exposed as LEXER STATE, valid until the next call, via [VL.LeadingSpace] / [VL.Line] / [VL.Column].
//
// Tokens are expected to have a short lifespan: when NextToken is called again,
// the memory backing the previous token's value (and the accessors' state) is
// reused. To keep a token, use its Clone() method.
//
// NOTE: the "token-vs-state arbitrage" keeps the emitted token [token.T] small at 32B (like the semantic lexer [L]),
// instead of using a heavier "verbatim" token.
func (l *VL) NextToken() token.T {
	// devirtualized pull core; see [L.NextToken].
	// Same wholeBuffer lane dispatch:
	// the buffer lane gives zero-copy blanks, the stream lane keeps the byte-by-byte blanks append across refills.
	if l.in.WholeBuffer {
		return scanTokenBufferVerbatim(l.L, verbatimPolicy{})
	}

	if l.in.NeedFirstFill {
		l.in.FirstFill() // §10.5f: promote to whole-buffer if the input fits
		if l.in.WholeBuffer {
			return scanTokenBufferVerbatim(l.L, verbatimPolicy{})
		}
	}

	return scanTokenStreamVerbatim(l.L, verbatimPolicy{})
}
