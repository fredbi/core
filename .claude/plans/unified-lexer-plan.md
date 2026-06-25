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

- **Stage 1 — VL native push via the generic core.** ✅ (`3f533ad`)
  Done. The generic core serves L and VL; VL.Tokens() is a native push path.
  **Result (reuse, 0 allocs): VL push ≈ 2.0–2.4× VL pull** (citm 177→361, twitter
  155→333, ints 66→156 MB/s) — roughly halves the L-vs-VL gap; residual is
  inherent (VL emits separators L elides + larger VT + dict emit). Gate
  `TestLabVerbatimPushEquivalence` green on all 95 must-accept fixtures against
  the unified-contract oracle (L's decoded values + VL's blanks/positions).
  **Behavior changes (need sign-off):** unified VL now (1) decodes `\u` escapes
  correctly — fixes a reference-VL `\u` bug — and (2) validates `\u` surrogates
  like L. Both are improvements; both change VL output on those inputs.
  ORIGINAL plan ↓
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
- **Stage 2 — Generic pull core for `NextToken`.** ✅
  `scanTokenG[T, P]` + `errCheckG[T, P]` now back both `L.NextToken` and
  `VL.NextToken` (policy adds `none()`/`eof()`). VL's look-ahead is retired (the
  legacy loop is kept as `nextTokenLegacy`, dead, for stage-4 deletion). Shared
  blanks state moved onto `L` (`blanks`/`trackBlanks`); the pull core accumulates
  blanks byte-by-byte so they survive streaming refills (verified at buffer
  size 8). Gates: L NextToken (bytes + streaming@64B) still == reference L, 0
  allocs, ~5–7% slower (accepted dict-emit cost); `TestLabVerbatimPullMatchesPush`
  asserts VL pull == VL push on all fixtures (push already validated against the
  unified contract). Net: 4 hand-written loops → 2 generic cores (pull + push),
  each serving L and VL.
- **Stage 3 — Single-source value scanners.** ✅ (`47b2aa7`, worktree
  `.worktrees/lexer/exploration`, branch `exploration`)
  Confirmed: number / string / bool / null were already single-source — both
  cores call L's `consumeString` / `consumeNumberWhole` / `consumeNumberStreaming`
  / `consumeBoolean` / `consumeNull`. VL's copies existed only to feed its
  look-ahead. No de-dup work needed beyond deleting those copies in stage 4.
- **Stage 4 — Delete the dead duplicated loops.** ✅ (`47b2aa7`)
  Removed L.scanToken, L.errCheck, push_tokens.go (L.scanPush), VL.nextTokenLegacy
  + VL's 7 look-ahead value scanners, and `indexStringSpecial` (only scanPush
  called it). Also removed the now-vestigial look-ahead state: VL.{next,
  nextBlanks,current} and L.{nextLine,nextCol,lastStack}. With no look-ahead,
  IndentLevel collapses to depth() for both lexers and VL is a thin policy adapter
  over an embedded *L. **Net -1885/+110; four hand loops → two generic cores.**
  New gate `TestIndentLevelEquivalence` proves lab L IndentLevel == reference L
  and lab VL == lab L (non-eliding) on every fixture (the stream tests didn't
  cover IndentLevel). go vet clean; TestWorkloadsLex green under -tags poolsdebug.
  Inherited lint (dogsled, gochecknoglobals, gocyclo, the `NeVerbatimWithBytes`
  godoc typo, embedded-field order) is pre-existing from the verbatim copy →
  deferred to the stage-5 lint pass.
- **Stage 5 — Promote lab → replace default-lexer.** ⏳ RESUME HERE. The only
  irreversible step; separate review. Keep the package/API identical so
  downstream is untouched. Fold in the inherited lint cleanup. Note the standalone
  `P`/`NewPush` push prototype (push.go) is kept as-is — it mirrors production
  `deflex.NewPush` (used by the benchmark suite), not part of the L/VL dedup.

## Gates

- `TestLabEquivalence` green at every stage (valid-input streams; error-state
  parity except the documented stage-1 VL change).
- Benchmark A/B: no L regression beyond the accepted ~5%; VL must move decisively
  toward L (close most of the 3–8× gap).
- 0 steady-state allocs preserved.
