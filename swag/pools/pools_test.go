package pools

import (
	"sync"
	"testing"
)

// resettable is a Resettable type that records how many times Reset was called and carries a
// reference we can observe to verify the pool does not pin it while idle.
type resettable struct {
	resets int
	ref    *int
	data   int
}

func (r *resettable) Reset() {
	r.resets++
	r.ref = nil
	r.data = 0
}

func TestPoolBorrowRedeem(t *testing.T) {
	p := New[resettable]()

	a := p.Borrow()
	a.data = 42
	x := 7
	a.ref = &x
	p.Redeem(a)

	// borrowing again should hand back a clean (reset) instance.
	b := p.Borrow()
	if b.data != 0 {
		t.Fatalf("expected data cleared on reuse, got %d", b.data)
	}
	if b.ref != nil {
		t.Fatalf("expected ref cleared on reuse, got %v", b.ref)
	}
}

func TestPoolResetOnBorrowAndRedeem(t *testing.T) {
	p := New[resettable]()

	a := p.Borrow() // fresh: new(T), then reset on borrow
	if a.resets != 1 {
		t.Fatalf("expected 1 reset after fresh borrow, got %d", a.resets)
	}
	p.Redeem(a) // reset on redeem
	if a.resets != 2 {
		t.Fatalf("expected 2 resets after redeem, got %d", a.resets)
	}

	b := p.Borrow() // recycled: reset on borrow
	if b != a {
		t.Skip("pool did not return the same instance; reset-timing assertion not applicable")
	}
	if b.resets != 3 {
		t.Fatalf("expected 3 resets after re-borrow, got %d", b.resets)
	}
}

func TestPoolRedeemNilIsSafe(t *testing.T) {
	p := New[resettable]()

	// Must not panic and must not poison the pool with a typed-nil.
	p.Redeem(nil)

	got := p.Borrow()
	if got == nil {
		t.Fatal("pool handed back a nil pointer after Redeem(nil) poisoned it")
	}
	got.data = 1 // would panic on a nil pointer
}

func TestRedeemableBorrowWithRedeem(t *testing.T) {
	p := NewRedeemable[resettable]()

	v, redeem := p.BorrowWithRedeem()
	if v == nil || redeem == nil {
		t.Fatal("expected non-nil instance and redeemer")
	}
	v.data = 99
	x := 3
	v.ref = &x
	redeem()

	w, redeem2 := p.BorrowWithRedeem()
	if w.data != 0 || w.ref != nil {
		t.Fatalf("expected clean instance on reuse, got data=%d ref=%v", w.data, w.ref)
	}
	redeem2()
}

func TestRedeemableDoubleRedeemPanics(t *testing.T) {
	p := NewRedeemable[resettable]()

	_, redeem := p.BorrowWithRedeem()
	redeem() // first redeem: fine

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected a panic on double redeem")
		}
	}()
	redeem() // second redeem: must panic
}

func TestRedeemableReborrowRearmsState(t *testing.T) {
	p := NewRedeemable[resettable]()

	// borrow/redeem several times: the state must be re-armed on each borrow so redeem keeps working.
	for i := 0; i < 5; i++ {
		_, redeem := p.BorrowWithRedeem()
		redeem()
	}
}

func TestPoolSliceDoubleRedeemPanics(t *testing.T) {
	p := NewPoolSlice[int]()

	s, redeem := p.BorrowWithRedeem()
	s.Append(1, 2, 3)
	redeem()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected a panic on double redeem of a pooled slice")
		}
	}()
	redeem()
}

func TestRedeemableZeroAllocRedeem(t *testing.T) {
	p := NewRedeemable[resettable]()

	// warm the pool
	v, redeem := p.BorrowWithRedeem()
	redeem()
	_ = v

	allocs := testing.AllocsPerRun(100, func() {
		x, r := p.BorrowWithRedeem()
		x.data++
		r()
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocs on warm borrow/redeem, got %v", allocs)
	}
}

func TestSliceBasic(t *testing.T) {
	p := NewPoolSlice[int]()

	s, redeem := p.BorrowWithRedeem()
	if s.Len() != 0 {
		t.Fatalf("expected empty slice, got len %d", s.Len())
	}
	s.Append(1, 2, 3)
	if s.Len() != 3 {
		t.Fatalf("expected len 3, got %d", s.Len())
	}
	if got := s.Slice(); len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatalf("unexpected slice contents: %v", got)
	}
	redeem()

	// reuse: must be reset to length 0.
	s2, redeem2 := p.BorrowWithRedeem()
	if s2.Len() != 0 {
		t.Fatalf("expected reset slice len 0, got %d", s2.Len())
	}
	redeem2()
}

func TestSliceResetZeroesElementReferences(t *testing.T) {
	// White-box: verify Reset zeroes the whole backing array so pointer elements are not retained.
	var s Slice[*int]
	a, b, c := 1, 2, 3
	s.Append(&a, &b, &c)
	full := s.inner[:cap(s.inner)]

	s.Reset()

	if s.Len() != 0 {
		t.Fatalf("expected len 0 after reset, got %d", s.Len())
	}
	for i, ptr := range full {
		if ptr != nil {
			t.Fatalf("element %d not cleared after reset: %v", i, ptr)
		}
	}
}

func TestSliceWithLengthIsZeroedAndSized(t *testing.T) {
	p := NewPoolSlice[int](WithLength(4), WithMinimumCapacity(8))

	s, redeem := p.BorrowWithRedeem()
	if s.Len() != 4 {
		t.Fatalf("expected fixed length 4, got %d", s.Len())
	}
	if s.Cap() < 8 {
		t.Fatalf("expected cap >= 8, got %d", s.Cap())
	}
	for i, v := range s.Slice() {
		if v != 0 {
			t.Fatalf("expected zeroed element at %d, got %d", i, v)
		}
	}
	// dirty it, redeem, and confirm it comes back zeroed at the configured length.
	raw := s.Slice()
	for i := range raw {
		raw[i] = i + 1
	}
	redeem()

	s2, redeem2 := p.BorrowWithRedeem()
	if s2.Len() != 4 {
		t.Fatalf("expected fixed length 4 on reuse, got %d", s2.Len())
	}
	for i, v := range s2.Slice() {
		if v != 0 {
			t.Fatalf("expected zeroed element at %d on reuse, got %d", i, v)
		}
	}
	redeem2()
}

func TestSliceConcatReusesCapacity(t *testing.T) {
	var s Slice[int]
	s.Grow(16)
	before := cap(s.inner)
	s.Concat([]int{1, 2, 3})
	s.Concat([]int{4, 5})
	if got := s.Slice(); len(got) != 5 || got[4] != 5 {
		t.Fatalf("unexpected concat result: %v", got)
	}
	if cap(s.inner) != before {
		t.Fatalf("concat reallocated backing array: before=%d after=%d", before, cap(s.inner))
	}
}

// Reset is what preserves grown capacity through a redeem (it does not clip). We test that directly:
// asserting capacity survives a *pool* round-trip would be unsound, since sync.Pool is free to drop
// an idle object and hand back a fresh one (it routinely does under -race).
func TestSliceResetPreservesCapacity(t *testing.T) {
	var s Slice[int]
	s.Grow(1024)
	grown := s.Cap()
	if grown < 1024 {
		t.Fatalf("expected cap >= 1024 after grow, got %d", grown)
	}
	s.Append(1, 2, 3)
	s.Reset()

	if s.Len() != 0 {
		t.Fatalf("expected len 0 after reset, got %d", s.Len())
	}
	if s.Cap() != grown {
		t.Fatalf("Reset must preserve grown capacity: before=%d after=%d", grown, s.Cap())
	}
}

// resetWithCapacity is the drop path a capped pool uses to discard an oversized backing array.
func TestSliceResetWithCapacityDropsBacking(t *testing.T) {
	var s Slice[int]
	s.Grow(4096)
	s.resetWithCapacity(64)

	if s.Len() != 0 {
		t.Fatalf("expected len 0, got %d", s.Len())
	}
	if s.Cap() != 64 {
		t.Fatalf("expected capacity dropped to 64, got %d", s.Cap())
	}
}

func TestWithMaxCapacityShrinksOversized(t *testing.T) {
	p := NewPoolSlice[int](WithMinimumCapacity(8), WithMaxCapacity(64))

	s, redeem := p.BorrowWithRedeem()
	s.Grow(1024)
	if s.Cap() < 1024 {
		t.Fatalf("expected cap >= 1024 after grow, got %d", s.Cap())
	}
	redeem() // cap > 64 → backing should be discarded and replaced

	s2, redeem2 := p.BorrowWithRedeem()
	if s2.Cap() > 64 {
		t.Fatalf("expected oversized backing to be dropped on redeem, got cap %d", s2.Cap())
	}
	if s2.Cap() < 8 {
		t.Fatalf("expected replacement to honor minimum capacity 8, got %d", s2.Cap())
	}
	redeem2()
}

func TestWithMaxCapacityHonorsLength(t *testing.T) {
	p := NewPoolSlice[int](WithLength(4), WithMaxCapacity(64))

	s, redeem := p.BorrowWithRedeem()
	s.Grow(1024)
	redeem() // dropped: replacement must still be a clean length-4 slice

	s2, redeem2 := p.BorrowWithRedeem()
	if s2.Len() != 4 {
		t.Fatalf("expected replacement length 4, got %d", s2.Len())
	}
	for i, v := range s2.Slice() {
		if v != 0 {
			t.Fatalf("expected zeroed replacement element at %d, got %d", i, v)
		}
	}
	redeem2()
}

func TestConcurrentBorrowRedeem(t *testing.T) {
	p := New[resettable]()
	ps := NewPoolSlice[int]()

	var wg sync.WaitGroup
	for g := 0; g < 50; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				v := p.Borrow()
				v.data = i
				p.Redeem(v)

				s, redeem := ps.BorrowWithRedeem()
				s.Append(i, i+1)
				redeem()
			}
		}()
	}
	wg.Wait()
}
