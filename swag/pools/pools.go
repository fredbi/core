package pools

import (
	"iter"
	"slices"
	"sync"
)

// Resettable is an interface for types that want to recycle a clean instance from a [Pool].
type Resettable interface {
	Reset()
}

type redeemable[T any] struct {
	inner    *T
	redeemer func()
}

// Pool wraps a [sync.Pool] to make it available for any type.
type Pool[T any] struct {
	sync.Pool
}

// PoolRedeemable wraps a [sync.Pool] to make it available for any type.
//
// It differs from [Pool] in the way objects are redeemed to the pool.
type PoolRedeemable[T any] struct {
	sync.Pool
}

// New builds a new [Pool] to recycle allocations of type T explictly using Redeem
// and the allocated pointer.
//
// Allocated instances of type T are set to their zero value.
//
// Recycled instances retrieved using [Pool[T].Borrow] are reset if they implement [Resettable].
func New[T any]() *Pool[T] {
	return &Pool[T]{
		Pool: sync.Pool{
			New: func() any {
				var zero T

				return &zero
			},
		},
	}
}

// NewRedeemable builds a new redeemable [Pool] to recycle allocations of type T,
// and use the inner Redeemer to relinquish objects to the pool.
func NewRedeemable[T any]() *PoolRedeemable[T] {
	p := &PoolRedeemable[T]{}
	p.Pool = sync.Pool{
		New: func() any {
			var zero redeemable[T]
			var innerZero T
			zero.inner = &innerZero
			zero.redeemer = func() {
				p.Pool.Put(&zero)
			}

			return &zero
		},
	}

	return p
}

// Borrow an instance from the pool.
//
// If the type T is [Resettable] the returned value is reset.
func (p *Pool[T]) Borrow() *T {
	ptr := p.Pool.Get()
	target := ptr.(*T)

	if resettable, ok := any(target).(Resettable); ok {
		resettable.Reset()
	}

	return target
}

// Redeem a borrowed instance to the pool.
func (p *Pool[T]) Redeem(ptr *T) {
	p.Pool.Put(ptr)
}

// BorrowWithRedeem borrows an instance from the pool and provides the
// corresponding redeem function.
//
// This is useful for instance to use with defer.
func (p *PoolRedeemable[T]) BorrowWithRedeem() (*T, func()) {
	ptr := p.Get()
	container := ptr.(*redeemable[T])

	if resettable, ok := any(container.inner).(Resettable); ok {
		resettable.Reset()
	}

	return container.inner, container.redeemer
}

// Slice is a struct that wraps a slice []T.
//
// This is useful to borrow and redeem slices from a pool, without having to constantly manipulate
// pointers to the slice.
type Slice[T any] struct {
	length int
	inner  []T
}

// Slice returns the inner slice.
func (s *Slice[T]) Slice() []T {
	return s.inner
}

// Grow the inner slice.
func (s *Slice[T]) Grow(size int) []T {
	s.inner = slices.Grow(s.inner, size)

	return s.inner
}

func (s *Slice[T]) Len() int {
	return len(s.inner)
}

func (s *Slice[T]) Cap() int {
	return cap(s.inner)
}

// Append elements to the inner slice.
//
// This should be prefered to the append builtin if you plan that the slice will
// grow and you want to reuse the newly allocated space.
func (s *Slice[T]) Append(elems ...T) []T {
	s.inner = append(s.inner, elems...)

	return s.inner
}

// Concat another slice to the inner slice.
func (s *Slice[T]) Concat(slice []T) []T {
	s.inner = slices.Concat(s.inner, slice)

	return s.inner
}

// IndexedElems iterates over the inner slice.
func (s *Slice[T]) IndexedElems() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for i, elem := range s.inner {
			if !yield(i, elem) {
				return
			}
		}
	}
}

// Reset the inner slice, but keep allocated memory
func (s *Slice[T]) Reset() {
	if s.length > cap(s.inner) {
		s.inner = slices.Grow(s.inner, s.length)
	}

	s.inner = s.inner[:s.length]
}

// Clip removes unused capacity from the inner slice.
func (s *Slice[T]) Clip() {
	s.inner = slices.Clip(s.inner)
}

// PoolSlice is a pool of [Slice[T]].
//
// [PoolSlice.BorrowWithRedeem] will return an empty inner slice by default.
// This default may be altered using [WithMinimumCapacity].
//
// Use [PoolSlice.BorrowWithSize] or [Slice.Grow] to grow the capacity of the inner slice.
type PoolSlice[T any] struct {
	*PoolRedeemable[Slice[T]]
}

// PoolSliceOption alters the default settings to allocate new pooled slices
type PoolSliceOption func(*poolSliceOptions)

type poolSliceOptions struct {
	minCapacity int
	length      int
}

func WithMinimumCapacity(size int) PoolSliceOption {
	return func(o *poolSliceOptions) {
		o.minCapacity = size
	}
}

// WithLength ensures that the borrowed slices have a fixed given initial length.
//
// By default, the borrowed slices are reset to length 0.
func WithLength(size int) PoolSliceOption {
	return func(o *poolSliceOptions) {
		o.length = size
	}
}

// NewPoolSlice builds a pool to recycle slices of type []T.
func NewPoolSlice[T any](opts ...PoolSliceOption) *PoolSlice[T] {
	var o poolSliceOptions
	for _, apply := range opts {
		apply(&o)
	}

	p := &PoolSlice[T]{
		PoolRedeemable: &PoolRedeemable[Slice[T]]{},
	}

	p.Pool = sync.Pool{
		New: func() any {
			s := &redeemable[Slice[T]]{
				inner: &Slice[T]{
					length: o.length,
					inner:  make([]T, o.length, max(o.length, o.minCapacity)),
				},
			}

			s.redeemer = func() { p.Pool.Put(s) }

			return s
		},
	}

	return p
}

// BorrowWithRedeem returns the slice wrapper and the redeem closure to relinquish the allocated wrapper.
func (p *PoolSlice[T]) BorrowWithRedeem() (*Slice[T], func()) {
	wrapper, redeem := p.PoolRedeemable.BorrowWithRedeem()

	return wrapper, redeem
}

// BorrowWithSizeAndRedeem borrows a slice []T from the pool and ensures that its capacity
// is at least the provided size.
func (p *PoolSlice[T]) BorrowWithSizeAndRedeem(size int) (*Slice[T], func()) {
	s, redeem := p.BorrowWithRedeem()
	s.Grow(size)

	return s, redeem
}
