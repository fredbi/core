# Ramblings — roads to speed without the easyjson tar pit

> Research notes, 2026-06-23. The maintainability question: how do we keep
> chasing jsontext-class throughput WITHOUT ending up like easyjson — an
> inlined-by-hand parser nobody dares touch after 10 years? Companion to
> [the perf/paradigm ramble](2026-06-perf-and-paradigm.md).

## The problem statement (Fred)

The structural optimizations that *actually matter* (pull vs push, number fast
paths, zero-copy numbers + unaltered strings) are finite. Past them lies a "race
of inlining more stuff" — diminishing returns, rising unreadability. easyjson is
the cautionary tale. We want a **maintainable set of lexers** (readable
functions, real tests) and are willing to *generate* the optimized artifact from
them. Sequencing: finish dissecting jsontext's tricks first (we can't spec the
generator until we know the exact shape it must emit), THEN choose the road.

Four roads were weighed. Verdicts below; the recurring test is **the per-token
ABI boundary** and **what the Go compiler already does for us**.

## Road 1 — Generate a flat lexer from a maintainable golden source (FAVOURED)

Write one readable, function-decomposed, unit-tested lexer (the "golden
source"); mechanically **flatten/inline** it into the optimized artifact; never
hand-edit the artifact. Tooling idea: `golang.org/x/tools/internal/refactor/inline`
(the engine behind gopls "inline call" and `//go:fix inline`) — `internal`, so
vendor/fork it. This lets us inline exactly what we want, **ignoring the
compiler's 80-cost inline budget** (push was 82 — a near-miss that the budget,
not the code, blocked).

Why it's the right shape:
- **Single source of truth, readable + testable.** easyjson-done-right (easyjson
  generates from struct types; we'd generate from a reference lexer).
- **May moot L/VL factorization.** Generate both `L` and `VL` from one
  parameterized golden source (policy: emit `token.T` vs `token.VT`, track blanks
  / positions or not). The ~750 lines of duplication vanish at source level. This
  may be the biggest prize, independent of raw speed.

Critical caveat — **inlining alone ≠ the wins we measured.** `refactor/inline`
inlines a call at a call site. But most of our speed came from **cursor-in-locals
(no per-byte struct writes)** + restructure, not call elimination. Inlining a
struct-based scanner into a struct-based loop yields a flat-but-slow function that
still writes `l.consumed` every byte. So the **golden source must be written in
local-cursor / threaded-state style** — small pure fns over `(data, pos, state)`
returning new positions — so that flattening collapses the locals into one frame
and the compiler register-allocates across the whole scan. The codegen target
dictates the golden-source discipline.

Before committing: **measure the prize.** → DONE 2026-06-24, see below.

### MEASURED (2026-06-24): the force-inline prize is ~3–7% — NOT worth a generator for speed

How to measure force-inline in Go (there is no `//go:inline` pragma):
1. **Inline map:** `go build -gcflags=-m=2` → costs of the hot callees. Ours:
   consumeStringWhole 702, consumeNumberWhole 384, consumeBoolean 254,
   consumeNull 105, scanPush 2446 — all ≫ budget 80, so every value token in
   scanPush crosses a real (non-inlined) call.
2. **Budget crank — dead end.** `-d=inlbudgetslack=N` only applies under the
   experimental new inliner; a plain build still reports "budget 80". No usable
   `inlbudget` flag.
3. **PGO is the clean force-inline tool** (mirrors the Go team's
   `cmd/compile/internal/test/pgo_inl_test.go`): generate a CPU profile
   (`-cpuprofile`), build with `-pgo=<file>` (compiler: `-gcflags=-pgoprofile=…
   -d=pgoinlinebudget=N,pgoinlinecdfthreshold=90`), verify with `-m=2` that the
   scanners now `inlining call to …` inside scanPush, then benchstat on/off. PGO
   is profile-driven: only HOT call sites inline, so profile the workload whose
   scanners you want collapsed (string-heavy citm collapsed the string chain;
   token-dense mixed collapsed string+number+bool+null).

**Result:** with the scanners verifiably inlined into scanPush, speedup was
**+2–3% on most workloads, +7.1% on the most token-dense (mixed)** — citm even
−3% (noise). The big scanners do real per-byte work; the call boundary +
cursor-sync is negligible beside it. (Contrast the int fast path, which won big
because it removed a state-machine *setup*, i.e. WORK, not just a call.) The
prototype P's large mixed lead (589 vs ~400) is therefore **validation cost +
leaner algorithm, not inlining** — i.e. the price of full conformance, which we
keep.

**Decision:** the inline/codegen road is **NOT justified for speed** (~3–7%). Our
readable modular code is already within 3–7% of fully-inlined. PGO could harvest
that 3–7% with zero generator, but a *library* can't propagate PGO to consumers'
builds, so it isn't worth shipping a default.pgo either. The generator's only
remaining justification is **L/VL unification / maintainability** — weigh that on
its own merits, not as a performance play. The "race of inlining" worry is moot:
there is no meaningful inlining prize left to chase.

Cheaper alternative for the L/VL half only: **Go generics with a concrete policy
type param** (not interface) can monomorphize + devirtualize — one generic lexer,
two instantiations, no vendored toolchain. Won't beat the inline budget but may
unify L/VL "well enough." Spike it against the codegen idea.

## Road 2 — Scalar assembly kernels (REJECTED, on evidence)

Idea: the push win is "registers not memory", so hand-asm a small sub-parser
(e.g. the int fast path) to force register residency. (Note: asm ≠ SIMD;
"staying a pure-Go player" is a non-goal — rejection is purely mechanical.)

Why it fails:
1. **The Go compiler already does it.** Disassembly of `consumeNumberWhole`'s
   digit loop: cursor `R11`, base `DX`, len `R9`, `MOVBLZX (R11)(DX*1)`,
   `LEAQ 1(R11)` — zero spills. The leaf kernels are already register-optimal;
   asm would reproduce the identical instructions at best.
2. **An asm kernel is a non-inlinable CALL, and the ABI spills state at the
   boundary.** The push win is register-residency **across the whole multi-token
   scan in one frame**. A *per-token* asm sub-parser breaks the frame into pieces:
   cursor goes register→(ABI)→register every token, plus a mandatory `CALL`/`RET`
   that never inlines. For short tokens (`ints` — exactly where you'd want it),
   per-token overhead is the dominant cost — so asm makes it worse.
3. The only asm form preserving register-across-tokens is "the whole loop in asm"
   = unmaintainable, per-arch, hand-doing what the Go compiler does for a flat Go
   function. So even in the limit asm doesn't beat "flat Go from generator".

Asm is the one construct **guaranteed to break inlining** — it works against the
mechanism we want.

## Road 3 — JIT assembly à la sonic / bytedance (REJECTED for a lexer)

Sonic JITs a type-specialized decoder at init via `golang-asm`
(`twitchyliquid64/golang-asm`, a vendored fork of Go's internal `cmd/internal/obj`).

Does the per-token ABI barrier apply? **No** — and that's the interesting part:
the JIT'd routine spans the **whole document**, so the Go→blob call is crossed
once (amortized), and inside the blob sonic controls all registers with no
inter-token spills and no internal calls. It *does* get register-residency across
the whole scan.

But it escapes the barrier the **same way "one giant asm routine" does**, and a
**flat AOT-compiled Go function reaches the same regime** (we proved the compiler
holds state in registers across a flat function). So for *scalar* lexing the JIT's
headroom over flat Go is the same near-zero as Road 2.

The one thing JIT uniquely buys — **runtime specialization to the target Go
type** (baking field offsets/types into the instructions, fusing
lex+parse+populate) — is what makes sonic fast and is **irrelevant to a
type-agnostic lexer**: there's no schema to bake in, so it would emit the same
routine every time ⇒ "build once at init" = "just have the routine" = no gain
over a precompiled flat Go function.

Cost is the **highest of all options**: `golang-asm` is pinned to a toolchain,
breaks across Go releases, needs a pure-Go fallback, W^X pages, GC can't see into
the blob (pointer/preemption discipline), amd64-first. Antithesis of the
maintainable-golden-source goal.

**Where JIT could legitimately return:** one layer UP, if we ever build a *typed
decoder* over the lexer (sonic/goccy-style "unmarshal into this Go type"), runtime
type-specialization becomes a real lever. Also flagged for the **schema validator
OP-program** (see context doc): compiling a schema into an optimized program is
precisely the kind of thing where pure-Go-codegen *or* JIT both make sense.

## Road 4 — SIMD (DEFERRED to an optional, separate variant)

Full custom SIMD (minio simdjson-go via c2goasm; klauspost) is a **two-stage
batch parser** (vectorize whole buffer → structural bitmaps → walk to a tape):
whole-buffer, not streaming, not token-at-a-time — collides with our low-memory /
streaming / pull goals. Per-arch kernels + feature detection + pure-Go fallback =
triple maintenance. It also **changes our competitive category** (jsontext is
non-SIMD).

If ever, the clean home is a **separate optional pluggable `simd-lexer` behind
the `lexers.Lexer` interface**, whole-buffer-only, for max throughput on big
in-memory docs — explicitly NOT woven into default-lexer. Our interface already
permits this. Realistic only via `GOEXPERIMENT=simd` native support, and per the
v2 programme this is **runtime-usage only** (untyped Document runtime), not a core
goal.

The cheap 80/20 of "vectorized scanning" we can have **for free**: stdlib
`bytes.IndexByte` is already SIMD asm — use it for the string fast path (see perf
ramble). No asm owned, streaming-compatible.

## Bottom line

- Finish the **structural** dissection first (push `Tokens()`, `IndexByte`
  strings, keep racing jsontext). Produce two deliverables: (1) a ranked list of
  transforms that actually matter, (2) a force-inline measurement sizing the
  codegen prize.
- Then favour **Road 1** (generate flat lexer(s) from a maintainable, local-cursor
  golden source), possibly with **generics** for L/VL unification. Reject scalar
  asm (Road 2) and JIT (Road 3) for the lexer on the evidence above.
- Keep asm/JIT/SIMD in the back pocket for **later, higher layers**: a possible
  typed decoder, the schema-validator OP-program, and a runtime-only SIMD/JSON-LD
  variant.
