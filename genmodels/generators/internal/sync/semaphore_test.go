package sync

import (
	"context"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestSemaphore(t *testing.T) {
	const n = 100
	input := make([]int64, 0, n)
	for i := range n {
		input = append(input, int64(i-1))
	}

	t.Run("should process a shuffled list of indices and render it ordered", func(t *testing.T) {
		s := NewWatermarkSemaphore()

		var x sync.Mutex

		shuffle := make([]int64, n)
		copy(shuffle, input)
		for i := range n {
			j := rand.IntN(n) //nolint:gosec
			shuffle[i], shuffle[j] = shuffle[j], shuffle[i]
		}

		grp, ctx := errgroup.WithContext(t.Context())
		result := make([]int64, 0, n)

		for i := range n {
			grp.Go(func() error {
				// process input [i]
				index := shuffle[i]
				if err := s.Acquire(ctx, index); err != nil {
					return err
				} // wait for i
				x.Lock()
				result = append(result, index)
				x.Unlock()
				s.Release(index + 1) // release i+1

				return nil
			})
		}

		require.NoError(t, grp.Wait())

		t.Run("result should be ordered", func(t *testing.T) {
			require.True(t, slices.IsSorted(result))
		})
	})

	t.Run("the nil semaphore is valid", func(t *testing.T) {
		var s *WatermarkSemaphore

		require.NoError(t, s.Acquire(t.Context(), 0))
		s.Release(1)
	})

	t.Run("should unblock whenever the parent context is cancelled", func(t *testing.T) {
		s := NewWatermarkSemaphore()

		parentCtx, cancel := context.WithCancel(t.Context())
		grp, ctx := errgroup.WithContext(parentCtx)

		for i := range n {
			grp.Go(func() error {
				// process input [i]
				index := input[i]
				if err := s.Acquire(ctx, index+1); err != nil {
					return err
				} // wait for i+1
				s.Release(index + 1) // release i+2

				return nil
			})
		}

		time.Sleep(100 * time.Millisecond) // let go routines stack up and block
		cancel()
		err := grp.Wait()
		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
	})
}
