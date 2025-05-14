package ast

type Tree struct {
	root Node
}

type Forest struct {
	roots []Node
}

type Node struct {
	children []Node
}
