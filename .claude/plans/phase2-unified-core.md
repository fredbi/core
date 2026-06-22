# Phase 2 — unified lexer core (productionizing the push spike)

> Status: **DRAFT for review** — confirm the architecture fork before coding.
> Last updated: 2026-06-22

## Architecture decision (the fork)

The push spike (`push.go`, `P`) proved its speed via three control-flow-independent
techniques, not via push/yield itself:

1. cursor in a **local** during scanning (no per-byte `consumed`/`offset` struct writes),
2. **terminator folded** into the scan (no separate look-ahead pass),
3. **zero-copy** numbers and unescaped strings.

- **A (recommended): optimized *pull* core.** One `scanToken`: load cursor into a
  local at entry, scan one token, write back once per call. `NextToken` native-fast;
  `Tokens()` = thin `for{NextToken()}` wrapper. One loop (dedup), no coroutine,
  streaming fits the existing `readMore`. The per-byte tax was the pull *design's*,
  not pull's inherent — removable.
- **B: true push core + `iter.Pull` bridge.** Max `Tokens()` speed but `NextToken`
  (used by node/constrained/dynamic) pays coroutine overhead; streaming-in-push hard.

Decision: **A → refined to A+ (2026-06-22, evidence-driven).**

Stage-1 spikes settled it:
- **1a (zero-copy strings, committed):** big win on long values — strings_plain
  ~379→728, twitter ~318→490, citm ~458→498. Per-byte localization pays off for
  long values.
- **1b (zero-copy/local numbers, reverted):** ~0 gain (ints ~200→200). Short
  tokens (numbers, delimiters, keys) are not bottlenecked by per-byte struct
  writes but by **per-token / per-call overhead** (calls, `lookAhead`, the
  `current`/`next` stash, the elision re-entry) — inherent to pull-per-call.
- **`iter.Pull` bridge measured (committed evidence):** a push core consumed via
  `iter.Pull` is **~2× slower than the current pull lexer and allocates** (citm
  273 vs 512, 10 allocs). So **B is dead** — no cheap pull-from-push for `NextToken`.

The 935 MB/s spike ceiling is reachable only by a native push `Tokens()` loop, which
`NextToken` can never share. Since `NextToken` is the primary consumer API, the
unified core is **A+**: a single optimized pull `scanOne` shared by `L`/`VL` and by
`NextToken` + `Tokens()`, folding `lookAhead` / `current-next` / elision to push pull
as far as it goes (target ~650–750; the spike's 935 is partly its lighter validation,
so the real gap is smaller). A dedicated push `Tokens()` (hybrid "C") is deferred
until a high-throughput `Tokens()` consumer materializes (YAGNI).

Status: 1a landed. Next = build the A+ `scanOne` core (bytes first), gated as below.

## Invariants / gates (every stage)

- JSONTestSuite conformance stays **y_ 95/95, n_ 188/188**.
- Full default-lexer test suite stays green; token streams unchanged for L and VL.
- Benchmarks: **no regression** vs current; bytes-mode L target ≥ the spike
  (citm ~900+ MB/s) modulo features that legitimately cost (strict validation,
  line/col).
- Public API (`lexers.Lexer` / `VerbatimLexer`, options, pools) unchanged.

## Stages

1. **Bytes-mode unified core for `L`.** Rewrite `scanToken` around a local cursor
   (writeback once/call), fold the terminator (drop the separate `lookAhead`
   pass / `current`+`next` stash where possible), keep zero-copy numbers, add
   zero-copy unescaped strings. Keep full grammar validation + all existing
   behavior (elision, errors). Gate: conformance + tests + bench vs spike.
2. **Streaming mode.** Fold `readMore` into the new loop: cursor local within a
   buffer span, writeback + refill at boundaries; zero-copy disabled when streaming
   (copy into `currentValue`, flush-on-refill). Verify the `L/reader` conformance
   modes + buffer-crossing tests.
3. **Features back in, re-measured.** Line/column tracking, `maxValueBytes` /
   `maxContainerStack` guards, `WithElideSeparator`. Re-run benchmarks; record the
   headroom these cost.
4. **`VL` on the shared core.** Verbatim as a thin policy layer over the same scan:
   blanks buffer, `token.VT`, line/col in the token, surrogate-pair escapes, the
   "verbatim doesn't alter strings" cleanup. Zero-copy strings for VL too.
5. **Pooling / `Reset`.** Ensure the new core recycles cleanly (0 allocs steady
   state) for both bytes and reader; Borrow/Redeem.
6. **Migrate consumers.** Move `json/nodes/light`, `json/constrained`,
   `json/dynamic` to the elided token model (built centrally in `json/options.go`
   and `json/dynamic/options.go`); flip them off `WithElideSeparator(false)`.
7. **Remove the duplication.** Delete the prototype `P` (folded into the core) and
   the old duplicated `L`/`VL` scan code; keep `push_test.go`-style equivalence
   checks as regression tests.

## Risks

- Folding the look-ahead (terminator) into the main loop while preserving the exact
  error semantics (trailing/repeated comma, value-after-value, etc.) is the most
  delicate part — the conformance suite is the gate.
- Streaming + zero-copy interaction (values must be copied when they may cross a
  refill, including via the trailing terminator scan) — already understood from 1.4.
- VL's blanks + positions across refills.
