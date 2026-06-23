# Conclusions — building a fast, maintainable, pure-Go JSON lexer

> Distilled, transferable findings from the `json/lexers/default-lexer`
> investigation (2026-06-21 .. 2026-06-24). This is the "start here" summary; the
> blow-by-blow data and decisions live in the companion ramblings:
> [perf & paradigm](2026-06-perf-and-paradigm.md),
> [codegen / asm / jit](2026-06-codegen-asm-jit.md),
> [go-openapi v2 context](2026-06-go-openapi-v2-context.md).
>
> Most conclusions are general to writing fast scalar scanners/parsers in Go, not
> specific to JSON or this repo. Numbers are MB/s, bytes mode, one machine,
> mid-2026; treat magnitudes as ratios, not absolutes.

---

## 0. The headline

A **pure-Go, fully-validating, zero-conversion** JSON lexer can reach
**parity-or-better with the state of the art (`encoding/json/v2`'s `jsontext`)
on 4 of 5 workloads** while keeping **near-zero allocations** and **no precision
loss** — without SIMD, without hand-assembly, without JIT, and without a code
generator. The remaining gap (pure long strings) is **algorithmic**, not a
missing low-level trick. The maintainable, modular implementation is within
**3–7%** of anything aggressive inlining could buy.

The practical lesson: **measure where the time actually goes before reaching for
heroics.** Almost every "obvious" low-level lever (asm, JIT, more inlining,
custom SIMD) turned out to buy little or nothing here, for reasons that are
mechanical and predictable once measured.

---

## 1. Choose the right yardstick

- **`jsontext` (encoding/json/v2) is the right bar** for a *lexer*: it is a
  genuine, fully RFC-validating streaming tokenizer, pure Go (no SIMD), and it
  does NOT convert numbers to native types — i.e. it does exactly our job. Beating
  it in pure Go is a meaningful result; reaching for SIMD would change the
  category and concede the comparison.
- **goccy/go-json and sonic are decoders, not lexers** — compiled-VM / JIT that
  unmarshal straight into Go types, with no extractable token stream. Not
  comparable as lexers.
- **easyjson's `jlexer` is a real pull lexer but its number paths bracket ours:**
  `Raw()` skips number validation (faster, unfair), `Float64()` validates *and*
  converts *and* loses precision (slower, also unfair). There is no easyjson API
  that does "validate, don't convert" — which is precisely our niche. **Pick peers
  that do the same work; otherwise the benchmark lies.**

## 2. The dominant cost in short-token scanning is per-TOKEN, not per-BYTE

The single most important finding.

- An **inlinable "simple value" fast path** at the call site (a plain integer
  scanned inline, no function call, no state-machine setup) gave **+175% on
  integer-heavy input** (205→552 MB/s). A *positive-only* variant captured the
  entire win; extending it to negatives barely moved the needle.
- The earlier attempt that kept the scanner a **separate (non-inlined) call** but
  used a local cursor gave **~0%**. Same per-byte work, still a call → no win.
- **Conclusion:** for short tokens (numbers, delimiters, keys), the bottleneck is
  the per-token *call + setup* overhead, not the per-byte loop. The fix is to make
  the common case **small enough to inline at the call site** (mirrors jsontext's
  `ConsumeSimpleNumber`). Order `switch`/`if` so the **common case is the first
  branch** — that alone recovered a few percent.

## 3. For per-byte loops: digit-runs + `uint()` bounds-check elimination

- Restructuring the number scanner into **tight digit-run loops**
  (`for uint(n) < uint(len(b)) && '0' <= b[n] && b[n] <= '9' { n++ }`), validating
  grammar only **at the transitions** between runs (sign / int / frac / exp)
  rather than classifying every byte through a big switch, **doubled float
  throughput** (+98% floats, +111% on float-heavy real data).
- The `uint(idx) < uint(len(slice))` idiom lets the compiler **prove the index is
  in range and drop the bounds check**. Indexing with a *separate length field*
  (`l.bufferized`) instead of `len(slice)` defeats BCE — every access pays a check.
- **Conclusion:** separate the hot "consume a run of class X" loop from the
  grammar state machine; feed the compiler `len()`-relative `uint` comparisons.

## 4. Pull vs push: it's about register residency, not the iterator

- A **push** lexer (self-driving scan loop yielding via range-over-func) beat the
  **pull** (`NextToken`-per-call) lexer by **+12–43%** on string/structure-heavy
  input.
- The mechanism is **NOT** "push is a nicer API." It is: the push loop keeps the
  cursor (and container stack) in **locals across the entire scan**, so the hot
  loop does **no per-byte writes to struct fields** (`l.consumed++`/`l.offset++`).
  A secondary effect is yielding directly vs returning-and-re-entering per token.
- **Disassembly proved the leaf scanners are already register-optimal:** in
  `consumeNumberWhole`'s digit loop the cursor lives in `R11`, base in `DX`, len in
  `R9`, with zero spills. So the push win comes specifically from the **main
  loop**, where the *pull* design wrote the cursor to memory every byte.
- **Conclusion:** to get "registers not memory," write a flat function with a
  **local** cursor across the whole scan; the Go SSA backend register-allocates it.
  You do not need assembly for this (see §7).

## 5. String scanning: multi-needle SWAR beats `IndexByte`, with caveats

- A JSON string body must stop at the **first of three** bytes: `"`, `\`, or any
  control char (`< 0x20`). `bytes.IndexByte` is **single-needle** (and is itself
  hand-tuned SIMD asm), so it can't express this in one pass.
- A **SWAR** scan (8 bytes/word, "has-byte-less-than" + "has-byte-equal" bit
  tricks, OR-ed) finds the first special byte in one pass; use a plain byte scan as
  the **source of truth** once a word flags (no dependence on marker placement, no
  false negatives). Inline it (don't make it a per-string call).
- **Tradeoff, and it's real:** SWAR **wins on medium/long strings** (citm/twitter
  +14%, plain/unicode +14–16%) but **loses on very short strings** (tiny-field
  payloads −8%) and **escape-heavy strings** (−13%, because the unescape slow path
  is unchanged and they just pay SWAR entry cost). A byte-prefix "protect short
  strings" hybrid **backfired** — it penalized the medium strings that were the
  biggest wins. Shipped pure SWAR because the target corpus (OpenAPI specs) is
  medium/long-string dominated with rare escapes.
- **Conclusion:** vectorized search pays in proportion to span length; for
  short-token-dense data it can be net-negative. Know your input shape.

## 6. The force-inline / codegen prize is ~3–7% — not a speed lever

We worried about an "easyjson tar pit" (inline-by-hand until unreadable) and
considered **generating a flat lexer from a maintainable golden source** via
`golang.org/x/tools/internal/refactor/inline`. Before building it, we measured the
ceiling.

- **How to force-inline-measure in Go (no `//go:inline` pragma exists):**
  1. `go build -gcflags=-m=2` → callee inline costs (ours: string 702, number 384,
     bool 254, null 105; all ≫ budget 80, so each is a real call).
  2. `-d=inlbudgetslack` is a dead end (only under the experimental new inliner).
  3. **PGO is the clean tool** (as in Go's own `pgo_inl_test.go`): `-cpuprofile`
     to get a profile, build with `-pgo=` / `-gcflags=-pgoprofile=…
     -d=pgoinlinebudget=N,pgoinlinecdfthreshold=90`, verify with `-m=2` that the
     scanners now inline into the driver, then benchstat on/off. PGO is
     profile-driven, so profile the workload whose call sites you want collapsed.
- **Result:** with the scanners verifiably inlined into the scan loop, speedup was
  **+2–3% on most workloads, +7% on the most token-dense one**. Big scanners do
  real per-byte work; the call boundary is negligible beside it. (Contrast §2: the
  int fast path won by removing *setup work*, not a call.)
- **Conclusion:** there is **no meaningful inlining prize left**. Maintainable
  modular code is within 3–7% of fully-inlined. A codegen-by-inlining generator is
  **not justified for speed**; its only possible justification is **L/VL-style
  unification / maintainability**, judged on its own merits. PGO could harvest the
  3–7%, but a *library cannot propagate PGO to consumers' builds*, so shipping a
  `default.pgo` is pointless too.

## 7. Scalar assembly buys nothing here (and usually won't)

- Motivation considered: "push wins via registers, so hand-asm a kernel to force
  registers." (asm ≠ SIMD; staying pure-Go was *not* the reason to reject it.)
- **Disassembly shows the compiler is already register-optimal** for the leaf
  kernels (§4). Asm would reproduce the same `MOVBLZX`/`CMPB`/`LEAQ`.
- **An asm kernel is a non-inlinable CALL, and the Go ABI spills register state at
  the boundary.** The push win is register-residency *across the whole multi-token
  scan in one frame*; a *per-token* asm sub-parser breaks the frame and forces
  state through memory every token — the opposite of the goal — plus a mandatory
  call. The only asm form that preserves the property is "the whole loop in asm" =
  unmaintainable, per-arch, hand-doing what the compiler does for flat Go.
- **Conclusion:** for scalar control-flow scanning, Go's codegen is at/near the
  ceiling; assembly is the one construct *guaranteed* to break inlining and
  reintroduce a frame boundary. Reserve asm for genuine vector ops Go can't express.

## 8. JIT (sonic-style) doesn't transfer to a type-agnostic lexer

- A JIT'd whole-document routine **does** escape the per-token ABI barrier (the
  Go→blob call is crossed once, amortized; inside, registers are fully controlled).
  But it escapes it the *same way* "one giant flat function" does — and a flat
  AOT-compiled Go function already reaches that regime (§4, §6).
- The thing JIT *uniquely* buys — **runtime specialization to the target Go type**
  (baking field offsets/types in, fusing lex+parse+populate) — is what makes sonic
  fast and is **irrelevant to a lexer**: there is no schema to bake in, so it would
  emit the same routine every time.
- Cost is the highest of all options (`golang-asm` = vendored fork of the internal
  assembler, pinned to a toolchain; W^X pages; GC can't see into the blob;
  per-arch; fallback required).
- **Conclusion:** JIT is the right hammer for *type-specialized decoders*, the
  wrong one for a lexer. It returns to the table only one layer up — a **typed
  decoder** or a **schema→program compiler** (validator) — where there's a type or
  schema to specialize on.

## 9. SIMD belongs elsewhere (and the cheap part is free)

- Full custom SIMD (simdjson-style) is a **two-stage batch parser** (vectorize the
  whole buffer → structural bitmaps → walk a tape): whole-buffer, not streaming,
  not token-at-a-time. It collides with low-memory/streaming/pull goals, needs
  per-arch kernels + feature detection + a fallback, and changes the competitive
  category.
- If ever wanted, the clean home is a **separate, optional, whole-buffer pluggable
  implementation behind the lexer interface** — not woven into the core.
- The cheap 80/20 of "vectorized scanning" is **free** via stdlib (`bytes.IndexByte`
  is SIMD asm); but for multi-needle string scanning, SWAR (§5) beat single-needle
  IndexByte anyway.

## 10. Allocations & zero-copy are the durable edge

- In **whole-buffer mode**, value tokens can **alias the input** (3-index slice
  `buf[start:end:end]` so `cap==len` makes any later append copy-on-write). This is
  zero-copy for unescaped strings and all numbers; only escaped strings copy.
- The constraint enabling aliasing is **buffer stability**, not caller ownership.
  Streaming (refilling) buffers must copy.
- Steady-state allocations: **~0 with reuse** (`Reset`-style rebinding) vs jsontext
  3–262/op and easyjson 1e4–1e5/op. For huge documents this — plus no numeric
  conversion / no precision loss — is a more durable differentiator than raw MB/s.
- **Reuse beats pooling:** allocate once + `ResetWithBytes` per input → 0 allocs/op
  and identical throughput; a sync.Pool path still showed ~1 alloc/op.

## 11. Methodology that paid off

- **benchstat A/B via `git stash`**: bench the working tree, stash the change,
  bench the baseline, `git stash pop` — same binary/machine, honest deltas. (Run
  the stash→bench→pop as one command so the pop always executes even on timeout.)
- **Equivalence gating**: a test that asserts the optimized path yields the exact
  same token stream + error state as the reference path, over the **whole
  conformance corpus** (318 fixtures), caught every divergence cheaply. Optimize
  freely behind a hard equivalence gate.
- **Disassembly (`-gcflags=-S`) and inline maps (`-gcflags=-m=2`)** answer
  "is this already optimal?" definitively — before writing asm or a generator.
- **Measure the ceiling before building the machine** (§6): PGO sized the inline
  prize at 3–7% and saved us from building a generator for nothing.
- Beware Go's `-bench` regex with `/`-containing names: alternation at a deep path
  segment silently drops sub-benchmarks. Prefer per-target runs.

## 12. Where each technique belongs (the layered map)

| Layer | Right tool | Why |
|---|---|---|
| Lexer (type-agnostic tokens) | flat Go, local cursor, inline fast paths, SWAR, zero-copy | compiler already register-optimal; no schema to specialize |
| Typed decoder (unmarshal into Go types) | codegen and/or JIT specialization | a *type* to bake in → real specialization win |
| Schema validator (OP-program) | codegen, possibly JIT | a *schema* to compile into an optimized program |
| Max-throughput batch parse of big in-memory docs | optional SIMD pluggable | whole-buffer batch is SIMD's home; keep it off the streaming core |

**Overarching conclusion:** push the pure-Go scalar lexer to the jsontext bar with
inlinable fast paths + digit-runs + SWAR + zero-copy, keep it modular and
readable (it's within 3–7% of any inlining heroics), and **defer asm/JIT/SIMD to
the higher layers where there is actually a type or schema to specialize on.**
