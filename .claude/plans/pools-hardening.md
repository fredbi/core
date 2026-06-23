# pools hardening & extension roadmap

> Status: **DRAFT for review** — iterate before committing to execution.
> Scope: `swag/pools` (generic pooling utility used across go-openapi).
> Branch: `worktree-pools-hardening`
> Last updated: 2026-06-23

## Progress

- ✅ **Phase 1 — Correctness + tests** (C1, C2, C3, C5, C7). D2 resolved (reset both sides).
- ✅ **Phase 2 — API hardening** (§4). D1 + D3 resolved. C6 closed (docs), de-embedded PoolSlice,
  always-on double-redeem panic guard on the redeemable pools. 14 tests, 86.4% cov, race-clean.
- ⏳ Phase 3 — Slice bloat (§5)
- ⏳ Phase 4 — Debug pool (§6) — full double-redeem/leak/ABA tracking + plain `Pool[T]` coverage
- ⏳ Phase 5 — Shared pools (§7)

## Legend

- ✅ done
- 🚧 in progress
- ⏳ planned / not started
- 🔬 needs design decision (see "Open design decisions")
- 💭 stretch / far-out

---

## 0. Context & constraints

- Library is **WIP, unpublished** → we fix aggressively, no backward-compat burden.
- It exists to make `sync.Pool` usage **less bug-prone**. Footguns are bugs.
- Primary consumer profile: OpenAPI processing → many large, reference-heavy
  objects, high allocation churn. Memory retention matters as much as CPU.
- Other agents work in parallel elsewhere in the repo; we stay in `swag/pools`.
- Deep-dive priorities (per review): **slice bloat (§5)** and **API hardening (§4)**.
  Correctness fixes (§3) are non-negotiable. Debug pool (§6) and shared pools (§7)
  are in-brief but lighter detail / later phases.

---

## 1. Vision

A small, **exactly-correct** generic pooling toolkit:

1. **One obvious right way** to borrow/redeem; the wrong way should be hard or
   impossible to express (no leaked `sync.Pool.Get/Put`).
2. **Memory-safe by default**: pooled objects don't pin reference graphs alive;
   slices don't retain stale element pointers.
3. **Provably-correct usage in tests**: a build-tagged instrumented pool that
   panics on double-redeem / borrow-of-unredeemed and asserts no leaks — generic,
   replacing `go-openapi/validate`'s 1000-line hand-rolled version.
4. **Bounded memory under growth**: slice pools don't degrade into a graveyard of
   oversized backing arrays.

---

## 2. Current state (baseline, 2026-06-23)

- ✅ Builds / vets / tests clean in isolation (`GOWORK=off`; repo `go.work`
  references temp modules owned by other agents).
- Surface: `Pool[T]`, `PoolRedeemable[T]`, `Slice[T]`, `PoolSlice[T]`,
  options `WithMinimumCapacity` / `WithLength`.
- `pools_test.go` is an empty `t.SkipNow()`. **No real tests, no benchmarks.**
- `testing/` sub-package exists with only a doc.go stub (intended home for the
  debug pool per `TODO.md`).

---

## 3. Correctness findings (fix aggressively) — Phase 1

| ID | Status | Severity | Finding | Fix |
|----|--------|----------|---------|-----|
| C1 | ✅ | high | `Redeem(nil)` / typed-nil `*T` is stored (interface non-nil), later handed back → nil-receiver `Reset()` panic / nil borrow. | Guard `if ptr == nil { return }` before `Put`. |
| C2 | ✅ | high | `Reset()` runs **on borrow**, so idle pooled objects pin their whole reference graph across a GC cycle. | **D2 resolved: reset on BOTH borrow and redeem.** Redeem clears refs promptly (no idle pinning); borrow guarantees a clean object regardless of history. `Reset` must be idempotent (runs ≥2×/cycle). |
| C3 | ✅ | high | `Slice.Reset()` reslices `[:length]`, never zeroing `[len:cap]` (and `[0:length]` for `WithLength`) → leaks element pointers, exposes stale data. | `clear()` whole used region before reslice; `WithLength` region zeroed. Append-monotonic-len invariant makes this complete. |
| C4 | 🚧 | high | Double-redeem / use-after-redeem is silent; cached redeemer makes `defer`+manual easy → `sync.Pool` corruption (one object to two borrowers). | **D3 resolved (partial, Phase 2):** always-on `atomic.Uint32` state on the redeemable wrapper panics loudly on a redeem of an already-idle slot; re-armed on borrow. Catches the common double-redeem. **Residual:** ABA (redeem racing a re-borrow of the same slot) and plain `Pool[T]` (no wrapper) → debug build, Phase 4. |
| C5 | ⏳ | med | Embedding `sync.Pool` exposes `.Get()/.Put()` returning the wrong type (`*redeemable[T]`, `any`). | **Done early:** `sync.Pool` is now an unexported `pool` field (no longer embedded). Public surface = `Borrow`/`Redeem`/`BorrowWithRedeem`. |
| C6 | ✅ | med | `Slice.Slice()` returns raw `[]T`, inviting builtin `append` whose regrown array is lost on redeem (defeats the whole point). | **D1 resolved:** keep idiomatic `[]T` returns but document the snapshot/growth caveat hard ("use the wrapper if you plan to grow"). Also de-embedded `PoolSlice`'s `*PoolRedeemable` → unexported field. |
| C7 | ✅ | low | `Pool[*X]` yields `**X`; `Concat` always allocates (defeats reuse). | Value-type contract documented; `Concat` now append-based. |

**Phase 1 added the missing test suite** (was `t.SkipNow()`): borrow/redeem round-trips,
reset timing (on redeem not borrow), nil-safety, slice element-zeroing/leak, `WithLength`
sizing, concat cap reuse, growth-survives-redeem, zero-alloc warm redeem, concurrency (race).
**11 tests, 85.2% coverage, `-race` clean.**

Note: C5 (unexport `sync.Pool`) shipped in Phase 1 because the rewrite touched every
constructor anyway; the `Slice` API reshape part of C6 remains for Phase 2.

---

## 4. API hardening (deep dive) — Phase 2  🔬

Goal: make misuse unrepresentable; keep the zero-alloc redeemer trick.

### 4.1 Stop leaking `sync.Pool`
- Change `Pool[T]` / `PoolRedeemable[T]` to hold an **unexported** `pool sync.Pool`
  field instead of embedding. Public surface becomes exactly:
  - `Pool[T]`: `Borrow() *T`, `Redeem(*T)`.
  - `PoolRedeemable[T]`: `BorrowWithRedeem() (*T, func())` (+ maybe `Borrow`/`Redeem`).
- Removes the `*redeemable[T]` type-assertion panic vector entirely.

### 4.2 Tame the `Slice` append footgun (C6)  🔬 design decision
Options to debate:
- **(a) Keep returning `[]T` but rename/doc**: `Slice()` → document "read-only view;
  mutate via `Append`/`Grow` or you lose pooling." Cheapest, still a footgun.
- **(b) Make the wrapper the only handle**: drop methods that hand back a `[]T` the
  caller is tempted to `append` to; provide `At(i)`, `Set(i,v)`, `Append`, `Len`,
  `Cap`, `Range`/iter, and a single `Detach() []T` for the "I'm done, take ownership"
  exit. `Slice()` stays for read-only ranging.
- **(c) Redeemer re-syncs**: have redeem read back the latest header — impossible,
  since a caller's local `append` never reaches the wrapper. Rejected.
- Leaning **(b)**: the wrapper *is* the abstraction; returning a live `[]T` for
  mutation is the original sin. Needs your call.

### 4.3 Redeemer ergonomics
- Confirm the cached-redeemer (no closure alloc at redeem) survives the refactor —
  benchmark `BorrowWithRedeem` alloc count (target: 0 allocs on warm pool).
- Consider a `Redeem` that nils the caller's handle is impossible in Go (no `&handle`
  passed) — instead document the "drop your reference after redeem" rule and let the
  debug pool enforce it.

---

## 5. Slice bloat strategy (deep dive) — Phase 3  🔬

**Problem (your point 5):** growing callers replace the backing array; the pool
slowly fills with oversized slices. Bucketing by size is CPU overhead; doing
nothing wastes memory and still allocates during warm-up.

**Proposed default — capacity cap on redeem (cheap, no bucketing):**
- `WithMaxCapacity(n)`: on redeem, if `cap(inner) > n`, **don't** return it to the
  pool (let it GC); optionally return a fresh right-sized one. Bounds steady-state
  memory with a single branch per redeem, zero per-op cost otherwise.
- Symmetric `WithMinimumCapacity` already exists for warm-up; pair them so the pool
  converges to `[minCap, maxCap]` backing arrays.

**Opt-in — size-classed pool (separate type):**
- `PoolSlicedBuckets[T]` with power-of-two (or configurable) size classes, each a
  `sync.Pool`. `Borrow(size)` rounds up to the class; redeem routes by `cap`.
- Only for callers who measured and want it; not the default (keeps the common path
  branch-free).

**Measurement first:** write benchmarks that model the real pattern (borrow → grow a
few times → redeem, repeated) and compare: current, cap-on-redeem, size-classed.
Decide defaults from numbers, not intuition. 🔬

Open questions:
- Does cap-on-redeem + min-capacity seeding eliminate most warm-up allocs in practice?
- Right-size-on-drop: when we drop an oversized slice, do we hand the pool a fresh
  `minCap` slice to keep it warm, or just let `New` handle it?

---

## 6. Debug / test pool — Phase 4 (in brief, lighter detail)

Generic, build-tagged (`//go:build poolsdebug`) instrumented variant living in
`swag/pools/testing` (or same package, tag-gated). Collapses validate's per-type
duplication into one generic type:
- Track per-pointer status: fresh → recycled → redeemed; panic on double-redeem and
  borrow-of-unredeemed.
- Record alloc + redeem **call sites** (`runtime.Caller`) for diagnostics.
- `AssertAllRedeemed(t testing.TB)` for `t.Cleanup` leak detection.
- Mutex-guarded maps (matches reference). Zero cost when tag absent.
- Directly answers C4. Highest external value (every consumer gets it free).

---

## 7. Shared common pools — Phase 5 (in brief, lighter detail)

Ready-made shared pools in a sub-package so consumers share warm pools:
- `Bytes` (`*PoolSlice[byte]`), a `*bytes.Buffer` pool, a `*bytes.Reader` pool.
- Caveat: shared `[]byte` is safe; shared stateful objects need disciplined Reset.
- Value-type contract from C7 applies (`Pool[bytes.Buffer]`, not `Pool[*bytes.Buffer]`).

---

## 8. Sequencing

1. **Phase 1 — Correctness + tests** (C1–C7). Build the safety net first.
2. **Phase 2 — API hardening** (§4): unexport pool, reshape `Slice`. Settle surface
   before anything builds on it.
3. **Phase 3 — Slice bloat** (§5): benchmarks → cap-on-redeem default → optional buckets.
4. **Phase 4 — Debug pool** (§6).
5. **Phase 5 — Shared pools** (§7).

Pause after Phase 2 to reassess (Fred's cadence): surface is the contract everything
else builds on.

---

## 9. Open design decisions (🔬)

- ~~**D1 (§4.2):** `Slice` API shape.~~ **Resolved: keep idiomatic `[]T` returns + hard docs**
  (snapshot semantics; use the wrapper to grow). De-embedded `PoolSlice` internals.
- ~~**D2 (§3 C2):** reset-on-redeem only, or both sides behind an option?~~ **Resolved: reset on both borrow and redeem** (memory window + last-moment defensiveness). `Reset` contract is now "must be idempotent".
- ~~**D3 (§3 C4):** any cheap non-debug double-redeem guard worth its cost?~~ **Resolved: yes —**
  always-on atomic-state panic guard on the redeemable wrapper (re-armed on borrow). Residual ABA +
  plain `Pool[T]` deferred to the Phase 4 debug build.
- **D4 (§5):** defaults from benchmarks — cap-on-redeem alone, or ship buckets too?
- **D5 (§3 C3):** zero element tail always, or fast-path value types that don't need it
  (via a `Resettable`-style marker / `reflect`-free detection)?
