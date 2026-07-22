package lexer

import "io"

// Input holds the lexer's buffered-input and scan-cursor state: the buffer window
// and read position, the value-accumulation scratch, the reader and refill buffers,
// the error state, and the two grammar flags the string scanner consults to finish a
// value (expectKey/afterKey). It is the cohesive, L-independent state the value
// scanners operate on; [L] embeds it as `in`. (Staging toward an internal package —
// the scanners will move onto *Input; the cores read its fields via l.in.*)
type Input struct {
	r   io.Reader
	err error

	buffer         []byte // determined by bufferSize
	currentValue   []byte // capped if maxValueBytes > 0
	previousBuffer []byte // used when keepPreviousBuffer > 0

	offset     uint64
	consumed   int
	bufferized int

	wholeBuffer   bool // the buffer holds the entire input (no refills): values may alias it
	needFirstFill bool // streaming: the initial read (and whole-buffer short-circuit) is pending
	expectKey     bool
	afterKey      bool // the previous token was an object key: a ':' must follow

	// trackBlanks mirrors L.trackBlanks for consumeString's dispatch to the raw
	// (validate-not-decode) string scanners. It is deliberately duplicated rather than
	// moved: the cores read L.trackBlanks on the hot whitespace-skip path, and
	// rewriting those refs to l.in.trackBlanks perturbed the core codegen (measured
	// mesh +7%). Set by VL setup alongside L.trackBlanks; L.reset() does not clear it.
	trackBlanks bool

	// noAVX2 disables the AVX2 string-stop scanner (mirrors options.noAVX2), and
	// maxValueBytes / keepPreviousBuffer mirror the like-named options so the value
	// scanners and readMore (methods on *Input) can enforce the caps/gates without
	// reaching back into L. The mirrored options are set in L.reset(). Placed last so
	// the hot cursor fields keep their offsets.
	noAVX2             bool
	maxValueBytes      int
	keepPreviousBuffer int
}
