package repo

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"reflect"
	"slices"
	"text/template"
	"text/template/parse"
)

type (
	// instrumenter knows how to inject nodes produced by a callback function before each template statement.
	instrumenter struct {
		execCallbackName string
		parseCallback    callbackFunc
		execCallback     funcMapFunc
	}

	// funcMapFunc is a function that may be called by a template.
	//
	// Notice that the [template.FuncMap] type knows how to assert "good" functions.
	funcMapFunc any

	// callbackBuilder takes a posConverterFunc function and returns:
	// * a callback name to be added to the template's funcmap (if not empty)
	// * a callback function to be called when parsing nodes
	callbackBuilder func(posConverterFunc) (name string, parse callbackFunc, exec funcMapFunc)

	// callbackFunc is a callback function to be called when parsing nodes, which prepends instrumentation nodes
	// to all (relevant) nodes.
	callbackFunc func(original parse.Node) (instrumented []parse.Node)

	// posConverterFunc is a function that converts a token offset position to a line number in the template source code
	posConverterFunc func(pos int) (line int)
)

// newInstrumenter builds a new template instrumenter from template source and a callback builder.
func newInstrumenter(source []byte, builder callbackBuilder) *instrumenter {
	lineConverter := positionToLineConverter(source)                       // establish a correspondance between token position and line number in source
	execCallbackName, callbackFunc, execCallBack := builder(lineConverter) // construct the callback equiped with this correspondance to locate its calling position.

	return &instrumenter{
		execCallbackName: execCallbackName,
		parseCallback:    callbackFunc,
		execCallback:     execCallBack,
	}
}

// instrumentTemplate rewrites the template by inserting function calls at every
// line or action.
//
// This is used to inject debug traces or test coverage counters, following a technique similar to how go testing
// captures test coverage.
func (i *instrumenter) InstrumentTemplate(tpl *template.Template) *template.Template {
	for _, sub := range tpl.Templates() {
		newTree := i.InstrumentParseTree(sub.Tree)
		var err error
		tpl, err = tpl.AddParseTree(sub.Name(), newTree)
		if err != nil {
			panic(fmt.Errorf("internal error: could not add parse tree: %w: %w", err, ErrTemplateRepo))
		}
	}

	if i.execCallbackName != "" && i.execCallback != nil {
		// the configured callback requires a new function to be added to the template's funcmap.
		return tpl.Funcs(template.FuncMap{
			i.execCallbackName: i.execCallback,
		})
	}

	return tpl
}

func (i *instrumenter) InstrumentParseTree(tree *parse.Tree) *parse.Tree {
	newTree := tree.Copy()
	newTree.Root.Nodes = i.InstrumentParseNode(newTree.Root)

	return newTree
}

func (i *instrumenter) InstrumentParseNode(n parse.Node) []parse.Node {
	if isNil(n) {
		return nil
	}

	switch node := n.(type) {
	case *parse.ListNode:
		if node == nil {
			return nil
		}
		clone := node.CopyList()
		var j int
		for _, sub := range node.Nodes {
			instrumented := i.InstrumentParseNode(sub)
			if len(instrumented) == 0 {
				j++
				continue
			}
			clone.Nodes = slices.Delete(clone.Nodes, j, j+1)
			clone.Nodes = slices.Insert(clone.Nodes, j, instrumented...)
			j += len(instrumented)
		}

		return clone.Nodes
	case *parse.ActionNode:
		return i.WithFuncNode(node)
	case *parse.BranchNode:
		if node == nil {
			return nil
		}
		clone := node.Copy().(*parse.BranchNode)
		if !isNil(node.List) {
			clone.List.Nodes = i.InstrumentParseNode(node.List)
		}
		if !isNil(node.ElseList) {
			clone.ElseList.Nodes = i.InstrumentParseNode(node.ElseList)
		}

		return i.WithFuncNode(clone)
	case *parse.IfNode:
		if node == nil {
			return nil
		}
		clone := node.Copy().(*parse.IfNode)
		if !isNil(node.List) {
			clone.List.Nodes = i.InstrumentParseNode(node.List)
		}
		if !isNil(node.ElseList) {
			clone.ElseList.Nodes = i.InstrumentParseNode(node.ElseList)
		}

		return i.WithFuncNode(clone)
	case *parse.RangeNode:
		if node == nil {
			return nil
		}
		clone := node.Copy().(*parse.RangeNode)
		if !isNil(node.List) {
			clone.List.Nodes = i.InstrumentParseNode(node.List)
		}
		if !isNil(node.ElseList) {
			clone.ElseList.Nodes = i.InstrumentParseNode(node.ElseList)
		}

		return i.WithFuncNode(clone)
	case *parse.WithNode:
		if node == nil {
			return nil
		}
		clone := node.Copy().(*parse.WithNode)
		if !isNil(node.List) {
			clone.List.Nodes = i.InstrumentParseNode(node.List)
		}
		if !isNil(node.ElseList) {
			clone.ElseList.Nodes = i.InstrumentParseNode(node.ElseList)
		}

		return i.WithFuncNode(clone)
	case *parse.BreakNode:
		return i.WithFuncNode(node)
	case *parse.ContinueNode:
		return i.WithFuncNode(node)
	case *parse.TemplateNode:
		return i.WithFuncNode(node)
	case *parse.TextNode:
		return i.WithFuncNode(node)
	default:
		// don't add anything
		return []parse.Node{n}
	}
}

func (i *instrumenter) WithFuncNode(n parse.Node) []parse.Node {
	if n == nil {
		return nil
	}
	// add a trace before node n:
	instrumented := i.parseCallback(n)
	instrumented = append(instrumented, n)

	log.Printf("instrumented list = %v", instrumented)
	return instrumented
}

// isNil is used when analyzing the parsing tree.
//
// Due to the heavy use of interfaces, we hit one of golang's darkest corners: "interface with a nil value" != nil.
func isNil(v any) bool {
	return v == nil || reflect.ValueOf(v).IsNil()
}

type lineBreak struct {
	Line     int
	StartPos int
	EndPos   int
}

// positionToLineConverter returns a function that knows how to convert an offet in the source
// code buffer (position of the token) into a line number.
//
// This execution only adds overhead to the template parsing time: it not called when the template is
// executed.
//
// TODO: go profiles also want columns
func positionToLineConverter(source []byte) posConverterFunc {
	lineBreaks := make([]lineBreak, 0, 100)

	scanner := bufio.NewScanner(bytes.NewReader(source))
	var (
		currentPosition int
		currentLine     int
	)

	for scanner.Scan() {
		currentLine++
		length := len(scanner.Bytes()) + 1 // add CR byte
		previousPosition := currentPosition
		currentPosition += length

		lineBreaks = append(lineBreaks, lineBreak{
			StartPos: previousPosition,
			EndPos:   currentPosition,
			Line:     currentLine,
		})
	}

	return func(pos int) (line int) {
		n, _ := slices.BinarySearchFunc(lineBreaks, pos, func(lb lineBreak, p int) int {
			switch {
			case p < lb.EndPos:
				return 1
			case p > lb.EndPos:
				return -1
			default:
				return 0
			}
		})

		if n >= len(lineBreaks) {
			return lineBreaks[len(lineBreaks)-1].Line
		}

		return lineBreaks[n].Line
	}
}
