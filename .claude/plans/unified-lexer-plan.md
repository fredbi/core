# Unified lexer (generics) — execution plan

> Started 2026-06-24. Companion to [default-lexer-roadmap](default-lexer-roadmap.md)
> Phase 2.1/2.2. Worked in the `json/lexers/default-lexer/lab` sandbox; promoted
> to production only at the final stage, behind the equivalence gate.

## Decision (locked 2026-06-24)

Unify L and VL from **one generic source**, road (a): a policy-parameterized
core with a concrete policy type per lexer. Accepted cost: **~5% on L** from the
per-token policy method call routing through the generics dictionary (Go does
not devirtualize type-param method calls; not fixable in a shipped library via
PGO). Rationale: buys VL's 3–8× speedup (it inherits L's fast paths) and deletes
~750 lines of L/VL duplication, from one source of truth. The generator road (b)
stays the escape hatch if the 5% ever bites on the real workload.

## Architecture

- **Generic push core** `scanPushG[T, P emitPolicy[T]]` (done for L, 2.1b):
  drives the scan loop, keeps the cursor in locals, emits via `p.emit`. The
  per-byte hot loop is policy-free (concrete `[]byte`/`int`).
- **Policies** (concrete, zero-size): `semanticPolicy` → `token.T` (identity
  emit); `verbatimPolicy` → `token.VT` (wrap T + blanks + position).
- **Non-inlined concrete shims** (`scanPushSemantic`, `scanPushVerbatim`) funnel
  the generic call so `Tokens()` stays inlinable and range-over-func keeps the
  yield closure on the stack (the +2-alloc fix from 2.1b).
- A **generic pull core** for `NextToken` comes in stage 2.
- `L`/`VL` become thin adapters that pick a policy.

## Stages (each gated by TestLabEquivalence + benchmark A/B)

- **Stage 1 — VL native push via the generic core.** 🚧
  Extend `emit` to `emit(t token.T, blanks []byte, line, col int) T`; add
  `token.T.AsVerbatim(blanks) VT` (zero-cost wrap, VT embeds T); add
  `verbatimPolicy`; track `blankStart` in `scanPushG` so the preceding
  whitespace run is sliced zero-copy as blanks; wire `VL.Tokens()` to a
  `//go:noinline scanPushVerbatim` shim. Gives VL a native push path it never
  had → the big VL speedup, and validates the policy abstraction on `token.VT`
  (emit does real work, not identity).
  - **DECISION FLAG (for Fred):** the push core uses L's folded/deferred-error
    semantics, not VL's look-ahead semantics. On VALID input the streams are
    identical (tokens + blanks + positions). On INVALID input, error code/position
    may differ from today's VL. This unifies error reporting across L and VL
    (arguably better/consistent), but it is a behavior change for VL errors.
    Stage-1 equivalence is asserted on valid-input streams; confirm the unified
    error semantics is acceptable before promoting.
- **Stage 2 — Generic pull core for `NextToken`.** Share one source between
  `L.NextToken` and `VL.NextToken`, removing the two big hand-written loops.
  Handle VL's look-ahead (or retire it in favor of the unified semantics from
  stage 1). Highest-line-count dedup.
- **Stage 3 — Single-source value scanners.** number / string / bool / null
  shared by both cores (already mostly shared; confirm and de-dup).
- **Stage 4 — Delete the dead duplicated loops.** lab is now the unified lexer.
- **Stage 5 — Promote lab → replace default-lexer.** The only irreversible step;
  separate review. Keep the package/API identical so downstream is untouched.

## Gates

- `TestLabEquivalence` green at every stage (valid-input streams; error-state
  parity except the documented stage-1 VL change).
- Benchmark A/B: no L regression beyond the accepted ~5%; VL must move decisively
  toward L (close most of the 3–8× gap).
- 0 steady-state allocs preserved.
