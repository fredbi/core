# default-lexer roadmap

> Status: **DRAFT for review** — iterate before committing to execution.
> Scope: `json/lexers/default-lexer` (+ shared `json/lexers`, `json/lexers/token`, `json/lexers/error-codes`).
> Last updated: 2026-06-21

## Legend

- ✅ done
- 🚧 in progress
- ⏳ planned / not started
- 🔬 needs design decision (see "Open design decisions")
- 💭 stretch / far-out

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

---

## 2. Current state (baseline, 2026-06-21)

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

### Phase 2 — Consolidation: de-duplicate L / VL  ⏳

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
on real docs (citm/twitter).** Cleanest cell = numbers (raw, ~0 alloc both sides):
our number scanning is ~2.5× slower than achievable → the per-byte main-loop cost
is the tax. BUT easyjson allocates a string per value (`String()`, ~10⁴ allocs on
citm) where we reuse a buffer (single digits). **Phase 2 target: close the per-byte
scan gap with the push-core while keeping zero-alloc strings + zero-copy numbers →
win on both speed and allocations.**

- ⏳ **2.1 Design the shared core — now a *push* core.** Decision (from 1.1): the
  shared lowest-level scanner drives its own loop and emits tokens via a yield
  callback; `L`/`VL` become thin adapters. `NextToken` is the pull adapter (a
  push→pull bridge or a thin re-entry), `Tokens()` the native push adapter. This
  delivers the dedup *and* the iterator speedup in one rewrite. Must stay
  conformance- and benchmark-neutral (Phase 0 suite is the gate).
- ⏳ **2.2 Migrate** `L` then `VL` onto the shared core; keep all tests green
  (Phase 0 suite is the gate). Re-run benchmarks; no perf regression allowed.

### Phase 3 — Semantic features  ⏳

- ⏳ **3.1 Optional input normalization.** Make UTF-8 / escape processing
  toggleable (sanitizer hooks for strings and numbers, per old TODO).
- ⏳ **3.2 JSON canonicalization (RFC 8785).** Opt-in transform for strings and
  numbers. ⚠️ Number canonicalization (ECMAScript double) conflicts with the
  no-resolution-loss invariant — must be explicit opt-in, likely a higher layer. 🔬
- ⏳ **3.3 NDJSON.** Line-delimited JSON, especially for streaming `io.Reader`;
  top-level value sequence separated by `\n`.

### Phase 4 — Performance  ⏳

- ⏳ **4.1 Reduce memcopy.** Hunt avoidable copies (`currentValue` appends,
  `consumeN`, buffer-overturn copies); apply zero-copy from 1.4 where owned.
- ⏳ **4.2 Inlining pass.** Inspect inlining (`-gcflags=-m`); evaluate `go:fix
  inline`. May assume go1.26 features where they help.
- ⏳ **4.3 Full comparative benchmarks.** vs stdlib `jsontext`, `encoding/json/v2`,
  `mailru/easyjson`, and (if extractable) the go-ccy lexer.
- 💭 **4.4 SIMD variant.** Separate type/impl behind the same interface, inspired
  by `~/src/github.com/fredbi/zimdjson`. Likely its own module for build
  constraints. Far-out; depends on go1.26 `simd` direction. 🔬

---

## 5. Open design decisions (for debate)

1. **Conformance suite integration (0.1):** vendor a copy, git submodule, or
   reference by env/path? Affects CI portability.
2. **Streaming security defaults (0.3):** do `io.Reader` lexers get non-zero
   default `maxContainerStack` / `maxValueBytes`? What magnitudes? Behavior change.
3. **Iterator signature (1.1):** `iter.Seq[token.T]` (+ post-loop `Err()`) vs
   `iter.Seq2[token.T, <pos|err>]`?
4. **ElideSeparator default (1.2):** default-on for semantic breaks current
   comma-asserting tests — acceptable? Option name/placement (lexer option vs
   iterator wrapper)?
5. **Line/col scope (1.3):** verbatim-only, or also semantic (cost of counting
   newlines in skipped blank runs)? Where stored — on `token.VT`, or returned by
   the iterator?
6. **Zero-copy model (1.4):** new token variant exposing offsets, or `T.Value()`
   aliasing the source buffer when safe? How to signal "value borrowed, don't
   mutate"? Only legal in `NewWithBytes` mode.
7. **L/VL refactor strategy (2.1):** shared byte-scanner + policy layers, vs
   generics over token type, vs VL-as-superset. Tradeoff: readability vs perf.
8. **RFC 8785 numbers (3.2):** canonicalization loses precision by design —
   keep it strictly opt-in and out of the core, or refuse number canon entirely
   and only canonicalize strings/structure?
9. **SIMD packaging (4.4):** separate module + GOARCH build tags; pure-Go
   fallback; relationship to go1.26 `simd`.

---

## 6. Suggested first step

Phase 0.1 + 0.3: stand up the conformance harness (immediate, objective signal
on where we are) and confirm/fix streaming security defaults. Everything else
rests on that net.
