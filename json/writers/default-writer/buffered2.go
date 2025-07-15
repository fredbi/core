package writer

import (
	"io"
	"runtime"
	"unicode/utf8"

	"github.com/fredbi/core/json/writers"
)

var (
	_ writers.StoreWriter    = &Buffered2{}
	_ writers.JSONWriter     = &Buffered2{}
	_ writers.TokenWriter    = &Buffered2{}
	_ writers.VerbatimWriter = &Buffered2{}
)

// Buffered2 JSON writer.
type Buffered2 struct {
	buffered2
	commonWriter[*buffered2]
}

type buffered2 struct {
	baseWriter
	*bufferedOptions
}

func NewBuffered2(w io.Writer, opts ...BufferedOption) *Buffered2 {
	writer := &Buffered2{
		buffered2: buffered2{
			baseWriter: baseWriter{
				w: w,
			},
			bufferedOptions: bufferedOptionsWithDefaults(opts), // always borrow options from the pool
		},
	}
	writer.commonWriter.jw = &writer.buffered2

	// when using New, borrowed inner resources must be relinquished when the gc claims the writer.
	runtime.AddCleanup(writer, func(o *bufferedOptions) {
		if o != nil {
			o.redeem()

			poolOfBufferedOptions.Redeem(o)
		}
	}, writer.bufferedOptions)

	return writer
}

func (w *Buffered2) Reset() {
	w.baseWriter.Reset()
	if w.bufferedOptions != nil {
		w.bufferedOptions.Reset()
	}
}

// Flush the internal buffer of the [Buffered2] writer to the underlying [io.Writer].
func (w *Buffered2) Flush() error {
	if w.err != nil {
		return w.err
	}

	w.flush()

	return w.err
}

func (w *buffered2) flush() {
	n, err := w.w.Write(w.buffer)
	w.inc(n)
	w.err = err
	w.buffer = w.buffer[:0]
}

// redeem inner resources
func (w *buffered2) redeem() {
	if w.bufferedOptions != nil {
		w.bufferedOptions.redeem()

		poolOfBufferedOptions.Redeem(w.bufferedOptions)
		w.bufferedOptions = nil
	}
}

func (w *buffered2) writeSingleByte(c byte) {
	if len(w.buffer) == cap(w.buffer) {
		w.flush()

		if w.err != nil {
			return
		}
	}

	w.buffer = append(w.buffer, c)
}

func (w *buffered2) writeBinary(data []byte) {
	var offset int

	for offset < len(data) {
		if len(w.buffer) == cap(w.buffer) {
			w.flush()
			if w.err != nil {
				return
			}
		}

		chunkSize := min(len(data[offset:]), cap(w.buffer)-len(w.buffer))
		w.buffer = append(w.buffer, data[offset:offset+chunkSize]...) // copy data to the buffer

		offset += chunkSize
	}
}

// TODO: rewrite common StringCopy => requires a "needMore" flag in writeText
func (w *buffered2) writeText(data []byte) (remainder []byte) {
	w.writeSingleByte(quote)
	if w.err != nil {
		return
	}

	var (
		p       int
		escaped bool
	)

	// first iterates over non-escaped bytes.
	for ; p < len(data); p++ {
		c := data[p]
		if c < lowestPrintable || c >= utf8.RuneSelf || c == '\t' || c == '\r' || c == '\n' || c == '\\' || c == '"' || c == '\b' || c == '\f' {
			escaped = true

			break
		}
	}

	if p > 0 {
		w.writeBinary(data[:p])
	}

	if !escaped {
		//  nothing to be escaped: we are done
		w.writeSingleByte(quote)

		return nil
	}

	for i := p; i < len(data); i++ {
		const (
			escapedSize        = 2
			escapedUnicodeSize = 6
		)

		c := data[i]
		available := cap(w.buffer) - len(w.buffer)

		switch {
		// TODO: compare with table lookup
		case c == '\t':
			if available < escapedSize {
				w.flush()
				if w.err != nil {
					return nil
				}
			}
			w.buffer = append(w.buffer, '\\', 't')
		case c == '\r':
			if available < escapedSize {
				w.flush()
				if w.err != nil {
					return nil
				}
			}
			w.buffer = append(w.buffer, '\\', 'r')
		case c == '\n':
			if available < escapedSize {
				w.flush()
				if w.err != nil {
					return nil
				}
			}
			w.buffer = append(w.buffer, '\\', 'n')
		case c == '\\':
			if available < escapedSize {
				w.flush()
				if w.err != nil {
					return nil
				}
			}
			w.buffer = append(w.buffer, '\\', '\\')
		case c == '"':
			if available < escapedSize {
				w.flush()
				if w.err != nil {
					return nil
				}
			}
			w.buffer = append(w.buffer, '\\', '"')
		case c == '\b':
			if available < escapedSize {
				w.flush()
				if w.err != nil {
					return nil
				}
			}
			w.buffer = append(w.buffer, '\\', 'b')
		case c == '\f':
			if available < escapedSize {
				w.flush()
				if w.err != nil {
					return nil
				}
			}
			w.buffer = append(w.buffer, '\\', 'f')
		case c >= 0x20 && c < utf8.RuneSelf:
			// single-width character, no escaping is required
			if available == 0 {
				w.flush()
				if w.err != nil {
					return nil
				}
			}
			w.buffer = append(w.buffer, c)
		case c < lowestPrintable:
			// control character is escaped as the unicode sequence \u00{hex representation of c}
			const chars = "0123456789abcdef"
			if available < escapedUnicodeSize {
				w.flush()
				if w.err != nil {
					return nil
				}
			}
			w.buffer = append(w.buffer, '\\', 'u', '0', '0', chars[c>>4], chars[c&0xf]) // hexadecimal representation of c
		default:
			// multi-byte UTF8 character.
			if !utf8.FullRune(data[i:]) {
				// needs more read to complete the current rune
				return data[i:]
			}

			r, runeWidth := utf8.DecodeRune(data[i:])
			if available < runeWidth {
				w.flush()
				if w.err != nil {
					return nil
				}
			}
			w.buffer = utf8.AppendRune(w.buffer, r) // invalid runes are represented as \uFFFD
			i += runeWidth - 1
		}
	}

	w.writeSingleByte(quote)

	return nil
}
