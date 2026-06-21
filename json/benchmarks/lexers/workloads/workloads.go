// Package workloads generates deterministic JSON payloads used to stress the
// lexer benchmarks along different axes (numbers, strings, escapes, nesting,
// keys, whitespace, ...).
//
// Each workload is generated to be roughly the same size in bytes, so that the
// per-byte throughput (MB/s) reported by the benchmarks is comparable across
// workload shapes, not just across implementations.
package workloads

import (
	"fmt"
	"strconv"
	"strings"
)

// targetBytes is the approximate size each generated workload aims for.
const targetBytes = 256 * 1024

// Workload is a named JSON payload.
type Workload struct {
	Name string
	Data []byte
}

// All returns the full set of workloads, each ~256KiB of valid JSON.
func All() []Workload {
	return []Workload{
		{Name: "ints", Data: arrayOf(intElem)},
		{Name: "floats", Data: arrayOf(floatElem)},
		{Name: "strings_plain", Data: arrayOf(plainStringElem)},
		{Name: "strings_escaped", Data: arrayOf(escapedStringElem)},
		{Name: "strings_unicode", Data: arrayOf(unicodeStringElem)},
		{Name: "bools_nulls", Data: arrayOf(boolNullElem)},
		{Name: "object_keys", Data: objectKeys()},
		{Name: "nested_arrays", Data: nested("[", "]")},
		{Name: "nested_objects", Data: nestedObjects()},
		{Name: "whitespace_heavy", Data: whitespaceHeavy()},
		{Name: "mixed", Data: mixed()},
	}
}

// arrayOf builds a JSON array "[e0,e1,...]" by appending elements until the
// target size is reached. elem renders element i.
func arrayOf(elem func(i int) string) []byte {
	var b strings.Builder
	b.Grow(targetBytes + 64)
	b.WriteByte('[')

	for i := 0; b.Len() < targetBytes; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(elem(i))
	}

	b.WriteByte(']')

	return []byte(b.String())
}

func intElem(i int) string {
	// vary sign and magnitude
	n := int64(i*2654435761) % 1_000_000_007
	return strconv.FormatInt(n, 10)
}

func floatElem(i int) string {
	// mantissa + fraction + exponent, kept as text (no rounding intent)
	return fmt.Sprintf("%d.%04de%d", i%1000, (i*7)%10000, (i%18)-9)
}

func plainStringElem(i int) string {
	return fmt.Sprintf("%q", fmt.Sprintf("item-%08d-value", i))
}

func escapedStringElem(i int) string {
	// embed escapes the lexer must process: quote, backslash, control shorthands
	return fmt.Sprintf(`"line\t%d\ncol\\\"%d\"end"`, i, i)
}

func unicodeStringElem(i int) string {
	// \u escapes, including a surrogate pair (U+1D11E) every few elements
	if i%4 == 0 {
		return fmt.Sprintf(`"snow☃ clef𝄞 n%04d"`, i%10000)
	}
	return fmt.Sprintf(`"accentéèê x%04d"`, i%10000)
}

func boolNullElem(i int) string {
	switch i % 3 {
	case 0:
		return "true"
	case 1:
		return "false"
	default:
		return "null"
	}
}

func objectKeys() []byte {
	var b strings.Builder
	b.Grow(targetBytes + 64)
	b.WriteByte('{')

	for i := 0; b.Len() < targetBytes; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `"field_%08d":%d`, i, i)
	}

	b.WriteByte('}')

	return []byte(b.String())
}

// nested builds an array nested to a depth that reaches the target size, then
// closes it. open/closing are the matching delimiters.
func nested(open, closing string) []byte {
	depth := targetBytes / 2

	var b strings.Builder
	b.Grow(2*depth + 8)
	b.WriteString(strings.Repeat(open, depth))
	b.WriteString(strings.Repeat(closing, depth))

	return []byte(b.String())
}

func nestedObjects() []byte {
	// {"k":{"k":{...1...}}} nested deeply
	const wrap = `{"k":`
	depth := targetBytes / (len(wrap) + 1)

	var b strings.Builder
	b.Grow(targetBytes + 16)
	b.WriteString(strings.Repeat(wrap, depth))
	b.WriteByte('1')
	b.WriteString(strings.Repeat("}", depth))

	return []byte(b.String())
}

func whitespaceHeavy() []byte {
	// a small object pretty-printed with deep, repeated indentation
	const indent = "\n\t\t\t\t\t\t\t\t"

	var b strings.Builder
	b.Grow(targetBytes + 64)
	b.WriteByte('[')

	for i := 0; b.Len() < targetBytes; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(indent)
		fmt.Fprintf(&b, "%d", i%100)
	}

	b.WriteString("\n")
	b.WriteByte(']')

	return []byte(b.String())
}

func mixed() []byte {
	// array of small heterogeneous objects, the closest to real-world payloads
	var b strings.Builder
	b.Grow(targetBytes + 128)
	b.WriteByte('[')

	for i := 0; b.Len() < targetBytes; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b,
			`{"id":%d,"name":"user_%06d","active":%t,"score":%d.%02d,"tags":["a","b"],"note":null}`,
			i, i, i%2 == 0, i%1000, i%100,
		)
	}

	b.WriteByte(']')

	return []byte(b.String())
}
