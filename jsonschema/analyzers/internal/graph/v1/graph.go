package graph

import (
	"iter"

	"github.com/fredbi/core/swag/typeutils"
	"github.com/hmdsefi/gograph"
	"github.com/hmdsefi/gograph/traverse"
)

type ErrGraph string

func (e ErrGraph) Error() string {
	return string(e)
}

const ErrNodeNotFound ErrGraph = "node not found"

type GID interface {
	~string
}

type Edge[T GID] struct {
	From T
	To   T
}

type DiGraph[T GID] struct {
	inner gograph.Graph[T]
}

func NewDiGraph[T GID]() *DiGraph[T] {
	return &DiGraph[T]{
		inner: gograph.New[T](gograph.Directed()),
	}
}

type Tree[T GID] struct {
	DiGraph[T]
}

func NewTree[T GID]() *Tree[T] {
	return &Tree[T]{
		DiGraph: DiGraph[T]{
			inner: gograph.New[T](gograph.Directed(), gograph.Acyclic()),
		},
	}
}

func (d *Tree[T]) IsTree() bool {
	return !d.HasMultipleRoots()
}

func (d *Tree[T]) Clone() *Tree[T] {
	clone := NewTree[T]()

	for _, vertex := range d.inner.GetAllVertices() {
		clone.inner.AddVertex(vertex)
	}

	for _, edge := range d.inner.AllEdges() {
		_, _ = clone.inner.AddEdge(edge.Destination(), edge.Source())
	}

	return clone
}

func (d *DiGraph[T]) AddNode(id T) {
	d.inner.AddVertexByLabel(id)
}

func (d *DiGraph[T]) RemoveNode(id T) {
	v := d.inner.GetVertexByID(id)
	if v == nil {
		return
	}

	d.inner.RemoveVertices(v)
}

func (d *DiGraph[T]) RemoveEdge(e Edge[T]) {
	source := d.inner.GetVertexByID(e.From)
	if source == nil {
		return
	}
	dest := d.inner.GetVertexByID(e.To)
	if dest == nil {
		return
	}

	d.inner.RemoveEdges(gograph.NewEdge(source, dest))
}

func (d *DiGraph[T]) AddEdge(e Edge[T]) error {
	source := d.inner.GetVertexByID(e.From)
	if source == nil {
		return ErrNodeNotFound
	}
	dest := d.inner.GetVertexByID(e.To)
	if dest == nil {
		return ErrNodeNotFound
	}

	_, err := d.inner.AddEdge(source, dest)

	return err
}

func (d *DiGraph[T]) Clone() *DiGraph[T] {
	clone := NewDiGraph[T]()

	for _, vertex := range d.inner.GetAllVertices() {
		clone.inner.AddVertex(vertex)
	}

	for _, edge := range d.inner.AllEdges() {
		clone.inner.AddEdge(edge.Destination(), edge.Source())
	}

	return clone
}

func (d *DiGraph[T]) Nodes() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, vertex := range d.inner.GetAllVertices() {
			if !yield(vertex.Label()) {
				return
			}
		}
	}
}

func (d *DiGraph[T]) Edges() iter.Seq[Edge[T]] {
	return func(yield func(Edge[T]) bool) {
		for _, edge := range d.inner.AllEdges() {
			var from, to T

			source := edge.Source()
			if source != nil {
				from = source.Label()
			}
			destination := edge.Destination()
			if destination != nil {
				to = destination.Label()
			}

			e := Edge[T]{
				From: from,
				To:   to,
			}

			if !yield(e) {
				return
			}
		}
	}
}

func (d *DiGraph[T]) Revert() *DiGraph[T] {
	reverted := NewDiGraph[T]()

	for _, vertex := range d.inner.GetAllVertices() {
		reverted.inner.AddVertex(vertex)
	}

	for _, edge := range d.inner.AllEdges() {
		reverted.inner.AddEdge(edge.Destination(), edge.Source())
	}

	return reverted
}

// TODO: find cycles?

func (d *DiGraph[T]) HasCycle() bool {
	_, err := gograph.TopologySort(d.inner)

	return err != nil
}

func (d *DiGraph[T]) Leaves() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, vertex := range d.inner.GetAllVertices() {
			if vertex.OutDegree() > 0 {
				continue
			}

			if !yield(vertex.Label()) {
				return
			}
		}
	}
}

func (d *DiGraph[T]) Roots() iter.Seq[T] {
	// TODO: wrong if cycle to root, then its no longer a root.
	return func(yield func(T) bool) {
		for _, vertex := range d.inner.GetAllVertices() {
			if vertex.InDegree() > 0 {
				continue
			}

			if !yield(vertex.Label()) {
				return
			}
		}
	}
}

func (d *DiGraph[T]) HasMultipleRoots() bool {
	var roots int
	for range d.Roots() {
		roots++
		if roots > 1 {
			return true
		}
	}

	return false
}

func (d *DiGraph[T]) TraverseBFS(start T) (iter.Seq[T], error) {
	iter, err := traverse.NewBreadthFirstIterator(d.inner, start)
	if err != nil {
		return nil, err
	}

	return func(yield func(T) bool) {
		for iter.HasNext() {
			vertex := iter.Next()
			id := vertex.Label()

			if !yield(id) {
				return
			}
		}
	}, nil
}

func (d *DiGraph[T]) TraverseBFSRoots() (iter.Seq[T], error) {
	iterators := make([]iter.Seq[T], 0)

	for root := range d.Roots() {
		iter, err := d.TraverseBFS(root)
		if err != nil {
			return nil, err
		}

		iterators = append(iterators, typeutils.VisitOnceIter(iter))
	}

	return typeutils.ConcatIter(iterators...), nil
}

func (d *DiGraph[T]) TraverseDFS(start T) (iter.Seq[T], error) {
	iter, err := traverse.NewDepthFirstIterator(d.inner, start)
	if err != nil {
		return nil, err
	}

	return func(yield func(T) bool) {
		for iter.HasNext() {
			vertex := iter.Next()
			id := vertex.Label()

			if !yield(id) {
				return
			}
		}
	}, nil
}

func (d *DiGraph[T]) TraverseDFSRoots() (iter.Seq[T], error) {
	iterators := make([]iter.Seq[T], 0)

	for root := range d.Roots() {
		iter, err := d.TraverseBFS(root)
		if err != nil {
			return nil, err
		}

		iterators = append(iterators, typeutils.VisitOnceIter(iter))
	}

	return typeutils.ConcatIter(iterators...), nil
}

func (d *DiGraph[T]) TraverseTopological() (iter.Seq[T], error) {
	iter, err := traverse.NewTopologicalIterator(d.inner)
	if err != nil {
		return nil, err
	}

	return func(yield func(T) bool) {
		for iter.HasNext() {
			vertex := iter.Next()
			id := vertex.Label()

			if !yield(id) {
				return
			}
		}
	}, nil
}

func (d *DiGraph[T]) TraverseClosestFirst(start T) (iter.Seq[T], error) {
	iter, err := traverse.NewClosestFirstIterator(d.inner, start)
	if err != nil {
		return nil, err
	}

	return func(yield func(T) bool) {
		for iter.HasNext() {
			vertex := iter.Next()
			id := vertex.Label()

			if !yield(id) {
				return
			}
		}
	}, nil
}
