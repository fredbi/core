package typeutils

import "iter"

// ConcatIter concatenates a list of iterators.
func ConcatIter[E any](seqs ...iter.Seq[E]) iter.Seq[E] {
	return func(yield func(E) bool) {
		for _, seq := range seqs {
			for e := range seq {
				if !yield(e) {
					return
				}
			}
		}
	}
}

func FilterIter[E any](seq iter.Seq[E], filter func(E) bool) iter.Seq[E] {
	return func(yield func(E) bool) {
		for e := range seq {
			if !filter(e) {
				continue
			}

			if !yield(e) {
				return
			}
		}
	}
}

func VisitOnceIter[E comparable](seq iter.Seq[E]) iter.Seq[E] {
	visited := make(map[E]struct{})

	return FilterIter(seq, filterVisited(visited))
}

func filterVisited[T comparable](visited map[T]struct{}) func(T) bool {
	return func(id T) bool {
		_, ok := visited[id]
		if !ok {
			visited[id] = struct{}{}

			return true
		}

		return false
	}
}

func ApplyIter[E any](seq iter.Seq[E], apply func(*E)) iter.Seq[E] {
	return func(yield func(E) bool) {
		for e := range seq {
			apply(&e)

			if !yield(e) {
				return
			}
		}
	}
}

func TransformErrIter[E any, F any](seq iter.Seq[E], transform func(E) (F, error)) iter.Seq[F] {
	return func(yield func(F) bool) {
		for e := range seq {
			te, err := transform(e)
			if err != nil {
				continue
			}

			if !yield(te) {
				return
			}
		}
	}
}

func TransformIter[E any, F any](seq iter.Seq[E], transform func(E) (F, bool)) iter.Seq[F] {
	return func(yield func(F) bool) {
		for e := range seq {
			te, ok := transform(e)
			if !ok {
				continue
			}

			if !yield(te) {
				return
			}
		}
	}
}
