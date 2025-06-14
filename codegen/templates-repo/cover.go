package repo

import (
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"math/rand/v2"
	"os"
	"slices"
	"strconv"
	"sync"
	"text/template/parse"
)

// FlushCoverProfile flushes captured coverage records to a profile.
//
// The format of the ouput file is the same as with go test -coverprofile.
//
// If no coverage has been captured, this function returns nil.
//
// Once flushed, coverage records are freed, so this function should only be
// called once, when you are done with executing templates.
//
// TODO: this almost works
// What is missing a this stage:
// * token column information
// * pseudo-package for gotmpl files
// * css color tweaking for clean rendering using go tool cover (how about stuff like codecov??)
func (r *Repository) FlushCoverProfile(coverProfile string) error {
	if len(r.coverageHandlers) == 0 {
		return nil
	}

	w, errFile := os.Create(coverProfile)
	if errFile != nil {
		return errors.Join(errFile, ErrTemplateRepo)
	}
	fmt.Fprintln(w, "mode: set")

	for _, h := range r.coverageHandlers {
		if err := h.FlushProfile(w); err != nil {
			return errors.Join(err, ErrTemplateRepo)
		}
	}

	return nil
}

// coverageHandler knows how to provide callbacks to an [intrumenter] and keep a record of coverage calls.
type coverageHandler struct {
	filename        string      // the template file
	coverageRecords map[int]int // map[{line number}]{number of times called}
	mx              sync.Mutex
}

func newCoverageHandler(filename string) *coverageHandler {
	return &coverageHandler{
		filename:        filename,
		coverageRecords: make(map[int]int),
	}
}

// coverCallbackBuilder produces a template instrumentation that calls back each template node at execution time
// and records a coverage profile.
//
// Notice that the execution callback has the following desirable properties:
//   - its name is random and unlikely to conflict with an existing funcmap symbol
//   - it may be called concurrently: the coverage profile is safe for concurrent write access
func (h *coverageHandler) CoverCallbackBuilder(posConverter posConverterFunc) (string, callbackFunc, funcMapFunc) {
	const suffix_length = 14
	callbackName := "cover_" + randomSuffix(suffix_length) // randomly pick a name for the callback to be added to the func map

	parseCallBack := func(n parse.Node) []parse.Node {
		line := posConverter(int(n.Position())) // TODO: should render line, startColumn, endColumn (startColumn is pos - startPos of the line, but endColumn is more complicated than that)

		log.Printf("DEBUG[%s]: added node callback: {{ %s %d }}", h.filename, callbackName, line)
		return []parse.Node{
			&parse.ActionNode{
				NodeType: parse.NodeAction,
				Pipe: &parse.PipeNode{
					NodeType: parse.NodePipe,
					Cmds: []*parse.CommandNode{
						{
							NodeType: parse.NodeCommand,
							Args: []parse.Node{
								&parse.IdentifierNode{
									NodeType: parse.NodeIdentifier,
									Ident:    callbackName,
								},
								&parse.NumberNode{
									NodeType: parse.NodeNumber,
									IsInt:    true,
									Int64:    int64(line),
									Text:     strconv.Itoa(line),
								},
								// TODO: should add start and end columns
								/*
																		&parse.NumberNode{
																			NodeType: parse.NodeNumber,
																			IsInt:    true,
																			Int64:    int64(startColumn),
																			Text:     strconv.Itoa(startColumn),
																		},
									// DO WE WANT END LINE TOO ??
																		&parse.NumberNode{
																			NodeType: parse.NodeNumber,
																			IsInt:    true,
																			Int64:    int64(endColumn),
																			Text:     strconv.Itoa(startColumn),
																		},
								*/
							},
						},
					},
				},
			},
		}
	}

	return callbackName, parseCallBack, h.ExecCallback
}

// ExecCallback is the callback function injected into the template.
//
// TODO: should be (line, startColumn, endColumn int) string
func (h *coverageHandler) ExecCallback(line int) string {
	log.Printf("DEBUG: %s called line %d", h.filename, line)
	h.increment(line)
	return ""
}

func (h *coverageHandler) FlushProfile(w io.Writer) error {
	const (
		startColumn = 0 // TODO: ?
		endColumn   = 0
	)
	for _, line := range slices.Sorted(maps.Keys(h.coverageRecords)) {
		_, err := fmt.Fprintf(w, "%s:%d.%d,%d.%d %d %d\n",
			h.filename, // TODO: should add packag name as prefix
			line,
			startColumn,
			line,
			endColumn,
			1,                       // # statements
			h.coverageRecords[line], // count
		)
		if err != nil {
			return err
		}
	}

	// free records
	h.coverageRecords = nil

	return nil
}

func (h *coverageHandler) increment(line int) {
	h.mx.Lock()
	v := h.coverageRecords[line]
	v++
	h.coverageRecords[line] = v
	h.mx.Unlock()
}

// randomSuffix is used to produce unique exec callback names that don't conflict with existing funcmaps.
//
// length must be =< 36
func randomSuffix(length int) string {
	chars := []byte("abcdefghijklmnoprstuvwxyz0123456789")

	rand.Shuffle(len(chars), func(i, j int) {
		chars[i], chars[j] = chars[j], chars[i]
	})

	return string(chars[:length])
}

/*
older stuff

		return []parse.Node{
			*
				&parse.TextNode{
					NodeType: parse.NodeText,
					Text:     fmt.Appendf(make([]byte, 0, 20), "\n// DEBUG [node: %d]", n.Type()), //nolint:mnd
				},
			*
			&parse.ActionNode{
				NodeType: parse.NodeAction,
				Pipe: &parse.PipeNode{
					NodeType: parse.NodePipe,
					Cmds: []*parse.CommandNode{
						{
							NodeType: parse.NodeCommand,
							Args: []parse.Node{
								// TODO: call a function that populates a coverage counter
								&parse.IdentifierNode{
									NodeType: parse.NodeIdentifier,
									Ident:    "callback",
								},
								&parse.NumberNode{
									NodeType: parse.NodeNumber,
									IsInt:    true,
									Int64:    int64(line),
									Text:     strconv.Itoa(line),
								},
								*
									&parse.StringNode{
										NodeType: parse.NodeString,
										Quoted:   `"%v\n"`,
										Text:     "%v\n",
									},
									&parse.DotNode{
										NodeType: parse.NodeDot,
									},
								*
							},
						},
					},
				},
			},
		}
	}
*/
