package repo

import (
	"fmt"
	"slices"
	"text/template"
	"text/template/parse"
)

// instrumentTemplate rewrites the template by inserting function calls at every
// line or action.
//
// This is used to inject debug traces or test coverage counters (following a similar technique as coverage capture
// by go testing).
func (r *Repository) instrumentTemplate(tpl *template.Template) *template.Template {
	for _, sub := range tpl.Templates() {
		newTree := instrumentParseTree(sub.Tree)
		var err error
		tpl, err = tpl.AddParseTree(sub.Name(), newTree)
		if err != nil {
			panic(fmt.Errorf("internal error: could not add parse tree: %w: %w", err, ErrTemplateRepo))
		}
	}

	return tpl
}

func instrumentParseTree(tree *parse.Tree) *parse.Tree {
	newTree := tree.Copy()
	newTree.Root.Nodes = instrumentParseNode(newTree.Root)

	return newTree
}

func instrumentParseNode(n parse.Node) []parse.Node {
	callback := demoCallback

	switch node := n.(type) {
	case nil:
		return nil
	case *parse.ListNode:
		if node == nil {
			return nil
		}
		clone := node.CopyList()
		var j int
		for _, sub := range node.Nodes {
			instrumented := instrumentParseNode(sub)
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
		return withFuncNode(node, callback)
	case *parse.BranchNode:
		if node == nil {
			return nil
		}
		clone := node.Copy().(*parse.BranchNode)
		if clone.List != nil {
			clone.List.Nodes = instrumentParseNode(node.List)
		}
		if clone.ElseList != nil {
			clone.ElseList.Nodes = instrumentParseNode(node.ElseList)
		}
		return withFuncNode(clone, callback)
	case *parse.IfNode:
		if node == nil {
			return nil
		}
		clone := node.Copy().(*parse.IfNode)
		if clone.List != nil {
			clone.List.Nodes = instrumentParseNode(node.List)
		}
		if clone.ElseList != nil {
			clone.ElseList.Nodes = instrumentParseNode(node.ElseList)
		}
		return withFuncNode(clone, callback)
	case *parse.RangeNode:
		if node == nil {
			return nil
		}
		clone := node.Copy().(*parse.RangeNode)
		if clone.List != nil {
			clone.List.Nodes = instrumentParseNode(node.List)
		}
		if clone.ElseList != nil {
			clone.ElseList.Nodes = instrumentParseNode(node.ElseList)
		}
		return withFuncNode(clone, callback)
	case *parse.WithNode:
		if node == nil {
			return nil
		}
		clone := node.Copy().(*parse.WithNode)
		if clone.List != nil {
			clone.List.Nodes = instrumentParseNode(node.List)
		}
		if clone.ElseList != nil {
			clone.ElseList.Nodes = instrumentParseNode(node.ElseList)
		}
		return withFuncNode(clone, callback)
	case *parse.BreakNode:
		return withFuncNode(node, callback)
	case *parse.ContinueNode:
		return withFuncNode(node, callback)
	case *parse.TemplateNode:
		return withFuncNode(node, callback)
	case *parse.TextNode:
		return withFuncNode(node, callback)
	default:
		// don't add anything
		return []parse.Node{n}
	}
}

// callbackFunc returns:
// * the funcion identifier
// * the arguments to pass as parse.Node
type callbackFunc func(parse.Node) []parse.Node

func demoCallback(n parse.Node) []parse.Node {
	return []parse.Node{
		&parse.TextNode{
			NodeType: parse.NodeText,
			Text:     fmt.Appendf(make([]byte, 0, 20), "\n// DEBUG [node: %d]", n.Type()), //nolint:mnd
		},
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
								Ident:    "printf",
							},
							&parse.StringNode{
								NodeType: parse.NodeString,
								Quoted:   `"%v\n"`,
								Text:     "%v\n",
							},
							&parse.DotNode{
								NodeType: parse.NodeDot,
							},
						},
					},
				},
			},
		},
	}
}

func withFuncNode(n parse.Node, callback callbackFunc) []parse.Node {
	if n == nil {
		return nil
	}
	// add a trace before node n:
	// // Instrument comment {{ humanize . }}
	instrumented := callback(n)
	instrumented = append(instrumented, n)

	return instrumented
}

// TODO: convert Pos into line
