package sync

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
)

// WatermarkSemaphore blocks acquirers with a "minRequired" requirement until the index has been released up to that level.
//
// Negative requirement indices are not blocked.
//
// The nil value is valid and provides a dummy (no-op) semaphore.
type WatermarkSemaphore struct {
	lastProcessedIndex atomic.Int64
	waiters            ordered
	sync.Mutex
}

// NewWatermarkSemaphore builds a new [WatermarkSemaphore] ready to be used.
func NewWatermarkSemaphore() *WatermarkSemaphore {
	const preallocatedWaiters = 10

	s := &WatermarkSemaphore{
		waiters: make(ordered, 0, preallocatedWaiters),
	}

	s.lastProcessedIndex.Store(-1) // first blocking requirement is 0.

	return s
}

func (s *WatermarkSemaphore) Acquire(ctx context.Context, minRequired int64) error {
	if s == nil {
		return nil
	}

	select {
	// short circuit when the parent context is already done
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// short circuit when the requirement is already met: no locking needed
	if s.lastProcessedIndex.Load() >= minRequired {
		return nil
	}

	// add to the wait list
	ready := make(chan struct{})
	w := waiter{
		n:     minRequired,
		ready: ready,
	}

	s.Lock()
	if s.lastProcessedIndex.Load() >= minRequired {
		// check again, now protected by the lock: don't wait if the requirement is met
		s.Unlock()
		return nil
	}

	s.waiters = s.waiters.Push(w)
	s.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ready:
	}

	return nil
}

func (s *WatermarkSemaphore) Release(index int64) {
	if s == nil {
		return
	}

	s.Lock()
	last := s.lastProcessedIndex.Load()
	if index > last {
		_ = s.lastProcessedIndex.Swap(index)
		last = index
	}
	s.notifyWaiters(last)
	s.Unlock()
}

func (s *WatermarkSemaphore) notifyWaiters(index int64) {
	if len(s.waiters) == 0 {
		return
	}

	i := len(s.waiters)
	for ; i >= 1; i-- {
		w := s.waiters[i-1]
		if !w.CheckReady(index) {
			break
		}
	}

	s.waiters = s.waiters[:i]
}

type waiter struct {
	n     int64
	ready chan<- struct{} // closed when semaphore acquired.
}

func (w *waiter) CheckReady(index int64) bool {
	if index >= w.n {
		close(w.ready)

		return true
	}

	return false
}

// Compare waiters by their index.
//
// Yields -1 when w > b, +1 when w < b and 0 when w == b.
func (w waiter) Compare(b waiter) int {
	switch {
	case w.n == b.n:
		return 0
	case w.n > b.n:
		return -1
	default:
		return 1
	}
}

/*
// String representation of a waiter.
func (w waiter) String() string {
	return strconv.FormatInt(w.n, 10)
}
*/

// ordered list of waiters, the lowest requirements are maintained at the tail of the list.
type ordered []waiter

// Push a waiter into the ordered list, putting the lower requirement indices at the end.
func (o ordered) Push(w waiter) ordered {
	if len(o) == 0 || w.n <= o[len(o)-1].n {
		o = append(o, w)

		return o
	}

	idx, _ := slices.BinarySearchFunc(o, w, func(a, b waiter) int { return a.Compare(b) })
	o = slices.Insert(o, idx, w)

	return o
}
