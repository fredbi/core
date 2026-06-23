package pools

import (
	"math/rand"
	"testing"
)

// sizeFor returns a slice size for iteration i: mostly small (~32), with an occasional large spike
// (~16384) every 1-in-rare iterations. This models the OpenAPI pattern where most buffers are tiny
// but the odd large value/spec mints a big backing array that would otherwise circulate forever.
func sizeFor(rng *rand.Rand) int {
	if rng.Intn(100) == 0 { // 1% large
		return 8192 + rng.Intn(16384)
	}
	return 8 + rng.Intn(64)
}

// runSliceWorkload borrows, grows to a drawn size, and redeems — the steady-state churn. Only the
// pool's own (re)allocations are measured: any allocation here is the pool growing a too-small
// backing array, so allocs/op and B/op isolate the bloat-vs-thrash trade-off.
func runSliceWorkload(b *testing.B, p *PoolSlice[int]) {
	b.Helper()
	rng := rand.New(rand.NewSource(42))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s, redeem := p.BorrowWithRedeem()
		s.Grow(sizeFor(rng))
		redeem()
	}
}

// Uncapped: oversized arrays from the 1% large requests are recycled and bloat the pool, but the
// common small path rarely reallocates once warm.
func BenchmarkSliceUncapped(b *testing.B) {
	runSliceWorkload(b, NewPoolSlice[int](WithMinimumCapacity(64)))
}

// Capped at a snug bound just above the common size: large requests are dropped on redeem (bounded
// memory) at the cost of reallocating each large cycle.
func BenchmarkSliceCappedSnug(b *testing.B) {
	runSliceWorkload(b, NewPoolSlice[int](WithMinimumCapacity(64), WithMaxCapacity(128)))
}

// Capped above the large spike: behaves like uncapped here (nothing exceeds the cap) — the control
// showing the cap check itself is ~free.
func BenchmarkSliceCappedLoose(b *testing.B) {
	runSliceWorkload(b, NewPoolSlice[int](WithMinimumCapacity(64), WithMaxCapacity(1<<20)))
}

// poolBloat drains the pool after a workload and sums the capacities still held, as a proxy for the
// pool's retained memory footprint. Single-goroutine and GC-quiet so the sample is stable.
func poolBloat(p *PoolSlice[int], iters, drain int) int {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < iters; i++ {
		s, redeem := p.BorrowWithRedeem()
		s.Grow(sizeFor(rng))
		redeem()
	}
	total := 0
	redeems := make([]func(), 0, drain)
	for i := 0; i < drain; i++ {
		s, redeem := p.BorrowWithRedeem()
		total += s.Cap()
		redeems = append(redeems, redeem)
	}
	for _, r := range redeems {
		r()
	}
	return total
}

// BenchmarkPoolBloat is not a timing benchmark; it reports retained capacity (summed cap of drained
// slices) as a custom metric so we can compare uncapped vs capped memory footprint. Run with
// -benchtime=1x to get a single, comparable sample.
func BenchmarkPoolBloatUncapped(b *testing.B) {
	for i := 0; i < b.N; i++ {
		total := poolBloat(NewPoolSlice[int](WithMinimumCapacity(64)), 10000, 64)
		b.ReportMetric(float64(total), "retained-cap")
	}
}

func BenchmarkPoolBloatCapped(b *testing.B) {
	for i := 0; i < b.N; i++ {
		total := poolBloat(NewPoolSlice[int](WithMinimumCapacity(64), WithMaxCapacity(128)), 10000, 64)
		b.ReportMetric(float64(total), "retained-cap")
	}
}
