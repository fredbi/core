# Ramblings — lexer performance dissection & the pull-vs-push paradigm

> Research notes, 2026-06-23. Not a plan; a record of what we learned while
> hardening `json/lexers/default-lexer` and racing it against other lexers.
> Keep for when we extend scope (see [the go-openapi v2 programme](2026-06-go-openapi-v2-context.md)).

## The yardstick: jsontext (encoding/json/v2)

We deliberately race against **jsontext** (`github.com/go-json-experiment/json`,
the future `encoding/json/v2`), not easyjson and not the SIMD crowd. jsontext is:

- a **genuine, fully-validating streaming tokenizer** (`Decoder.ReadToken`),
- **pure Go, no SIMD** — the same category we want to win in,
- it **validates number grammar while scanning and does not convert** numbers to
  native types — i.e. it does exactly what our lexer does. Closest peer of all.

Surprise of the project: it took the Go team 12+ years to ship a good tokenizer.
jsontext is excellent and is the right bar. If/when it leaves the experimental
phase, **wrapping it and ending the story is a legitimate option** for the
non-verbatim path.

goccy/go-json and sonic were evaluated and set aside: **they are decoders, not
lexers** (compiled-VM / JIT that unmarshal straight into Go types; no extractable
token stream). easyjson's jlexer *is* a real pull lexer and a fair-ish peer, but
its number paths bracket us (Raw = no validation; Float64 = validate+convert+lose
precision) so it never does *exactly* our job.

## Standings (bytes mode, MB/s, mid-2026-06-23, after the number rewrite)

| workload | ours pull `L` | push proto `P` | jsontext | easyjson-raw | easyjson-f64 |
|---|---|---|---|---|---|
| ints | 566 | 320 | 785 | 515 | 196 |
| floats | 475 | — | — | — | — |
| canada | 495 | — | 593 | 527 | 209 |
| strings_plain | 878 | 1185 | 1421 | 388 | 392 |
| citm | 689 | 940 | 1122 | 578 | 492 |
| twitter | 628 | 916 | 631 | 407 | 392 |

Allocations: ours 2–8/op (≈1 pooled, **0 with `ResetWithBytes` reuse**),
jsontext 3–262/op, easyjson 1e4–1e5/op, stdlib v1 1e5–1e6/op. **Allocations +
no numeric conversion + no precision loss are our durable edge**, not raw MB/s.

## What actually moved the needle (ranked)

1. **Inlinable simple-number fast path** (`4a309ec`). Plain integer scanned inline
   in `scanToken` — no call, no state machine — mirroring jsontext's
   `ConsumeSimpleNumber`. ints 205→552 (+175%). The win is **eliminating the
   per-token call + state-machine setup**, NOT faster per-byte work: a
   positive-only variant captured the whole gain; extending to negatives was a
   wash on ints. Lesson burned in: *short tokens are bottlenecked by per-token
   overhead, not per-byte scanning.*
2. **Digit-run number scanner** (`9bee708`). Fractions/exponents via tight digit
   runs (`uint()`-BCE loops), grammar validated only at run transitions, value
   aliased. floats 240→475 (+98%), canada +111%. Streaming/capped numbers keep the
   old byte-by-byte scanner; only whole-buffer+no-cap uses the digit-run path.
3. **Zero-copy strings (whole-buffer)** — earlier work. strings ~379→728, etc.
   Alias unescaped strings (cap==len), copy only on escape.
4. **Zero-copy numbers (whole-buffer)** — alias the contiguous number bytes.
5. **Folding look-ahead / elision / key→colon into scanToken** — removed the
   `current`/`next` stash and a separate look-ahead pass. Accepted semantic
   change: malformed tokens may surface as a shorter valid value with the error
   **deferred one token** (`1.2.3`→`1.2` then reject `.3`); document still
   rejected; conformance unaffected; no test asserts a specific number error code.

What did NOT help: a per-number "local cursor" spike with the scanner still a
non-inlined call (~0 gain — proved the bottleneck was the call, not the cursor).

## The pull-vs-push paradigm — the real mechanism

Push (self-driving scan loop yielding via range-over-func) beats pull (NextToken
per call) on **strings/structure-heavy docs** (+35–46% on citm/twitter/strings),
loses on numbers **only because the prototype's `scanNumber` predates the fast
path**. Give push the same number scanner and that flips back.

Why push wins, precisely (two effects, first dominates):

1. **Cursor/state in locals across the WHOLE multi-token scan, one stack frame.**
   The tight loop does **no per-byte writes to struct fields** (`l.consumed++`/
   `l.offset++`). Profiling fingered exactly those as hottest. Pays off most on
   long values (strings) and dense structure.
2. **Direct yield, no re-entry** — emits via callback instead of returning up
   through `NextToken` and re-entering per token. Real but secondary.

Crucial nuance proven by disassembly (`go build -gcflags=-S`): **the Go compiler
already keeps the value-scanner cursor in registers.** In `consumeNumberWhole`'s
digit loop the cursor is `R11`, buffer base `DX`, length `R9`, byte load
`MOVBLZX (R11)(DX*1)`, increment `LEAQ 1(R11)` — **zero spills**. So the leaf
scanners are already register-optimal. The push advantage lives in the **main
loop**, where the *pull* path touches struct memory per byte. The fix is the
push loop **in pure Go** (local cursor) — the compiler register-allocates it
across all tokens. Not assembly (see [the codegen/asm/jit ramble](2026-06-codegen-asm-jit.md)).

### Decision

Build a native push `Tokens()` as a **duplicated main loop** carrying the inline
int fast path, reusing the value scanners (number/string/bool/null). Pull
`NextToken` stays the primary API; push backs the iterator. The main loop must
be duplicated (pull writes struct + returns per token; push keeps locals +
yields — different control flow, no clean Go reuse), but the **value scanners can
be single-sourced** if written as `(data, pos) -> (newPos, value, err)`.

### DONE (`0858b3f`) — push `Tokens()` landed, cursor-sync variant

`scanPush` (whole-buffer fast path of `Tokens()`): local cursor across the whole
scan, mirrors `scanToken` validation exactly, reuses the value scanners by
syncing `l.consumed`/`l.offset` around each call, inline int fast path duplicated.
**318/318 JSONTestSuite fixtures: stream + error-state identical to `NextToken`.**

| workload | pull NextToken | **push Tokens()** | Δ | proto P | jsontext |
|---|---|---|---|---|---|
| ints | 560 | 736 | +31% | 318 | 785 |
| floats | 514 | 652 | +27% | 454 | — |
| canada | 552 | 675 | +22% | 420 | 593 |
| strings | 877 | 978 | +12% | 1153 | 1421 |
| citm | 696 | 997 | +43% | 920 | 1122 |
| twitter | 636 | 792 | +25% | 911 | **631 (we win)** |
| mixed | 334 | 420 | +26% | 589 | — |

**Delta analysis (where push Tokens() trails the bare prototype P):** pure strings
(978 vs 1153) and tiny-token-dense `mixed` (420 vs 589). Cause: our per-value
**cursor-sync + full-scanner call** (`consumeString` key-detection,
`consumeBoolean`) is paid on every tiny token and not amortized; P inlines those
leanly with lighter validation. We *beat* P wherever values are larger / density
lower (citm, twitter) or numbers dominate (our fast scanner; P's is the old one).
**Next lever:** inline the no-escape string / bool / null cases directly in
`scanPush` (+ `bytes.IndexByte` string scan) to recover the strings/mixed gap —
exactly the duplication-vs-speed tradeoff the codegen generator would resolve from
one source.

## SWAR string fast path — DONE (`6946f19`)

Stdlib `bytes.IndexByte` is single-needle; a JSON string body needs the FIRST of
three needles (`"`, `\`, `<0x20`). So instead of `IndexByte`, an **8-byte SWAR
scan** (has-byte-less-than / has-byte-equal bit tricks) finds the first special
byte in one pass, with a linear scan as the source of truth once a word flags
(no false negatives). Inlined into `consumeStringWhole` to avoid a per-string
call. Helper `indexStringSpecial` kept as the exhaustively unit-tested reference.

**Tradeoff (shipped on Fred's call):** net win on real-world + string-heavy docs
(pull/push): citm +14/+15%, twitter +7/+14%, strings_plain +14/+14%,
strings_unicode +14/+16%; **regresses tiny-field-dense `mixed` −8% and
escape-heavy `strings_escaped` −13%**. SWAR's per-word setup loses for very short
strings; the unescape slow path (escapes) is unchanged so escaped strings only
pay SWAR entry cost. A byte-prefix hybrid to protect short strings was tried and
reverted — it penalized the medium-string wins (the biggest ones). OpenAPI specs
are medium/long-string-dominated with rare escapes, so this favours the use case.

## FINAL STANDINGS (2026-06-23 EOD) — best path (push Tokens()) vs jsontext

| workload | pull bytes | push Tokens() | jsontext | Tokens/jsontext |
|---|---|---|---|---|
| citm | 710 | 1022 | 1120 | 91% |
| twitter | 717 | **889** | 626 | **142% (win)** |
| strings_plain | 1010 | 1140 | 1395 | 82% |
| ints | 577 | 733 | 765 | 96% |
| mixed | 311 | **390** | 359 | **109% (win)** |

Our best path is at **parity-or-better with jsontext on 4 of 5 workloads**, only
trailing on pure long strings (82%) — while also doing zero-copy aliasing,
single-digit (0 reused) allocs, and no numeric conversion / no precision loss.
For a pure-Go validating lexer that's a strong place to pause the speed race.

## VL (verbatim lexer) baseline vs L — recorded 2026-06-24

First time we've measured VL. Steady-state, both reused (L `reset` /
VL `verbatim-reset`, allocated once + `ResetWithBytes` in the loop, both
**0 allocs/op**), bytes mode, MB/s. This is the only meaningful comparison — VL
has no competitor in its arena (full-fidelity tokenizing for TUIs/LSPs).

| workload | L MB/s | VL MB/s | L/VL (VL slowdown) |
|---|---|---|---|
| ints | 568 | 67 | **8.5×** |
| object_keys | 658 | 86 | 7.7× |
| floats | 529 | 74 | 7.1× |
| canada_geometry | 574 | 87 | 6.6× |
| whitespace_heavy | 555 | 91 | 6.1× |
| golang_source | 489 | 80 | 6.1× |
| strings_plain | 1005 | 199 | 5.1× |
| mixed | 306 | 64 | 4.8× |
| bools_nulls | 237 | 49 | 4.8× |
| strings_unicode | 922 | 201 | 4.6× |
| twitter_status | 708 | 159 | 4.5× |
| nested_objects | 177 | 40 | 4.4× |
| citm_catalog | 672 | 186 | 3.6× |
| nested_arrays | 111 | 38 | 2.9× |
| strings_escaped | 624 | 250 | 2.5× |

**Reading it.** VL is **3–8× slower than L**, NOT because verbatim fidelity is
inherently that costly, but because **VL never received any of L's
optimizations**. VL's `NextToken` is the *old* byte-by-byte main loop with a
mandatory per-token **look-ahead** (`l.next`), plus blank accumulation, and **no
fast paths at all**: no inline-int, no digit-run number scanner, no SWAR strings,
no whole-buffer zero-copy. The gap is therefore largest exactly where L's fast
paths pay most — numbers + tiny tokens (ints 8.5×, object_keys 7.7×, canada
6.6×) — and smallest on escape-heavy strings (2.5×), where L itself falls back
to the slow unescape path so the two converge.

**Implication for unification.** This is the strongest argument yet for
generating/parameterizing L and VL from **one optimized source**: VL would
inherit every fast path, and the *residual* gap would shrink to the genuine cost
of fidelity (blanks + always-on positions + look-ahead) — plausibly ~1.2–1.5×,
not 3–8×. The current 3–8× is an **optimization-debt gap, not a fidelity gap.**
Don't hand-port the fast paths into VL's bespoke loop (that doubles the
maintenance we're trying to kill); fix it at the unification step instead.

Side win from this measurement: added `VL.ResetWithBytes` / `VL.ResetWithReader`
(parallel to L), so VL is now poolable/reusable with 0 steady-state allocs.

## Open levers still worth trying (pure Go)

- **L/VL unification** is now the priority lever: it's both the maintainability
  fix AND a 3–8× VL speedup (VL inherits L's fast paths). See codegen ramble.
- Pure long strings (strings_plain 82% of jsontext) is the last L gap — would
  need a faster unescape slow path and/or SWAR-ing the slow-path clean runs.
- Force-inline measurement already sized the codegen *speed* prize for L at
  ~3–7% (see codegen ramble); for VL the prize is far bigger (the table above).
