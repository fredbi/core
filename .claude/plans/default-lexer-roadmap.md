# default-lexer roadmap

> Status: **active** — Phases 0 & 1 complete; first R&D pass complete; L/VL
> unification is the next major stream (paused by request 2026-06-24).
> Scope: `json/lexers/default-lexer` (+ shared `json/lexers`, `json/lexers/token`, `json/lexers/error-codes`).
> Last updated: 2026-06-24
>
> **Companion research** (the "why" behind the decisions below) lives in
> `.claude/plans/ramblings/`. Start with `2026-06-conclusions-fast-go-lexing.md`
> (distilled, transferable). Detail: `2026-06-perf-and-paradigm.md` (pull-vs-push,
> SWAR, standings, L-vs-VL baseline), `2026-06-codegen-asm-jit.md` (the four
> roads to speed; asm/JIT/SIMD rejected for the lexer), `2026-06-go-openapi-v2-context.md`
> (why this block is shaped as it is). See also memory `go-openapi-v2-programme`.

## Legend

- ✅ done
- 🚧 in progress
- ⏳ planned / not started
- 🔬 needs design decision (see "Open design decisions")
- 💭 stretch / far-out
- ❌ dropped / out of scope

---

## 1. Vision & differentiators

A low-level JSON building block that **lexes without evaluating**. Concretely:

1. **No value evaluation.** Numbers and strings stay as `[]byte`; no numeric
   conversion, so no loss of resolution. Comparable in spirit to the experimental
   `encoding/json/v2` + `encoding/json/jsontext`.
2. **Near-zero allocation, low peak memory.** Everything lives in short-lived,
   recycled buffers. The only hard-coded literals are `null`/`true`/`false`.
   Peak memory ≈ longest single string/number value.
3. **Pluggable behind a common interface** (`json/lexers`), so alternative
   implementations (incl. one on top of `encoding/json/v2`, and a SIMD one) drop in.
4. **Two flavors:**
   - **Semantic** (`L` → `token.T`): drops insignificant whitespace, normalizes.
   - **Verbatim** (`VL` → `token.VT`): preserves whitespace/escapes; for
     linters / LSPs / formatters sensitive to exact positions.
5. **Streaming or buffered:** `io.Reader` (internally buffered) or `[]byte`.

The current alteration applied to input is UTF-8 normalization of `\u` escapes
(canonicalization of runes). Goal: make this **optional**.

**Yardstick (settled 2026-06-24):** we race against **jsontext** (`encoding/json/v2`'s
tokenizer) — the only pure-Go, fully-validating, non-converting peer. Not
easyjson (brackets us: unvalidated `Raw` vs lossy `Float64`), not the decoders
(goccy/sonic unmarshal into Go types — no extractable token stream), not SIMD
(different category). If jsontext leaves experimental, **wrapping it** for the
non-verbatim path is an acceptable end-state.

---

## 2. Current state (baseline, 2026-06-21)

> Historical snapshot. The ⚠️/🐛 items below are **all since resolved** (Phase 0
> conformance work, the benchmark module, pool tests, the nit fixes); kept for the
> record. Current status is the phase list + §3b.

- ✅ Builds clean, `go vet` clean, `go test` green. Coverage **65.5%**.
- ✅ Core grammar enforcement (delimiters, key/object/array context, RFC 8259
  numbers, surrogate pairs), nesting stack with bit-packed depth, optional
  circuit breakers (`WithMaxContainerStack`, `WithMaxValueBytes`), pooling,
  rich error context.
- ⚠️ `VL` largely duplicates `L` (~750 lines parallel to ~750).
- ⚠️ `benchmarks/comparative/` is an empty `go.mod`; `examples/` likewise empty.
- ⚠️ No conformance suite wired in; coverage gaps (pools untested).
- 🐛 Minor: `ErrRepeatedDecimalSeparator` message wrong; `stackScale` comment
  (`2^8-1` → `2^6-1`); options doc says uint64 = "4 bytes".

---

## 3. Guiding principles for sequencing

1. **Build the safety net before remodeling.** Conformance suite + benchmark
   baseline come first, so the big refactor can't silently regress correctness
   or performance.
2. **Settle the interface/API surface before the L/VL refactor**, so we design
   the unified core once, already aware of what it must support (iterator,
   elide-separator, line/col, optional normalization).
3. **Features that risk the core invariants (no-eval, no-alloc) are opt-in**
   and explicitly flagged — notably RFC 8785 number canonicalization.

---

## 3a. Decisions taken (2026-06-21)

- **Conformance suite (0.1):** ✅ **vendor a copy** into
  `testdata/JSONTestSuite/` (rev `1ef36fa`, MIT license + SOURCE.md included).
- **First execution step:** ✅ **conformance harness (0.1)** — done; see baseline below.

## 3b. Decisions & re-orientations (2026-06-22 → 06-24)

The R&D pass since the roadmap was first drafted landed several decisions that
supersede or reframe items below. Recorded here so the phase list stays honest.

1. **Push `Tokens()` shipped as a duplicated main loop, not yet the unified core.**
   The 2.0 spike (push prototype `P`) graduated into a native `scanPush` backing
   `Tokens()` in whole-buffer mode — proven token+error identical to `NextToken`
   over all 318 fixtures. But it is a **second** hand-written main loop, not the
   "thin adapters over one shared core" of 2.1. So Phase 2's *dedup* goal is still
   open; we bought the iterator speedup first. (push `Tokens()` is parity-or-better
   with jsontext on 4/5 workloads.)
2. **Number + string fast paths landed in `L`.** Inlinable simple-number fast
   path (ints ~+175%), digit-run number scanner with `uint()`-BCE (floats/canada
   ~+100%), SWAR multi-needle string scan (zero-copy alias, win on real-world
   docs). Lesson: **short tokens are bottlenecked by per-token overhead, not
   per-byte work.** Details in the perf ramble.
3. **The "race of inlining" is over — measured ~3–7% ceiling.** Force-inline
   (via PGO; there is no `//go:inline`) buys only +2–3% typical / +7% on the most
   token-dense workload. Our modular code is already within 3–7% of fully-inlined.
   So codegen-for-speed is **not** justified.
4. **Scalar asm and JIT rejected for the lexer (on evidence); SIMD deferred.**
   The compiler already register-allocates the leaf scanners; an asm/JIT kernel
   breaks the per-token ABI boundary or has no type to specialize. SIMD collides
   with streaming/low-memory goals and changes our competitive category. All three
   stay in the back pocket for **higher layers** (typed decoder, schema-validator
   OP-program) or a **separate optional whole-buffer `simd-lexer`** — never woven
   into default-lexer. See codegen/asm/jit ramble.
5. **Pool hardening (`55f112c`, `ec43cdb`, `1de7d74`).** Three steps, since
   **pools are used systematically by our callers** (load-bearing):
   - `Reset()` fixed: drops the aliased caller buffer + reader (no pinning/leak of
     user specs) and no longer allocates on bytes-borrows.
   - `VL` gained `ResetWithBytes`/`ResetWithReader` — VL is now reuse-safe with 0
     steady-state allocs.
   - L pool moved to the **redeemable, leak-checkable** type
     (`pools.NewRedeemable[L]`). **Breaking API:** `BorrowLexerWith{Bytes,Reader}`
     now return `(*L, func())` (cached redeemer, no per-borrow closure alloc);
     `RedeemLexer` removed. Under `-tags poolsdebug` the pool detects
     double-/foreign-redeem and leaks; tests assert `pools.AssertNoLeaks` via
     `t.Cleanup` and run green in both build modes. Exported `pools.DebugBuild`
     for both-mode tests. **VL pooling deferred to the L/VL unification.**
6. **L/VL unification is now the priority lever — and it's a perf play too.**
   First recorded L-vs-VL baseline (06-24): **VL is 3–8× slower than L**, because
   VL never received any of L's fast paths (still the old byte-by-byte loop +
   per-token look-ahead). Generating/parameterizing both from one optimized source
   would collapse that gap to the genuine cost of fidelity (~1.2–1.5× est.) *and*
   kill the ~750-line duplication. Two candidate roads: **generics with a concrete
   policy type** vs a **vendored `refactor/inline` generator** — to be spiked
   head-to-head (Phase 2.1). *(Stream paused by request 2026-06-24.)*
7. **Canonicalization is OUT of the lexer's scope (dropped 2026-06-24).** RFC 8785
   JSON canonicalization is about producing a stable byte form for **signing**:
   it mandates `float64` numbers (violates our no-resolution-loss invariant) and
   **sorted keys** (we deliberately preserve original key order). That is a
   different job from lexing. If we ever need it, it is a **separate component
   built on top of** the lexer, not in it. (Fred's original narrower idea — number
   "canonicalization" as the shortest correct decimal form — is also dropped from
   the lexer for the same reason: the lexer does not evaluate numbers.) Phase 3.2
   is therefore removed.
8. **Streaming on all fast paths is still owed.** Native push `Tokens()` is
   whole-buffer only today; streams fall back to the `NextToken` loop. Functionally
   complete, but the fast push path doesn't yet cover streams (incl. the
   sliding-window-reload cost question). Folded into the unification work so we
   don't hand-write a third main loop.

## 4. Phased roadmap

### Phase 0 — Baseline & safety nets  🚧

Protect and measure before changing anything.

- ✅ **0.1 Conformance harness.** Vendored JSONTestSuite (318 `test_parsing`
  + 22 `test_transform`). `conformance_test.go` drains every case through 3
  modes — `L`/bytes, `L`/reader (64B buffer to stress buffer-crossing),
  `VL`/bytes — asserting `y_` accept / `n_` reject, recording `i_` behavior,
  with an xfail set keeping the suite green while the backlog is worked.
  **Baseline: y_ 95/95 pass (no false-rejects), n_ 172/188 (16 false-accepts).**

- ✅ **0.4/0.5 conformance fixes (groups A–E) + deep-nesting bug (step F).**
  All 16 false-accepts fixed and the severe nesting bug resolved; the parsing
  suite is now **100% conformant (y_ 95/95, n_ 188/188, xfail 0)**. Committed as
  groups A (numbers), B (control chars), C (value-less comma), D (EOF/empty
  paths), E (VL `\u` validation), F (stack rewrite + `IndentLevel` fix, with
  `stack_test.go` covering deep nesting & the `maxContainerStack` breaker).
  New error codes: `ErrControlChar`, `ErrMissingValue`, `ErrNoData`.

#### 0.1 baseline findings (16 false-accepts + extras) — ✅ all resolved

| # | Group | Files | Root cause | Affects |
|---|-------|-------|------------|---------|
| A | Number grammar | `0.e1`, `2.e3`, `2.e+3`, `2.e-3` | `.` with no fractional digit before exponent: `isFractional` cleared on `e`, so the `fractionalPart==0` check is skipped. Fix: check `hasFractional && fractionalPart==0`. | L+VL |
| A | Number grammar | `invalid+-` (`0e+-1`) | double sign in exponent allowed (sign gate only checks `exponentPart>0`). | L+VL |
| B | Strings | `unescaped_ctrl_char`, `unescaped_newline`, `unescaped_tab` | raw control chars U+0000–U+001F accepted; RFC 8259 forbids unescaped. Fix: reject `b < 0x20` in `consumeString`. | L+VL |
| C | Commas | `array_comma_and_number` (`[,1]`), `array_missing_value` (`[ , ""]`) | comma accepted with no preceding value (after an opening delimiter). | L+VL |
| D | EOF paths | `structure_unclosed_array` (`[1`), `array_unclosed_with_new_lines`, `object_no-colon` (`{"a"`) | `lookAhead` / `expectColon` return `EOFToken` **without** erroring when still inside a container / awaiting colon. | L+VL |
| D | Empty doc | `structure_no_data` (empty), `single_space` | blank-only / empty input accepted (no top-level value). | L+VL |
| E | VL escape validation | `string_invalid_unicode_escape` (`\uqqqq`) | verbatim string scan does not validate `\u` escapes (L does). | VL only |

**Extra findings (not counted in the 16):**

- 🐛 **Deep-nesting stack bug (real correctness, severe).** Balanced nesting
  **> 63 levels** is rejected with "mismatched ]" (boundary exactly at the
  `uint64` word in the bit-packed stack). The multi-word overflow path in
  `pushArray`/`pushObject` writes the wrong sentinel to the new word, so
  `popContainer` unwinds prematurely. Surfaced via `i_structure_500_nested_arrays`
  and reproduced at depth 64. *Valid JSON wrongly rejected* — should be fixed
  regardless of any configured stack cap.
- **L/VL surrogate divergence.** Many `i_string_*surrogate*` cases: L rejects
  lone/invalid surrogates, VL accepts (same un-validated-escape root as E).
  Implementation-defined, but the divergence underlines the dedup priority.
- **Documented good behavior (the differentiator at work):** huge
  exponents / overflowing integers (`i_number_huge_exp`, `i_number_real_pos_overflow`,
  …) are all **accepted** — we don't evaluate numbers, so no overflow. UTF-16/BOM
  inputs are rejected (UTF-8-only, as documented).

- ✅ **0.2 Benchmark harness + baseline.** New module `json/benchmarks/` (own
  go.mod, in go.work) isolates heavy deps. `lexers/` compares implementations via
  the `lexers.Lexer` interface: `default-lexer` (bytes / pooled / verbatim) vs a
  `stdlib` baseline on `encoding/json` v1 (UseNumber). 11 synthetic workloads
  (`workloads/`) + 4 vendored real-world datasets (canada/citm/twitter/golang,
  gzip-embedded, BSD). `TestWorkloadsLex` guards that all inputs lex cleanly.
  **Baseline: default-lexer ~5–10× stdlib throughput with flat single-digit
  allocs/op (1 when pooled) vs stdlib's 10^5–10^6.** (jsontext / easyjson / jsonv2
  baselines deferred to Phase 4.)
- ✅ **0.3 Security-by-default audit.** Stance: total input bounded by the caller
  via `io.LimitReader`; lexer keeps two orthogonal breakers (depth, per-value
  memory); guards off by default with a documented hardening recipe. Fixed two
  bugs: VL's shadow `options` struct made its value cap dead (removed), and the
  verbatim blanks buffer was unbounded (now under `WithMaxValueBytes`). Added
  `security_test.go`. No `WithStreamDefaults` bundle (avoided magic numbers).
  <details><summary>original 0.3 notes</summary>

  Review guards against hostile streams:
  unbounded nesting, unbounded value size, buffer growth. Decide **safe defaults
  for streaming mode** (today both breakers default to 0 = unlimited). 🔬
  </details>
- ✅ **0.4 seriot.ch pitfalls pass.** Effectively covered by the JSONTestSuite
  conformance work (groups A–E + step F); the suite encodes the seriot cases.
- ✅ **0.5 Fix known nits.** `stackScale` comment & `IndentLevel` (step F), pools
  path exercised (benchmarks + tests), `ErrRepeatedDecimalSeparator` message and
  `options.go` "4 bytes → 8 bytes" doc all fixed.

### Phase 1 — Interface & API surface  ✅

Additive, low-risk; done before the refactor so the core targets the final shape.

- ✅ **1.1 Iterator API.** Added `Tokens() iter.Seq[token.T]` (and
  `iter.Seq[token.VT]`) to the interfaces, implemented on `L`/`VL`/stdlib baseline.
  EOF ends the range; errors via `Err()` after the loop. **Wrapper impl measures
  identical to the manual loop (free ergonomics, no speedup).** A faster *native
  push* iterator is intentionally deferred to Phase 2: the push core (scan loop
  that yields directly, no `current`/`next` stash) becomes the shared foundation
  for both `NextToken` and `Tokens()` — built once, not duplicated.
- ✅ **1.2 `WithElideSeparator` option.** Default-ON for `L`: elides **both `,`
  and `:`** (jsontext parity; `Key` token makes `:` redundant). `scanToken` still
  produces/validates them (context checks intact); `NextToken` filters; `Tokens()`
  inherits it. `VL` ignores the option (always preserves all tokens). Existing
  tests opt out via the `getLexer` helpers; new `elide_test.go` covers the default.
  ⚠️ **Migration debt:** `json/nodes/light`, `json/constrained`, `json/dynamic`
  still rely on separators and break under the new default — they construct lexers
  centrally in `json/options.go` and `json/dynamic/options.go`. Migrate to the
  elided model (the "revisit node.go deeply" work) — until then their tests fail.
- ✅ **1.3 Line/column tracking.** Always-on for both (1-based); negligible cost
  (one increment per newline + a token-start snapshot) — semantic `L` benchmarks
  unchanged. `token.VT` carries `line`/`col` (`Line()`/`Col()`/`WithPosition`);
  `token.T` stays lean and `L`/`VL` expose `Line()`/`Column()` methods. Kept off
  the `Lexer` interface (stdlib baseline can't provide positions; an optional
  `PositionedLexer` could come later). Tests in `position_test.go` (incl. streaming
  buffer-crossing + CRLF).
- ✅ **1.4 Zero-copy values (numbers).** In whole-buffer mode (`wholeBuffer` flag)
  a number's value aliases the input (`buffer[start:end:end]`, cap==len) instead
  of copying into `currentValue`. `consumeNumber` is branch-free (validate, no
  per-byte append; alias or single bulk-copy at the end; flush-on-refill for
  streaming). **Measured +23% (ints) / +31% (canada) throughput, fewer allocs.**
  Key lesson: the constraint is *buffer stability*, not caller ownership — bytes
  mode is safe (noopReader never overwrites; look-ahead's readMore is a no-op),
  streaming is not (refill, incl. during look-ahead). **Strings stay on the copy
  path**: a lazy-copy-on-escape variant added hot-path cost + an escaped-string
  regression for no gain → deferred to Phase 4 with a fast happy-path scanner
  (IndexByte/SIMD). Streaming zero-copy + a rotating-buffer knob also Phase 4+.

### Phase 2 — Consolidation: de-duplicate L / VL  🚧

Risky remodel, executed with Phase 0 net in place and Phase 1 shape known.

#### Profiling insights (2026-06-22, `lexers/profile_test.go`) — inform the design

- **Allocations are already optimal — don't chase them.** The 6/8 allocs/op in the
  headline benchmark are pure measurement bias (a fresh lexer per iteration:
  `new(L)`, nesting-stack word, `currentValue` growth, +4KB buffer for readers).
  **Pooled/recycled, scanning is 0 allocs/op.**
- **CPU is dominated by `scanToken` (the main loop): ~55% flat / 92% cum** (citm,
  pooled bytes). Then `consumeString` 17%, `consumeNumber` 10%, `readMore` 4%.
- **Hottest lines are per-byte bookkeeping, not logic:** `offset++`/`consumed++`,
  the `for consumed < bufferized` condition, and the blank/`lineFeed` switch
  dispatch. ⇒ **The Phase 2 push-core's main lever:** scan with `consumed`/`offset`
  (and line state) in **locals**, writing back to the `L` struct only at token
  boundaries — removes per-byte struct writes the pull model can't avoid.
- **Inlining:** small predicates/stack helpers inline (`isInObject`, `push*`,
  `popContainer`, `depth`, `unhex`); the big scan funcs don't (too complex —
  expected). Near-miss: `push` cost 82 vs budget 80 — trivial trim fully inlines
  the container-push path.

#### easyjson comparison — the headroom target (2026-06-22)

Added `mailru/easyjson/jlexer` (the pull []byte lexer that inspired this one) as a
benchmark point (`lexers/easyjson/`), numbers taken raw (no conversion). **easyjson
is faster on every workload: ~2.5–2.7× on number-heavy input (ints/canada), ~1.3×
on real docs (citm/twitter).** ⚠️ **Caveat (resolved later):** easyjson's `Raw()` does NOT validate numbers — it
only validates on conversion (`Float64`). So the "2.5× on numbers" compared our
always-validating lexer to easyjson's *unvalidated* scan. Rebalanced with
`Float64` (`easyjson-f64`, which validates but is lossy), the push prototype is
**~1.8–2× faster than easyjson on numbers AND lossless** (see 2.0 spike). The real
lesson stands: the per-byte main-loop cost was the tax on the *pull* design.
**Phase 2 target: push-core with zero-alloc strings + zero-copy numbers → beat
easyjson on both speed and allocations** (the spike confirms this on every workload).

- ✅ **2.0 Push-core spike (prototype `P`, `push.go`).** Validated the design:
  self-driving scan loop yielding via range-over-func, cursor+state in **locals**
  (no per-byte struct writes), terminator folded in (no look-ahead pass), full
  number validation, zero-copy numbers **and** unescaped strings. Token stream
  proven identical to `L`. **Results (MB/s): push ~1.8–2.9× the current pull `L`
  everywhere, and beats easyjson on real/string docs** (citm 963 vs 576, twitter
  920 vs 419, strings 1110 vs 403) thanks to zero-copy strings + 0 allocs; trails
  easyjson only on pure-number input (~350 vs ~530 — our strict number validation
  vs easyjson's deferred). Confirms: build the unified core push-first, and
  zero-copy strings belong in it (they couldn't pay off in the pull design).
  Caveats to close when productionizing: bytes+streaming, full structural error
  detection/conformance, line/col, VL blanks, surrogate pairs, pooling.
- 🚧 **2.0b Native push `Tokens()` landed (`0858b3f`+).** The spike graduated:
  `scanPush` backs `Tokens()` (whole-buffer fast path), carrying the inline-int
  fast path and reusing the value scanners. Token+error stream proven identical
  to `NextToken` over all 318 fixtures. **But it is a second hand-written main
  loop** — the iterator speedup was bought ahead of the dedup. The dedup (2.1/2.2)
  is still open, and now also owes a *streaming* push path (see 3b.8).
- ✅ **2.1a Sandbox stood up (`23a1649`).** Package
  `json/lexers/default-lexer/lab` is a verbatim copy of the lexer (package
  `lexer` → `lab`; `lab.L` is "L2"), kept side by side with the reference so the
  unification can be spiked without risk. `lab/equivalence_test.go` asserts
  behavioral parity vs the reference over all 318 fixtures in four modes (bytes
  NextToken, streaming NextToken@64B, whole-buffer `Tokens()`, VL) — the instant
  regression signal as lab diverges. Benchmarks expose `lab/bytes` + `lab/tokens`
  next to `default-lexer/*` for direct A/B. Starts == reference within noise.
- 🔬 **2.1b Generics spike — done, measured (`49e62e4`).** Policy-parameterized
  push core `scanPushG[T, P]` in lab, `semanticPolicy` (identity emit) backing
  L's `Tokens()`; streams identical (equivalence green). Findings:
  - **Stenciling is NOT the cost** (answers the struct-stenciling worry): the
    token struct gets its own GC-shape stencil, the per-byte loop is concrete and
    byte-identical; with a plain-func driver the generic core is **0 allocs, at
    parity** (216 vs 218 ns).
  - **range-over-func + generics across packages = +2 allocs/call** (the iter.Seq
    yield closure heap-allocates when its body holds a generic call). **Fixed**
    by funnelling the generic call through a `//go:noinline` concrete shim so the
    Seq body keeps the reference's cross-package "yield does not escape" summary →
    back to 0 allocs.
  - **Residual ~5% per-token cost**: a method call on a type-param value
    (`p.emit`) routes through the **generics dictionary (indirect call)** even
    for a concrete zero-size policy — Go does not devirtualize it. Controlled
    reuse A/B: citm/whitespace +5%, twitter +4%. Within the ~3–7% ceiling band.
  - **Verdict:** generics viable (correct, 0-alloc) at ~5% L cost from per-token
    dict dispatch. **DECIDED 2026-06-24 (Fred): accept the ~5%, land it with
    generics** (road a). Generator (road b) stays the escape hatch. Execution is
    tracked in [unified-lexer-plan.md](unified-lexer-plan.md).
- 🚧 **2.1 Unify L/VL from one generic source — road (a) chosen; STAGES 1–4 DONE.**
  Detailed stage tracker: [unified-lexer-plan.md](unified-lexer-plan.md). Worked
  in the `lab` sandbox; from 2026-06-25 in worktree `.worktrees/lexer/exploration`
  (branch `exploration`); promoted to production only at stage 5.
  - ✅ **Stage 1 (`3f533ad`)**: generic push core `scanPushG[T,P]` serves L and VL;
    `VL.Tokens()` gets a native push path → **VL push ≈ 2.0–2.4× VL pull**
    (citm 177→361, twitter 155→333, ints 66→156 MB/s), 0 allocs. Unified VL also
    **fixes a reference-VL `\u`-decode bug** and validates surrogates like L
    (both behavior changes signed off).
  - ✅ **Stage 2 (`5ec002d`)**: generic pull core `scanTokenG[T,P]` (+`errCheckG`)
    backs `L.NextToken` and `VL.NextToken`; VL look-ahead retired (legacy kept
    dead). **The 4 hand loops → 2 generic cores.** L still == reference (bytes +
    streaming), 0 allocs, ~5–7% slower (accepted). VL pull == VL push on all 318
    fixtures.
  - ✅ **Stages 3–4 (`47b2aa7`)**: value scanners confirmed single-source (both
    cores call L's); deleted all dead legacy loops + the vestigial look-ahead
    state (VL.{next,nextBlanks,current}, L.{nextLine,nextCol,lastStack}).
    **Net −1885/+110** — bigger than the ~750 estimate because it removed the
    whole legacy loops (scanToken, scanPush, VL's bespoke impl), not just the
    value-scanner overlap. VL is now a thin policy adapter over `*L`; IndentLevel
    is `depth()` for both. New `TestIndentLevelEquivalence` gate (lab L == ref L,
    lab VL == lab L non-eliding, all fixtures).
  - ⏳ **Stage 5 remaining — RESUME HERE**: promote `lab` → replace
    `default-lexer` (irreversible; separate review). Fold in the inherited lint
    cleanup (dogsled / gochecknoglobals / gocyclo / `NeVerbatimWithBytes` godoc
    typo / embedded-field order — all pre-existing from the verbatim copy). Keep
    the `P`/`NewPush` push prototype (mirrors production `deflex.NewPush`).
  - Historical context (the head-to-head framing that led to choosing road a):
- 🔬 **2.1-orig Unify L/VL — the two roads weighed (road a chosen).**
  Reframed by the R&D pass: this is no longer only a maintainability play. The
  L-vs-VL baseline shows **VL is 3–8× slower than L purely from missing fast
  paths** (3b.6), so unification *also* makes VL fast and gives streaming push for
  free (3b.8). The force-inline ceiling (~3–7%, 3b.3) means the choice is decided
  on **maintainability + VL speed**, not L speed. Two roads to spike head-to-head:
  - **(a) Generics with a concrete policy type param** (not interface →
    monomorphizes + devirtualizes): one generic core, two instantiations (L emits
    `token.T` / drops blanks; VL emits `token.VT` / tracks blanks + positions). No
    vendored toolchain. May not beat the inline budget but likely unifies "well
    enough."
  - **(b) Vendored `golang.org/x/tools/internal/refactor/inline` generator**:
    write one readable, local-cursor "golden source"; mechanically flatten/inline
    into the artifact. Single source of truth; can ignore the 80-cost inline
    budget. Heavier machinery; the golden source must be written in
    local-cursor/threaded-state style or flattening yields flat-but-slow code.
  Spike one value scanner (e.g. the number path) under each, compare real code +
  numbers, then commit. Gate: conformance- and benchmark-neutral for L; VL must
  improve and stay conformant.
- ⏳ **2.2 Migrate** `L` then `VL` onto the chosen shared core; native push for
  **both bytes and streams**; keep all tests green (Phase 0 suite is the gate).
  Re-run benchmarks; no L regression, VL must close most of the 3–8× gap.

### Phase 3 — Semantic features  ⏳

- ⏳ **3.1 Optional input normalization.** Make UTF-8 / escape processing
  toggleable (sanitizer hooks for strings and numbers, per old TODO).
- ❌ **3.2 JSON canonicalization (RFC 8785) — DROPPED (2026-06-24).** Out of the
  lexer's scope. RFC 8785 exists to produce a stable byte form for **signing**:
  it mandates `float64` numbers (breaks our no-resolution-loss invariant) and
  **sorted keys** (we deliberately preserve original order). Different job from
  lexing. If ever needed, build it as a **separate component on top of** the lexer.
  (Even the narrower "shortest-decimal number form" idea is out: the lexer does
  not evaluate numbers.) See 3b.7.
- ⏳ **3.3 NDJSON.** Line-delimited JSON, especially for streaming `io.Reader`;
  top-level value sequence separated by `\n`.

### Phase 4 — Performance  ⏳

- ⏳ **4.1 Reduce memcopy.** Hunt avoidable copies (`currentValue` appends,
  `consumeN`, buffer-overturn copies); apply zero-copy from 1.4 where owned.
  Remaining known L gap: **pure long strings (strings_plain ~82% of jsontext)** —
  needs a faster unescape slow path and/or SWAR-ing the slow-path clean runs.
- ✅ **4.2 Inlining pass — measured, ceiling found.** Force-inline buys only
  ~3–7% (3b.3); modular code is already within that of fully-inlined. No further
  inlining race. (Full PGO methodology in the codegen ramble.)
- ✅ **4.3 Full comparative benchmarks.** `jsontext` (encoding/json/v2 — the
  yardstick) and `mailru/easyjson` (raw + Float64) wired in; goccy/sonic evaluated
  and excluded (decoders, no extractable lexer); stdlib v1 baseline kept. Standings
  recorded in the perf ramble. Remaining: not chasing more competitors.
- 💭 **4.4 SIMD variant — deferred, runtime-usage only.** A **separate optional
  whole-buffer `simd-lexer`** behind the `lexers.Lexer` interface (NOT woven into
  default-lexer); realistic only via `GOEXPERIMENT=simd`. Per the v2 programme it
  matters only for the untyped-Document runtime, not the spec-gobbling core. The
  free 80/20 (`bytes.IndexByte`, already SIMD asm) is in scope where it fits. 🔬

---

## 5. Open design decisions

Resolved (kept for the record):

1. ✅ **Conformance suite (0.1):** vendored a copy (`testdata/JSONTestSuite/`).
2. ✅ **Streaming security defaults (0.3):** guards off by default; total input
   bounded by the caller via `io.LimitReader`; two orthogonal breakers (depth,
   per-value memory) with a documented hardening recipe. No magic-number bundle.
3. ✅ **Iterator signature (1.1):** `iter.Seq[token.T]` + post-loop `Err()`.
4. ✅ **ElideSeparator default (1.2):** default-on for `L` (elides `,` and `:`);
   `VL` always preserves. (Downstream migration debt tracked in 1.2.)
5. ✅ **Line/col scope (1.3):** always-on for both; on `token.VT` + `Line()`/
   `Column()` methods; kept off the `Lexer` interface.
6. ✅ **Zero-copy model (1.4):** `T.Value()` aliases the source (`buf[s:e:e]`)
   only in whole-buffer mode; constraint is buffer stability, not ownership.
7. ✅ **Numbers/canonicalization (3.2):** dropped — out of lexer scope (3b.7).

Still open:

8. 🔬 **L/VL unification road (2.1):** generics-with-concrete-policy-type vs
   vendored `refactor/inline` generator. Decide by head-to-head spike — on
   maintainability + VL speed, not L speed (which is at its ~3–7% ceiling).
9. 🔬 **SIMD packaging (4.4):** separate module + build tags; pure-Go fallback;
   relationship to `GOEXPERIMENT=simd`. Runtime-usage only; far-out.

---

## 6. Next step (when the unification stream resumes)

**Phase 2.1 spike:** prototype one value scanner (the number path) under both
unification roads — (a) generics with a concrete policy type, (b) vendored
`refactor/inline` generator — and compare real code + benchmarks before
committing. Deliver as a reviewable plan doc first (Fred's preferred rhythm),
then validate with the spike. Success = same/better L throughput, a much faster
VL (close most of the 3–8× gap), and the ~750-line L/VL duplication gone, with
native push covering **both** bytes and streams.

*(Stream paused 2026-06-24 by request; pick up here.)*
