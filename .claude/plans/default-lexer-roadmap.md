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

#### 0.1 baseline findings (16 false-accepts + extras)

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

- ⏳ **0.2 Benchmark harness + baseline.** Flesh out `benchmarks/comparative`
  with at least stdlib `jsontext` to capture a baseline now (full field added in
  Phase 4). Record ns/op + B/op + allocs/op for a few representative fixtures
  (small object, big array, deeply nested, long strings, huge numbers).
- ⏳ **0.3 Security-by-default audit.** Review guards against hostile streams:
  unbounded nesting, unbounded value size, buffer growth. Decide **safe defaults
  for streaming mode** (today both breakers default to 0 = unlimited). 🔬
- ⏳ **0.4 seriot.ch pitfalls pass.** Walk https://seriot.ch/security/parsing_json.html;
  map each documented pitfall to a test case + our behavior. Overlaps with 0.1.
- ⏳ **0.5 Fix known nits** (§2 🐛) + raise coverage on existing code (pools path).

### Phase 1 — Interface & API surface  ⏳

Additive, low-risk; done before the refactor so the core targets the final shape.

- ⏳ **1.1 Iterator API.** Add a range-over-func walk to the `Lexer`/`VerbatimLexer`
  interfaces (Go 1.23+). Signature TBD (`iter.Seq[token.T]` + check `Err()`, vs
  `iter.Seq2`). 🔬
- ⏳ **1.2 `WithElideSeparator` option.** Skip `,` tokens in the emit loop
  (grammar still validated internally). **Default ON for semantic `L`**, off for
  verbatim. Reconcile with existing tests that assert comma tokens. 🔬
- ⏳ **1.3 Line/column tracking.** Verbatim tokens expose line/col, not just
  offset (TUI/GUI/LSP positioning). Cheap in `VL` (already scans blanks);
  optional in `L`. 🔬
- ⏳ **1.4 Zero-copy / offset path (design).** For callers that fully own the
  input `[]byte`: expose token values as offsets/sub-slices into the source
  instead of copies, when no unescaping/normalization is needed. 🔬

### Phase 2 — Consolidation: de-duplicate L / VL  ⏳

Risky remodel, executed with Phase 0 net in place and Phase 1 shape known.

- ⏳ **2.1 Design the shared core.** Candidate: extract a lowest-level byte
  scanner producing grammar events; `L` and `VL` become thin policy layers
  (blank handling, token construction). Alternatives in Open decisions. 🔬
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
