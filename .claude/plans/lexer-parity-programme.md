# default-lexer — parity programme

**Goal.** Reach *at least parity* with `go-json-experiment/json` (jsontext) on the
paths where we do comparable work, after a structured lab phase that locks in
proven gains one at a time. Stop running in circles: every change is justified by
a laser-focused micro-benchmark first, then validated on real corpora, before it
is allowed back into the reference implementation.

Status legend: ✅ done · 🚧 in progress · ⬜ todo · ⏸️ parked · ❌ rejected (kept for the record)

---

## 0. TL;DR — the decisions this doc locks in

1. **Define "parity" honestly, then minimize the tax.** We *eagerly unescape*
   strings inside the tokenizer; jsontext *validates now, unescapes later*. On any
   escaped string we do strictly more work, so we will **never** match their
   escaped-string throughput while keeping the eager "ready-to-use value"
   contract — and we keep it (it is the feature). Parity is measured on the
   **work-symmetric paths** (plain strings, numbers, structure, literal UTF-8).
   The whole string game (§4.2) is using SWAR to *avoid the eager work* when there
   is none to do (no escapes → zero-copy, done) or little (long clean trail →
   batch, done; in-place no-grow → candidate). No lazy-token detour.
2. **16-byte token: REJECTED with measurement, both representations (§5.3).** Tested
   `*[]byte`+stable-field AND `unsafe.Pointer`+`uint32` len vs the 32 B token. The
   32 B by-value token is the sweet spot: its copy is already cheap (NOT the
   bottleneck) and its value is ready without reconstruction. `*[]byte` loses
   17-26% (forces the slice header into memory to take its address). `unsafe.Pointer`
   removes that round-trip and **ties** kind-only but **loses ~6-7%** read-value
   (rebuilding the slice via `unsafe.Slice` costs more than the copy saved) — plus
   GC/unsafe burden. Net: no win in any consumer profile. Per-token cost is
   **dispatch, not token size** → §5.1 devirt / §5.2 classification, not the struct.
3. **Hand-rolled SWAR stays — but gets consolidated.** Stdlib gives us nothing
   for the fused `{0x22, 0x5c, <0x20}` test (`IndexByte` is single-needle SIMD;
   `IndexAny` is a scalar ASCII bitmap; `<0x20` has no primitive). We build one
   small in-tree, all-inlinable SWAR module and route every scan through it.
4. **Devirtualization by codegen, as PREP — no build tags, no dispatch layer.**
   Follows the writer's `writegen` model (`json/writers/default-writer/internal/writegen`):
   lift the generic scan cores into plain concrete functions (monomorphized,
   policy bound concrete so `p.emit` inlines). The lab keeps *both* the generic
   and the generated entry points so a benchmark calls each and measures the gap.
5. **Dispatch: our switch is a binary search, not a jump table (verified).** The
   sparse JSON byte set compiles to ~4 `CMPB` comparisons, so there *is* dispatch
   headroom. Capture it with a 256-entry **classification table** (one load → a
   dense switch on a contiguous class enum the compiler *can* jump-table → direct,
   inlinable arms) — **not** a `[]func` table (O(1) but an indirect call per
   token, which re-introduces the very tax §5.1 removes and blocks inlining).

---

## 0.5 Scoreboard — tentatives & outcomes (2026-06-27)

One line per tentative. "Gain" = measured B/s vs the frozen reference (cumulative)
or vs the in-binary generic baseline (devirt), count=10 unless noted.

**Prep (tooling — no direct perf):**
- ✅ Micro-benchmark suite — 12 single-path workloads + harness + jsontext oracle guard.
- ✅ Lab re-synced from reference (option A, pristine copy) — establishes a clean baseline.
- ✅ In-tree SWAR module (`internal/swar`) — inlinable exact masks; string-stop consolidated **perf-neutral**.
- ✅ `lexgen` devirt generator — monomorphizes the cores; idempotent; lexgen = source-of-truth.

**Adopted (shipped to the lab):**
- ✅ **push-devirt** — `Tokens()` via generated cores — **push +7…+18%** (in-binary).
- ✅ **number fast path → full grammar** (`[-]int[frac][exp]` inline) — **decimals +18/+36%, exponential +13.6/+38%** (pull/push).
- ✅ **string fast/slow split + FirstByte** — **unicode +15%, uescaped +14.7%, plain +5.6%** (pull); no escaped_long regression.
- ✅ **pull-devirt** — `NextToken()` via generated cores — **pull +4.23% geomean** (after the regression resolved itself via the string split).
- ✅ **adaptive-SWAR slow path** (Fred's heuristic: escapes sparse → scalar-probe then SWAR the long tail) — **escaped_long +75% (2375, beats jsontext 1708), no dense regression**.
- ✅ cleanups (`consumeNumberWhole` doc, uint/BCE) — no perf change, clarity; BCE verified preserved.

**Rejected with measurement (recorded so we don't relitigate):**
- ❌ **16-byte token, `*[]byte`** — **−17…−26%** (header materialized to memory to take its address).
- ❌ **16-byte token, `unsafe.Pointer`+len** — **tie kind-only / −6-7% read-value** (slice rebuild + GC/unsafe burden).
- ❌ **classification-table dispatch** — **−5.16% geomean** (binary-search switch predicts better; L1 load + computed jump cost more).

**Cumulative (lab vs frozen reference, geomean):** **pull +13.5% · push +14.3%**
(`ramblings/2026-06-lab-scoreboard.txt`). Biggest movers: decimals/exponential
(+22…+32%), unicode/uescaped (+20…+24%), plain (+12…+16%).

**Rematch vs jsontext (pull/bytes, after adaptive-SWAR):** **WIN** twitter +15%,
golang_source +5.5%, escaped_long +39%; **parity** canada (99%); **trail** citm
(55% — many-tiny-tokens regime) and dense escaped (68% — structural: we unescape,
they don't). `ramblings/2026-06-adaptive-slowpath-swar.txt`. → **citm is the next
target** (per-token throughput on small string keys + numbers).

**Lesson banked:** the wins all came from **doing less work** (devirt, number/string
fast paths) and **splitting fragile shared cores** (which even cured the pull-devirt
regression). Per-token *structure* levers (token size, dispatch shape) both failed
measurement — the 32 B token + binary-search switch are already near-optimal.

---

## 1. Where we stand vs json/v2 (the parity map)

Source-grounded against the local clone at
`/home/fred/src/github.com/go-json-experiment/json` (design overview in its
README). "Us" = `default-lexer` (semantic `L` / verbatim `VL`).

| Dimension | default-lexer (us) | jsontext (them) | Note |
|---|---|---|---|
| Input encoding | UTF-8 JSON | UTF-8 JSON | same |
| Number validation | full grammar | full grammar | same (both beat easyjson here) |
| Number representation | raw `[]byte`, no parse | raw `[]byte`, no parse | same; parse deferred both sides |
| **Strings** | **eager unescape** (ready to use) | **validate-only; unescape deferred** to `AppendUnquote`/`Token.String()` | *the* asymmetry — see §4.2 |
| **Surrogate repair** | yes (we fix `\uD8xx\uDCxx`) | no (deferred to caller) | edge-case convenience we provide |
| Token size | `T` = 32 B, `VT` = 72 B | `Token` ≈ 2 words + raw ptr | ours exposes the value inline |
| State tracked | offset **+ line + column** + sticky err | byte **offset only**, sticky peek-err | line/col is ours alone, at a cost |
| Allocations | zero unamortized | initial 64 B buffer →grows; carries name/namespace stacks for dup-key detection | both ≈ zero in steady whole-buffer |
| **Streaming** | yes (slower path) | yes — clean *resumable* state machine (`ConsumeStringResumable`, `ConsumeNumberResumable`) | learn from their resumable design (§6) |
| **Push/pull API** | **pull (`NextToken`) + push (`Tokens` iterator, faster)** | **pull only** (`ReadToken`); no token-level iterator | our push path is genuinely distinctive |
| Separator handling | elision **on option** (default on) | **always elided** (inferred from grammar) | we can also surface `,`/`:` when asked |
| Verbatim/semantic | **two tokens**: semantic (fast) + verbatim (blanks+line+col, for UI/LSP/colorizers) | always raw bytes; no semantic-vs-verbatim mode | our verbatim token is distinctive |

**Reading of the map.** Our distinctive features (line/col, push API, verbatim
token, surrogate repair, eager unescape, optional separators) are all real value
for go-openapi v2 consumers — but several are *paid for on the hot path*. The
programme's job is to make the *shared* paths match jsontext and make the
*extra-value* paths as cheap as possible, not to delete the value.

---

## 2. The lab

### 2.1 What "the lab" is
The `exploration` worktree (off master `1ac8025`) **is** the lab; master is the
reference. We iterate here; we retrofit to master only at §7. No second physical
copy of the package — a full copy rots. Instead:

- **Pure-function A/B** (a candidate scanner vs the baseline scanner): write both
  as *separate named functions* in the lab and benchmark them head-to-head in the
  same binary. No build tags, no stash-swap needed — this is the default for §3/§4.
- **Whole-lexer A/B** (generic vs devirtualized): both entry points coexist in the
  package (no build tags), so one benchmark drives each — see §5.1.
- Retire the `git stash` swap methodology — it can't hold many variants at once
  and conflates the change under test with thermal drift.

### 2.2 Micro-benchmark suite ✅
Built: `workloads.Micro()` (12 single-path payloads, ~256 KiB each) +
`BenchmarkMicro` / `TestMicroWorkloads` in `json/benchmarks/lexers/`. The guard
test uses `jsontext.Walk` as an **independent RFC 8259 oracle** (confirmed e.g.
`-0.44e10` is legal) and drains ref+lab to EOF so no bench times a partial scan.

- **numbers** (new splits; `intElem`/`ints` in `All()` stays mixed-sign)
  - `ints_pos` ✅ · `ints_neg` ✅ · `decimals` (no exponent) ✅
  - `exponential` (`1e10`, `1E-10`, `-0.44e10`, neg mantissa) ✅
- **strings** (same generators as `All()`)
  - `strings_plain` ✅ · `strings_unicode` ✅ · `strings_escaped` ✅ ·
    `strings_escaped_long` ✅ · `strings_uescaped` ✅
- **short tokens** (split out of `bools_nulls`)
  - `nulls` ✅ · `bools` ✅ · `separators` (`[{},[],...]`, delimiter-dense) ✅

Each runs ref-vs-lab in three modes (`bytes` pull / `tokens` push / `reset`). Kept
separate from the heavy `Suite()` gauntlet so iteration stays fast; jsontext/
easyjson peers omitted here (§2.3). Method: `count=10`, same session, median, A vs
baseline in one binary. Decision rule: **a change ships only if it wins its target
micro-bench AND does not regress any other beyond noise; then it must also hold on
the real corpora (twitter, canada, citm, golang_source) before retrofit.**

**First signal (smoke):** `separators` ≈108 MB/s vs `ints_pos` ≈450+ MB/s — the
dispatch-dominated path is ~4× slower per byte, the clean probe for §5.1/§5.2.
`reset` = 0 allocs/op confirmed. Full count=10 baseline:
`ramblings/2026-06-micro-baseline.txt`.

### 2.3 Heavy jsontext comparisons — paused ⏸️
We know roughly where we stand (§1 + the prior string study). We stop running the
full vs-jsontext gauntlet every iteration; we resume it only at gate reviews,
when a lever is proven on micro + corpus.

---

## 3. SWAR foundation 🚧

**Verdict from the landscape survey:** keep hand-rolled fused SWAR; stdlib does
not make it moot (`IndexByte` = single-needle SIMD, great only for a lone needle;
`IndexAny` = scalar ASCII-bitmap, inapplicable; no `<0x20` primitive). The
`fredbi/swar` lib has sound, inlinable primitives (`HighBitWhereLess/Greater/Equal`,
byte arithmetic) but **no JSON composites and no digit-class** — illustrative, not
drop-in.

**Built:** `internal/swar` (in-tree, no external dep). All primitives exact,
multibyte-safe, and **inline-gated** (`TestInlinable` execs `go build -gcflags=-m`
and fails if any is not "can inline"). Exhaustive per-lane correctness vs a scalar
oracle (`TestStringStopMask`, …):

- `StringStopMask(w)` ✅ — fused `{<0x20, ==0x22, ==0x5c}`, lean ASCII-needle form.
- `MaskEqual` / `MaskLess` / `MaskGreater` ✅ — general exact comparators (the
  `fredbi/swar` byte-isolating + `0x7f` forms).
- `DigitMask` / `NonDigitMask` ✅ — exact "hasbetween" range test (self-contained
  so it inlines; the `MaskLess|MaskGreater` composition busted budget 80).
- `FirstByte(mask)` ✅ — `TrailingZeros>>3`, exact-locate the first flagged lane.

**Consolidation results (lab vs frozen reference, count=10):**
- String-stop routed through `swar.StringStopMask` (minimal change, original
  control flow): **perf-neutral** — escaped_long 1344 vs 1340, plain 951 vs 958,
  unicode 879 vs 873. Proves the helper inlines with zero cost; the bit-twiddling
  now has one tested home. ✅ shipped to lab.
- **Lead for §4.2:** also using `FirstByte` to exact-locate in the fast path lifted
  fast-path workloads **+7–14%** (plain +6.7%, unicode +13%, uescaped +14%) but
  **regressed escaped_long −12.5%** — not the algorithm (escaped_long is
  slow-path-dominated, fast path runs ~1 word) but codegen perturbation of the
  shared `consumeStringWhole`. Real fast-path win *if* we split fast/slow into
  separate funcs to insulate the slow-path codegen. Parked as a string experiment.

**Still ⬜ (algorithm experiments, not pure consolidation):**
- digit-run scan scalar→SWAR using `NonDigitMask` (`number.go:54,67,81`) → §4.1.
- whitespace skip (duplicated `generic.go:115-128,557-590`, `push.go:68`) — needs a
  whitespace mask `{0x20,0x09,0x0a,0x0d}`; niche, deferred.

**Inlining gate (standing rule):** `go build -gcflags=-m` must show every helper
inlined; a shared-but-not-inlined helper *regresses* the fast path (learned with
`scanStringStop` cost 98). Enforced for the module by `TestInlinable`.

---

## 4. Algorithms — optimize for less work

Identify shorter paths gated by a cheap SWAR probe; implement as separate
functions; pit against baseline on the targeted micro-bench, then on a mix, then
on real corpora.

### 4.1 Numbers ✅ (full-grammar inline fast path)
The whole-buffer fast path already covered `[-] int` (positive *and* negative — the
`minusSign` arm). Extended it to the **whole grammar inline** — `[-] int [frac]
[exp]` — so `consumeNumberWhole` (the slow full-grammar validator) is now reached
**only for malformed numbers** (error reporting). Done in both generic cores
(scanTokenG + scanPushG), regenerated, validated against the full **JSONTestSuite
conformance corpus** (lab ≡ reference on every number edge case).

Wins (lab vs frozen reference, `ramblings/2026-06-number-fastpath.txt`):
- decimals **+18% pull / +36% push**, exponential **+13.6% pull / +38% push**.
- ints flat pull (already fast), +11% push (devirt). pull = pure fast path; push =
  fast path × devirt compounding.

Lesson banked: a first cut fast-pathed decimals but *bailed to slow on exponent*,
which re-scanned int+frac → exponential −10%. **Never partial-scan then bail to a
re-scanner** — complete inline. Minor side-effect: `strings_escaped_long` pull
−3.8% (within the ~6% alignment-noise floor; the larger number block perturbs the
shared `scanTokenG` codegen — same fragility as the FirstByte finding).

SWAR digit scan (`swar.NonDigitMask` is ready) **not applied**: numbers are short
(< 8 bytes), the scalar run already wins big, SWAR setup likely costs more than it
saves. Deferred unless a long-number corpus says otherwise.

### 4.2 Strings ⬜ — minimize the eager-unescape tax
**Decided: keep eager unescape** (the "ready-to-use value" is the feature). The
game is using cheap SWAR probes to *avoid* unescape work, not to defer it. Keep
the zero-copy alias path (unaltered string) ✅. Split:
- pure ASCII / pure unescaped UTF-8 → SWAR finds the closing quote first → alias,
  no unescape (already done) ✅
- dense escapes + clean tail → clean-run batching shipped ✅
- general randomly-positioned escapes / surrogates → slow path ✅
- **fast/slow split + FirstByte exact-locate** ✅ — fast path uses `swar.FirstByte`
  to jump straight to the stop lane, and the unescape slow path is extracted into
  `consumeStringEscaped` so the two no longer share a frame. Banks the fast-path
  win **without** the earlier −12.5% `escaped_long` regression (which was codegen
  perturbation of the shared function): unicode +15.1%, uescaped +14.7%, plain
  +5.6% (pull); escaped/escaped_long flat. Push stacks devirt on top. Validated by
  full JSONTestSuite conformance + equivalence, race-clean. **Lesson: split fragile
  shared cores so a hot-path tweak can't regress a cold path** (same as the number
  block / consumeNumberWhole theme).
- **widen zero-copy to "alter but do not grow"** ⏸️ — an unescaped string is always
  ≤ source length, so we could unescape *in place* over the aliased region. Blocked:
  whole-buffer mode aliases the *caller's* data (`lexer.go:221`), so in-place
  mutation is unsafe without an opt-in own-the-buffer mode. Deferred.

Lazy/raw-token deferral is **out of scope** — it would shed the eager tax by
pushing work onto every consumer, which contradicts the contract we keep.

---

## 5. Tricks — inlining, switch cost, token size

### 5.1 Devirtualization by codegen (PREP) ✅ built — adoption pending
The tax: `scanTokenG[T,P]`/`scanPushG[T,P]` call `p.emit/p.none/p.eof` through the
generics dictionary (indirect, non-devirtualized). The generic cores are ~350
lines and will *never* inline; the point is **not** to inline them but to
**remove the indirect dictionary call** so the tiny `emit` body becomes a direct,
inlinable construction.

**Built:** `lab/internal/lexgen` generates `lab/scan_gen.go` (6 funcs: the 3 cores
× 2 policies), monomorphized — type params erased, policy bound concrete. Confirmed
via `gcflags=-m`: the generated cores **inline `semanticPolicy.emit/none/eof`**;
the generic cores show **no** such devirtualized inline (dictionary call). Devirt
proven **byte-for-byte equivalent** to generic (pull+push, L+VL, whole-buffer +
streaming, valid + malformed) by `TestDevirtEquivalence*`.

**Measured gap (in one binary — no cross-pkg alignment noise; count=8,
`ramblings/2026-06-devirt-lexgen.txt`):**
- **Push: +7…+18% everywhere** (separators +18, ints +17, exp +12, decimals/uesc
  +10, plain +9). The dict tax on push is far bigger than the assumed ~5% — the
  push core is hot/resident, every `yield(p.emit())` was a dictionary call.
- **Pull: mixed** — +2…+9% on dispatch/value-heavy paths, but **−4%/−2% on
  ints_pos/ints_neg** (number fast-path codegen shifts unfavourably when
  monomorphized; escaped_long flat). 

**ADOPTED — push (2026-06-27):** `Tokens()` (L and VL) routes through the devirt
push shims; lab push beats reference at the corpus level (separators +20%, ints
+14.5%, plain +8.7%).

**Pull regression RESOLVED (2026-06-27):** the original −4%/−1.8% on pull-ints was
real at the time (the inlined policy methods bloated the then-larger `scanTokenG`,
hurting the register-hungry number arm). The later "split fragile shared cores"
work (extracting `consumeStringEscaped` + restructuring the number fast path)
shrank `scanTokenG` and **cured it as a side effect** — devirt's frame is now 80 B
< generic's 88 B (no spills). Full pull sweep (in-binary, count=12, benchstat):
**+4.23% geomean, no regression** (ints +5.3%, separators +10.2%, strings +5-7%;
escaped/escaped_long flat). `ramblings/2026-06-devirt-pull-resolved.txt`. → **pull
is ready to adopt** (`NextToken` → `scanTokenSemantic`). Generic cores stay as
lexgen source-of-truth + A/B baseline (`*Generic` test helpers).

Design — port the writer's `writegen` (`json/writers/default-writer/internal/writegen/main.go`),
which lifts `commonWriter[T]` method bodies onto concrete receivers verbatim so
type-param dictionary calls become statically-typed inlinable calls; the generic
stays the single source of truth and the copies "cannot drift". For the lexer the
policy is a *parameter*, not a field, so the lift monomorphizes instead of
receiver-swapping:
- A `lexgen` generator parses `generic.go`, and for each instantiation
  (`token.T`/`semanticPolicy`, `token.VT`/`verbatimPolicy`) emits a plain concrete
  function — type params erased, `p` bound to a concrete value — into a separate
  `scan_gen.go` (DO-NOT-EDIT marker, idempotent strip+re-inject, like writegen).
  `semanticPolicy{}.emit(...)` is then a direct call the compiler inlines.
- **No build tags, no dispatch layer.** Both the generic core and the generated
  concrete functions stay in the package and are *both* reachable; the lab
  benchmark drives each (`benchGeneric` vs `benchDevirt`) to measure the gap in one
  binary. Once a winner, `NextToken`/`Tokens` call the generated funcs directly;
  generic remains source-of-truth for re-gen.
- Built **as prep** (per Fred): it is body-agnostic — as long as the cores keep the
  `(l *L, p P)` / `(l *L, p P, yield func(T) bool)` shape, every later algorithm
  change is re-lifted with one `go generate`, so experiments are cheap to compare.

PGO fallback (per Fred): for any helper that *would* benefit from real inlining
but busts budget, run the "is it worth it?" experiment by raising the budget via a
PGO profile; if it pays, mark it **needs-advanced-codegen** and **park** it — don't
build the advanced generator speculatively.

### 5.2 Dispatch: classification table vs binary-search switch ❌ REJECTED
**Verified (asm probe):** our switch on the lead byte does **not** become a jump
table — the JSON byte set is sparse (`"`=34 … `}`=125), so the compiler emits a
**binary search of ~4 `CMPB`** (one `JMP` for the contiguous digit range). That
looked like dispatch headroom, but the binary search turns out to be *already
near-optimal*.
- ❌ `[]func(*L) T` table — O(1) index but an **indirect call per token**: blocks
  inlining and re-adds the exact tax §5.1 removes. Never built.
- ❌ **classification table** `[256]tokClass` → dense `switch tokenClass[b]` (a
  compiler jump table) → direct arms. Built and measured (clean lab before/after,
  same package, count=10, benchstat, `ramblings/2026-06-classification-table.txt`):
  **−5.16% geomean B/s, slower on every workload** — separators −6.6%, nulls
  −5.4%, bools −5.1%, ints −10/−11%. Why it loses: the ~4 `CMPB` are cheap and
  **branch-predict extremely well** (e.g. separators alternating `{}[]`), while
  the table forces an **L1 load on every token's critical path + a computed jump
  that predicts worse**; and the larger dispatch perturbed the shared core's
  codegen (number arm −10%, the recurring fragility theme).
- **Conclusion:** the binary-search switch stays. Combined with §5.3, both
  per-token-*structure* levers (token size, dispatch shape) fail measurement — the
  switch and the 32 B token are already near-optimal. The real per-token win was
  **§5.1 devirt** (removing the indirect dictionary call), not restructuring it.

### 5.3 Token size ❌ REJECTED with measurement
- `T` today = **32 B** = 24 (`[]byte` header) + `valueDelimiter`+`kind`+`valueBool`
  (3 B) + **5 B trailing padding**. `VT` = **72 B**.
- **Union `uint64` idea: pointless** — the three small fields already fit the 8-byte
  tail; padding is *after* them, so hiding them in a `uint64` frees nothing.
- **Both 16 B representations REJECTED by a sizing microbench**
  (`compact_token_bench_test.go`, `ramblings/2026-06-compact-token-sizing.txt`,
  count=8). Three reps × {kind-only, read-value} consumer:

  | rep | kind-only vs tok32 | read-value vs tok32 |
  |---|---|---|
  | `*[]byte` + stable field | −17…−19% | −24…−26% |
  | `unsafe.Pointer` + `uint32` len | −1…+0.5% (**tie**) | **−6…−7%** |

  - The 24 B slice header carried **by value** in the 32 B token is nearly free —
    registers/stack, the copy is not the bottleneck.
  - `*[]byte` must **materialize the header into memory** (`l.value`) to take its
    address (24 B store/token) + deref per read — memory round-trip > the copy.
  - `unsafe.Pointer`+len (Fred's suggestion) removes the round-trip (pointer
    computed directly into the buffer, no backing field) → **ties** kind-only, but
    a real consumer must **rebuild the slice** via `unsafe.Slice` → −6-7%, and it
    adds GC/`unsafe` safety burden.
- **Conclusion:** no representation beats the 32 B by-value token in any consumer
  profile. Per-token cost is **dispatch, not token size**. Token stays 32 B; chase
  the dispatch floor via §5.1 (devirt, done) and §5.2 (classification table). Blast
  radius would also have been large (`token.T` in 14 pkgs, `MakeWithValue` 43 call
  sites) — the microbench killed it for ~1h of work, before any refactor.

---

## 6. Streams ⏸️ (after symmetric-path parity)
Only once `L` holds its ground on the symmetric paths. jsontext's **resumable**
state machine (`ConsumeStringResumable`, `ConsumeNumberResumable`, refetch on
`io.ErrUnexpectedEOF` recomputing `pos = absPos - baseOffset`) is the design to
study — our streaming string path is still byte-by-byte (`string.go:214-330`) and
does not share the whole-buffer fast paths. Trade-offs (buffer growth policy,
token-spanning-boundary, applying §3/§4 fast paths to refilled buffers) get their
own section when we get there.

---

## 7. Final validation — retrofit to the reference
For each proven lever, in order: (1) green on its micro-bench, (2) no micro
regressions, (3) holds on real corpora, (4) `go test ./... && go test -race`,
(5) cherry-pick/retrofit from the lab worktree onto master as the new reference,
(6) re-run the full vs-jsontext gauntlet at the gate and update
`default-lexer-roadmap` memory. Levers that fail any gate are recorded here as ❌
with the measurement, so we don't relitigate them.

---

## 8. Ordered worklist

**Prep (do first — tooling the experiments depend on):**
1. ✅ Micro-benchmark suite (§2.2) + count=10 baseline captured
   (`ramblings/2026-06-micro-baseline*`).
2. ✅ **Lab re-synced from the current reference** (option A — old unification
   sandbox discarded). 14 source files copied byte-for-byte (package rename only);
   equivalence tests pass; escaped_long 770→1365 (= reference), separators even.
   **Noise-floor finding:** with identical code, `decimals` still showed a stable
   ~6% lab-vs-reference gap — cross-package code-alignment/ordering noise, not a
   real delta. So lab-vs-reference carries a ~6% floor on some workloads;
   use same-binary pure-function A/B (§2.1) for algorithm calls where possible.
3. 🚧 In-tree SWAR module + inline gate (§3) — DONE: module built/tested/inline-gated,
   string-stop consolidated (perf-neutral). Remaining: digit-run SWAR (→§4.1),
   whitespace (deferred). FirstByte fast-path lead parked for §4.2.
4. ✅ `lexgen` devirt generator (§5.1) — built, generates 6 monomorphized cores,
   devirt proven equivalent + inlined. Gap: push +7-18%, pull mixed (ints -4%).
   Adoption pending: wire push→devirt; hold pull pending ints investigation.

**Experiments (each: win micro-bench → no regressions → hold on corpora):**
- ✅ Push-devirt adopted (§5.1) — Tokens() L+VL; lab beats reference +7-20% push.
- ✅ Numbers fast path (§4.1) — full-grammar inline; decimals +18/+36%, exp +13.6/+38%.
- ✅ Strings fast/slow split + FirstByte (§4.2) — unicode +15%, uescaped +14.7%, plain +5.6%; no escaped_long regression.
- ❌ Token 16-byte `*[]byte` / `unsafe.Pointer` (§5.3) — REJECTED with measurement (tie-to-−20%; memory round-trip / slice rebuild > by-value copy).
- ❌ Dispatch classification table (§5.2) — REJECTED with measurement (−5.16% geomean; binary-search switch predicts better, L1 load on critical path).
- ⏸️ Strings in-place no-grow unescape (§4.2) — blocked on buffer ownership.
- ⏸️ pull/ints devirt regression — open; NextToken stays generic.

**Takeaway:** per-token *structure* levers (token size, dispatch shape) both fail
measurement — the 32 B token + binary-search switch are near-optimal. The wins
came from doing *less work* (devirt, number/string fast paths), not restructuring.

**Later:**
- ⏸️ Streams (§6).
- ⬜ Gate review + retrofit to master (§7).

---

## 9. Resume plan (next session) — 2026-06-28

Today's gate review (§7, `ramblings/2026-06-gate-review.txt`) is a good stopping
point: WIN/parity on 4/5 real corpora + content-rich data; trails are understood
(tiny-token dispatch floor; eager-unescape by design). The methodology is settled —
**size the prize, measure in isolation, reject with data (Occam's razor)**.

Ordered by priority for the resume:

### 9.1 The unstable main loop ⚠️ (top concern)
Symptom seen ALL session: adding one line/MV to `scanTokenG` perturbs *unrelated*
arms (notably numbers) by several % — fragile, non-local. Theories: register
pressure (the giant function spills; amd64 has 16 GP regs), or inlining flipping on
an arm. **It doesn't feel right and it caps every other optimization.** Plan:
1. **Diagnose the mechanism** on a known-perturbing change (e.g. re-add a line to the
   number arm): diff `-gcflags=-m` (did an inline decision flip?), diff the STEXT
   `locals=` frame size (did it spill?), and diff the number arm's asm
   (`-gcflags=-S`) counting spill/reload (`MOVQ ...(SP)`). Pin spill-vs-inline-flip.
2. **Structural fix hypothesis: split the core.** Extract per-token-type handlers
   (number fast path, string, delimiter/structure) out of `scanTokenG` into
   functions so each arm's codegen is independent. We have strong precedent that
   splitting stabilizes (the string fast/slow split fixed BOTH the FirstByte
   regression AND, as a side effect, the devirt-pull regression). Success metric:
   after the split, adding a line to one arm no longer moves the others.
3. Keep the lexgen generator working through the split (cores stay the source of
   truth; helpers are shared methods, not monomorphized).

### 9.2 Push-core whitespace skip ⬜ (clear, small win)
The gate showed push trails pull on whitespace-heavy (citm push 74% vs pull 91%;
whitespace_heavy too) because `consumeWhitespace` is **pull-only**. Apply the same
batch skip to `scanPushG`'s semantic path (`i += consumeWhitespace(data[i:])`).
Expect citm push → pull-level. Low risk; do early.

### 9.3 AVX-512 SIMD via avo (refinement, test-and-see) ⬜
Idea (Fred): replace the 64-bit (8-byte) SWAR string-stop scan with avo-generated
AVX-512 asm — a 512-bit (64-byte) register gobbles long strings faster. Targets
LONG strings (already good: plain/escaped_long), so upside is uncertain — apply
Occam: ship only if it earns its keep. Notes:
- The asm call won't inline → only worth it where the per-call cost amortizes
  (long runs), NOT short strings. Likely a length-gated dispatch: SWAR for short,
  AVX-512 for long.
- Try the same for `consumeWhitespace` (long indent runs), though the compiler's
  tight scalar loop is already near-optimal — lower expected payoff.
- Reference: Fred has an example repo (AVX-512-from-avo, a SIMD-accelerated grep).
- Measure on `strings_plain`, `strings_escaped_long`, `whitespace_heavy`, citm;
  needs CPU AVX-512 support check + a scalar fallback build tag.

### 9.4 Retrofit lab → master (§7) ⬜
Once 9.1 (and maybe 9.2) land, cherry-pick the proven lab gains onto the reference
`default-lexer`: devirt (push+pull) via lexgen, number full-grammar fast path,
string fast/slow split + FirstByte, adaptive-SWAR slow path, jsontext-style
whitespace skip, drop semantic line/col (position → verbatim only). Re-run the gate
on master to confirm parity holds post-retrofit.

### 9.5 Publish the benchmark chart ⬜
Produce barcharts (lab vs jsontext vs reference vs easyjson) from the gate
benchmark JSON, using Fred's `benchviz` tool (as done for the default-writer). Wire
the gate run to emit benchviz-consumable output.

### Parked / not chasing
- Tiny-token dispatch floor (bools_nulls 36%): jsontext is leaner per token; the
  per-token structure levers (token size, classification table) already failed
  measurement. Pull's gap is per-NextToken-call overhead (push is near-parity).
  Revisit only if 9.1 (the split) opens new room.
- Dense escaped strings (63%): structural (eager unescape is the feature). Not chasing.
