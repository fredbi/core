package pools

import (
	"iter"
	"slices"
	"sync"
)

// Resettable is an interface for types that want to recycle a clean instance from a [Pool].
//
// When T (or rather *T) implements Resettable, the pool calls Reset on an instance when it is
// redeemed, so that no references held by the instance are retained while it sits idle in the
// pool, and so that the next borrower receives a clean object.
type Resettable interface {
	Reset()
}

// resetIfResettable calls Reset on v when *T implements [Resettable].
func resetIfResettable[T any](v *T) {
	if r, ok := any(v).(Resettable); ok {
		r.Reset()
	}
}

type redeemable[T any] struct {
	inner    *T
	redeemer func()
}

// Pool wraps a [sync.Pool] to make it available for any type.
//
// T must be the value type of the pooled object (e.g. Pool[bytes.Buffer]): [Pool.Borrow] returns a *T.
// Using a pointer type as T (e.g. Pool[*bytes.Buffer]) would yield a **T and is almost certainly a mistake.
type Pool[T any] struct {
	pool sync.Pool
}

// PoolRedeemable wraps a [sync.Pool] to make it available for any type.
//
// It differs from [Pool] in the way objects are redeemed to the pool: borrowing also yields a
// cached redeem closure, so no closure is allocated at redeem time.
type PoolRedeemable[T any] struct {
	pool sync.Pool
}

// New builds a new [Pool] to recycle allocations of type T explicitly using [Pool.Redeem]
// and the allocated pointer.
//
// Freshly allocated instances of type T are set to their zero value (then Reset if Resettable).
//
// Instances are reset (if they implement [Resettable]) when they are redeemed, not when they are
// borrowed: this clears any references they hold promptly, so the pool does not pin a reference
// graph alive across a GC cycle.
func New[T any]() *Pool[T] {
	p := &Pool[T]{}
	p.pool = sync.Pool{
		New: func() any {
			v := new(T)
			resetIfResettable(v)

			return v
		},
	}

	return p
}

// NewRedeemable builds a new redeemable [Pool] to recycle allocations of type T,
// and use the inner redeemer to relinquish objects to the pool.
func NewRedeemable[T any]() *PoolRedeemable[T] {
	p := &PoolRedeemable[T]{}
	p.pool = sync.Pool{
		New: func() any {
			r := &redeemable[T]{inner: new(T)}
			resetIfResettable(r.inner)
			r.redeemer = func() {
				resetIfResettable(r.inner)
				p.pool.Put(r)
			}

			return r
		},
	}

	return p
}

// Borrow an instance from the pool.
//
// The returned instance is clean: it was reset when last redeemed (or is a fresh zero value).
func (p *Pool[T]) Borrow() *T {
	return p.pool.Get().(*T)
}

// Redeem a borrowed instance to the pool.
//
// A nil pointer is ignored (it would otherwise corrupt the pool: a typed-nil boxed into an
// interface is not the nil interface that [sync.Pool.Put] skips).
//
// The instance is reset (if it implements [Resettable]) before being returned to the pool.
// After calling Redeem, the caller must drop its reference to ptr: continuing to use it is a
// use-after-redeem bug.
func (p *Pool[T]) Redeem(ptr *T) {
	if ptr == nil {
		return
	}
	resetIfResettable(ptr)
	p.pool.Put(ptr)
}

// BorrowWithRedeem borrows an instance from the pool and provides the
// corresponding redeem function.
//
// This is useful for instance to use with defer.
//
// The instance is reset (if it implements [Resettable]) when the returned redeem closure is
// called, not when it is borrowed. After calling the redeem closure, the caller must drop its
// reference to the returned instance.
func (p *PoolRedeemable[T]) BorrowWithRedeem() (*T, func()) {
	container := p.pool.Get().(*redeemable[T])

	return container.inner, container.redeemer
}

// Slice is a struct that wraps a slice []T.
//
// This is useful to borrow and redeem slices from a pool, without having to constantly manipulate
// pointers to the slice.
//
// The wrapper must remain the single source of truth for the slice header: always mutate through
// its methods ([Slice.Append], [Slice.Grow], [Slice.Concat]). If you grow the raw slice returned
// by [Slice.Slice] using the builtin append, the regrown backing array lives only in your local
// copy and is lost when the wrapper is redeemed.
type Slice[T any] struct {
	length int
	inner  []T
}

// Slice returns the inner slice.
//
// Treat the result as a read-only view for ranging. To grow or append, use the wrapper methods so
// the new backing array is tracked and recycled (see [Slice]).
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
//
// Unlike [slices.Concat], this reuses the inner slice's capacity instead of always allocating a
// fresh backing array.
func (s *Slice[T]) Concat(slice []T) []T {
	s.inner = append(s.inner, slice...)

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

// Reset the inner slice to its configured initial length, keeping allocated capacity.
//
// All elements are zeroed, so the pool never retains stale element references (which would keep a
// referenced graph alive for slices of pointers) and so a [WithLength] slice is handed out clean
// rather than carrying data from a previous borrower.
func (s *Slice[T]) Reset() {
	clear(s.inner)
	if s.length > cap(s.inner) {
		s.inner = slices.Grow(s.inner[:0], s.length)
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
// Use [PoolSlice.BorrowWithSizeAndRedeem] or [Slice.Grow] to grow the capacity of the inner slice.
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

	p.pool = sync.Pool{
		New: func() any {
			s := &redeemable[Slice[T]]{
				inner: &Slice[T]{
					length: o.length,
					inner:  make([]T, o.length, max(o.length, o.minCapacity)),
				},
			}

			s.redeemer = func() {
				s.inner.Reset()
				p.pool.Put(s)
			}

			return s
		},
	}

	return p
}

// BorrowWithRedeem returns the slice wrapper and the redeem closure to relinquish the allocated wrapper.
//
// The wrapper is reset (elements zeroed, length restored) when the redeem closure is called.
func (p *PoolSlice[T]) BorrowWithRedeem() (*Slice[T], func()) {
	return p.PoolRedeemable.BorrowWithRedeem()
}

// BorrowWithSizeAndRedeem borrows a slice []T from the pool and ensures that its capacity
// is at least the provided size.
func (p *PoolSlice[T]) BorrowWithSizeAndRedeem(size int) (*Slice[T], func()) {
	s, redeem := p.BorrowWithRedeem()
	s.Grow(size)

	return s, redeem
}
