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

### STATE at 2026-07-02 session end (parity ACHIEVED — remaining work is upside)
The core programme is **done**: L (semantic) leads json/v2 on **13/16** workloads
(extended simdjson-go corpus), the unstable-loop itch (§9.1) is diagnosed at the
register level and the semantic cores are stable, and both remaining losses are
principled design costs (tiny-token floor, eager-unescape). Rebased onto master.
Deliverables: `json/lexers/benchmark` 4-scenario benchviz chart; AVX2 scanner
preserved in `internal/simd`. Everything below §9.2 is either DONE or PARKED.
(2026-07-20 UPDATE: arc 1 shipped — AVX2 gate now in `internal/strscan`; the
`internal/simd` experiment module was folded into it and deleted. See §9.3.)

**The three forward arcs (Fred, session end) — each a self-contained next session:**
1. ✅ **Leave competitors in the dust on string-heavy workloads** → DONE 2026-07-20:
   the AVX2 string-stop gate is shipped (§9.3), full-corpus geomean +5.8%, gsoc
   +40%/+26%, azure +9.7%/+14%. The 6–14× isolated kernel now pays off end-to-end
   behind the `guessLong=16` clean-bytes gate. Next widener: AVX-512 on capable silicon.
2. **Make VL catch up** → §9.6 Tier 1 (alias blanks whole-buffer + batch-skip-with-
   newline-count = the VL analog of the §9.2 win) then Tier 2 (register de-pressure).
   (The AVX2 gate already lifted VL a bit: gsoc VL +26%, azure VL +4%.)
3. **Make stream-mode catch up with in-memory** → §6 Streams: the buffer-refill
   path copies where whole-buffer aliases zero-copy — the deepest, "final" arc.

Ordered by priority for the resume:

### 9.1 The unstable main loop — DIAGNOSED at the register level (2026-07-02)
Symptom seen ALL session: adding one line to `scanTokenG`/`scanPushG` perturbs
*unrelated* arms (notably numbers) by several %. **Mechanism CONFIRMED via
disassembly** (method: `-gcflags=-S` → isolate the fn by its `ABIInternal`
boundary → normalize away addresses/jump-targets/SP-offsets/temp-ids → diff
mnemonic+register operands; NOT just frame-size/spill proxies):
- **Root cause = register saturation.** The push main loop references ALL 14
  allocatable GP regs (AX BX CX DX SI DI BP R8–R13 R15; only R14/g reserved). Zero
  headroom → historically any addition forced reallocation somewhere.
- **The line/col decision was the lever.** When the semantic core still tracked
  line/col, `l.line`/`l.lineStart` were live across EVERY arm (per-token position),
  holding ~2 regs hostage globally → no headroom → the §9.2 whitespace batch-skip
  regressed numbers ~-6.35% (why it was reverted). Dropping line/col (§9.4/§5) freed
  those regs. PROOF at the register level: re-adding the batch-skip now adds only
  whitespace-local instrs (CMPB $9/$10/$13/$32 + SWAR mask on R9–R13); the number/
  string/delimiter arms are BYTE-IDENTICAL, spills 72→72. Clean (§9.2 shipped).
- **Spill contrast = the headroom map.** semantic pull core 12 spills ($32 frame),
  semantic push 72 ($88); VERBATIM pull 130 ($256), verbatim push 208 ($336). The
  verbatim cores are the register-starved ones (position + blanks tracking live
  across the loop) — that's where a future addition WILL still ripple.
Takeaways: (a) the semantic cores are now stable — the instability was position
tracking eating the register budget, and we already removed it; (b) remaining §9.1
risk is verbatim-only. Structural options if verbatim needs de-pressuring: split
per-token-type handlers out of the core into functions (precedent: string fast/slow
split fixed FirstByte + devirt-pull), or move blanks/position off hot registers.
lexgen must keep working through any split (cores stay source of truth).

### 9.2 Push-core whitespace skip ✅ (done 2026-07-02, commit 684aef1)
`scanPushG` walked whitespace byte-by-byte; routed the SEMANTIC push path into the
same `i += consumeWhitespace(data[i:])` batch-skip the pull core uses, gated by the
existing `tracksPosition()` compile-time constant (verbatim keeps per-byte: it must
accumulate the preceding-blanks slice + count lines). RESULT: citm push +28%
(1040→1334 MB/s), canada/twitter neutral-to-positive, pull core BYTE-IDENTICAL.
This is the change that regressed unrelated arms before (§9.1) — clean NOW, and the
register-level proof IS the §9.1 finding below.

### 9.3 SIMD via avo — AVX2 string-stop gate ✅ SHIPPED (2026-07-20)
REVIVED from PARKED (Fred: "revive the AVX2 gate with the SWAR-probe heuristic; if
not satisfactory, replace the heuristic by an options knob"). It IS satisfactory
and is now wired into all four whole-buffer string scans. **Full-corpus geomean
+5.8% throughput (L+VL, 18 workloads), zero alloc change**, big wins on the target
(go-openapi/azure L +9.7% / pretty +14%, gsoc +40%/+26%, github, numbers +13%,
mesh +12%, canada +9.9%). One soft spot: **citm VL −3.6%** (whitespace-heavy, NO
string value ≥64 B — AVX2 can only add overhead there; high-variance workload we
already dominate). Left as-is; the options knob is the lever if a caller needs it.

**The gate (as built).** Package `internal/strscan` (part of the json module, no
avo dependency). `ScanStop(data) int` = AVX2 kernel when the CPU supports it and
the slice is ≥ `avx2Min` (32 B), else the same 8-byte SWAR loop the inline probe
uses; identical stop semantics everywhere (reuses `internal/swar`). AVX2 support is
detected once via a hand-written CPUID+XGETBV `.s` (OSXSAVE+AVX+XCR0 YMM + leaf-7
AVX2) — **no `x/sys/cpu` dependency**, cross-checked against `/proc/cpuinfo` in a
test. Non-amd64 builds get a SWAR-only `ScanStop` (build-tagged).

**Two heuristic lessons (both measured, both cost real % before they were found):**
1. **The gate signal is clean-bytes-seen, NOT slice length.** `ScanStop` receives
   the *buffer remainder* (huge mid-document), so its own length guard can't tell a
   short value from a long one. The real "guess long" signal is how many clean
   leading bytes the inline probe already saw → `const guessLong` (bytes; **16** is
   the swept sweet spot). Only after `guessLong` clean bytes do we delegate.
2. **Keep the call OUT of the per-word loop (this is the §9.1 lesson again).** A
   first cut delegated *inside* the SWAR word loop — geomean +4.4% but a real
   regression cluster (citm L −4.4%, instruments/apache/golang/payload all down):
   the mere presence of a call in the loop body pessimised its register allocation
   for every short string that never reached it. Hoisting the call to *after* the
   loop recovered all of them (geomean +4.4%→+5.8%) with the SAME wins. The tight
   short-string loop must stay call-free.

**Sweep (full corpus, hoisted):** guessLong 8 → geomean +5.73% but citm VL −5.1%,
instruments VL −2.1%; **16 → +5.76%, citm VL −3.6%, instruments VL −0.8% (chosen)**;
32 → +4.58%, adds apache −2%. 8 maximises string-heavy wins but regresses the
short-string workloads; 16 keeps ~all the win and clears the regressions bar citm.

**Module layout.** `internal/strscan/` ships the kernel (`stringstop_amd64.s` +
`scan_amd64.go` stub/gate + `scan_noasm.go` + CPUID `.s` + `detect_amd64.go` +
`strscan.go` SWAR core + oracle/bench tests). The avo generator lives in
`internal/strscan/_asm/` — its own module (`replace`→local avo) under a
`_`-prefixed dir the parent go tool ignores, run via `go generate ./internal/strscan`
(regeneration reproduces the committed `.s` byte-identically). `internal/simd` (the
old parked experiment module) was folded into this and deleted. avo never enters
`json/go.mod` (verified: `go mod tidy` adds nothing; `go list -deps` has no avo).

**Corpus.** Added `azure_swagger[.pretty]` — 16 go-openapi/analysis Azure Network
specs merged into one document (30% of bytes in string values ≥32 B; short keys,
long descriptions) — the go-openapi mammoth-spec profile Fred flagged as important.

**WithoutAVX2 knob ✅ (2026-07-20).** `WithoutAVX2(bool)` option on both L and VL:
when set, the fast-path `guard` is pushed past the buffer (`n+1`) so the inline SWAR
word loop scans the whole value and never delegates — a pure, CPU-independent SWAR
path, no vector call anywhere. Equivalence test proves the token stream is identical
on/off for L and VL. HONEST FINDING (measured): the knob is **perf-neutral on citm**
(on 3.48ms / off 3.49ms) — citm's −3.6% is the per-word `i >= guard` COMPARE in the
hot loop (structural, present regardless of the knob), NOT the AVX2 path or the
delegation. So the knob is a determinism / differential-testing / safety switch, not
a citm recovery lever; the doc says so. Not worth a separate guard-free loop to claw
back citm's 3.6% (the guard branch is what buys +5.8% everywhere else).

Follow-ups (parked): AVX-512 width-doubling on capable silicon; the escaped-path
clean-run delegation (sites 2 & 4) still calls in-loop — fine for the escaped
workloads measured (twitterescaped +2.8%), hoist it too if it ever shows up.
Original AVX-512 rationale kept below.

Idea (Fred): replace the 64-bit (8-byte) SWAR string-stop scan with avo-generated
AVX-512 asm — 512-bit (64-byte) registers gobble long strings faster. TWO payoffs,
and they connect to the unstable loop (9.1):
1. **Speed (long inputs):** 64 bytes/iter vs 8. Targets LONG strings (already good:
   plain/escaped_long) → upside uncertain; Occam: ship only if it earns its keep.
2. **Register / loop relief (the 9.1 angle, per Fred):** the asm uses the ZMM vector
   registers (zmm0–zmm31), SEPARATE from the 16 GP registers the main loop contends
   for; and Go does NOT inline assembly, so the scan moves OUT of `scanTokenG`,
   shrinking it. Both directly attack the clogged-registers / fragile-loop itch.

**Occam guard (do during 9.1):** the *shrink-the-loop* effect does NOT require
AVX-512 — a plain `//go:noinline` Go extraction of the scan also shrinks the core.
So test a noinline-Go extraction as the CONTROL: if that alone stabilizes the loop
(adding a line stops perturbing other arms), the stability win is "smaller core",
not "AVX-512", and asm becomes a *pure speed* add-on for long strings. (Caveat from
this session: noinline `consumeWhitespace` cost citm ~6% per-call overhead — so
extraction trades inline-perturbation for call cost; asm must beat that with faster
scanning to win.)

Notes:
- Won't inline → length-gated dispatch: SWAR for short, AVX-512 for long.
- Same idea for `consumeWhitespace` (long indent runs); lower expected payoff (the
  compiler's tight scalar loop is already near-optimal).
- Reference: Fred has an avo AVX-512 example repo (SIMD-accelerated grep).
- Measure on `strings_plain`, `strings_escaped_long`, `whitespace_heavy`, citm;
  needs an AVX-512 CPU-feature check + a scalar fallback (build tag / runtime guard).

### 9.4 Retrofit lab → reference ✅ (done 2026-07-02, ahead of 9.1)
Promoted the whole first-workable lab version onto the reference `default-lexer`
(Fred: "reap the benefit … before the next lab experiment"). Done ahead of 9.1 so
the next lab experiment forks from a devirt'd baseline. What moved (all impl files
byte-identical to lab modulo `package lab`→`package lexer`):
- generic.go (tracksPosition gating, consumeWhitespace batch-skip, full number +
  string fast paths), iterator.go (push → devirt shims), lexer.go (NextToken →
  scanTokenSemantic; semantic L.Line()/Column() REMOVED), number.go, string.go
  (fast/slow split + adaptive-SWAR), verbatim.go (VL.Line()/Column() added).
- NEW: devirt.go (noinline push shims), scan_gen.go (REGENERATED in-place; byte-
  identical to lab), internal/lexgen/ (generator, header pkg → lexer).
- `internal/swar` was already in the reference — kept as-is (lab imported it).
- Tests fixed for the dropped semantic position: lexer_test.go
  (TestResetWithBytesReuse drops line/col), position_test.go (rewritten
  verbatim-only). Brought devirt_test.go + devirt_bench_test.go as the
  generic↔generated equivalence guard (scan_gen.go is generated → must not drift).
- VERIFIED: build/vet/test/`-race` green; `go generate` idempotent (scan_gen.go
  unchanged on re-run); devirt cores inline (gcflags -m); no external consumer used
  semantic Line()/Column(); json module + direct consumers (json, nodes/light,
  benchmarks) build+pass. Bench confirms win carried: twitter +54% vs jsontext,
  citm_min ahead, citm pull near-parity; default-lexer/* == lab/* within noise.
- lab/ left in place (stale duplicate of reference now) as the base to re-sync for
  the NEXT experiment — re-sync when that starts, not now.

### 9.4b Verbatim-token API hardening ✅ (done 2026-07-02, pre-lab prep)
Three API/behaviour fixes Fred asked to lock in before the next lab session:
1. Position accessors: `token.VT.Col()` → `VT.Column()` (renamed, matches
   `VL.Column`); semantic `L` still has none. Call sites updated (handoff/position
   tests).
2. Verbatim strings are now kept RAW (escapes intact) instead of eagerly decoded —
   the documented `[VT]` contract that the impl had been violating (a
   round-tripping bug: `A` had re-emitted as `A`). `VT.Value()` = raw source;
   new `VT.Unescaped() []byte` / `VT.UnescapedString() string` decode on demand
   (validated at scan time, so no error path). New raw scanners `consumeStringRaw*`
   (string_raw.go) validate every escape but don't materialise — whole-buffer
   aliases (zero copy), streaming copies raw; adaptive SWAR clean-run skip. Decoder
   lives standalone in the token pkg (token/unescape.go, no lexer dep).
   KEY: dispatch is inside `consumeString` on `l.trackBlanks` (the VL flag), NOT in
   the shared scan core — so generic.go/scan_gen.go are BYTE-IDENTICAL to baseline
   (routing it through the policy or an inline core branch perturbed the semantic
   core's escape analysis → +1 alloc under -race; §9.1 fragility, avoided). Writer
   ripple fixed: default-writer `VerbatimToken` writes raw string values between
   quotes WITHOUT re-escaping (would double-encode); added the missing byte-exact
   round-trip test (writer_test TODO). Store round-trip verified.
3. `WithElideSeparator` now honoured by VL (default false / elide-off for faithful
   round-trip; caller may opt in) — was hard-clobbered in VL.reset(); fixed via a
   prepended default option + dropping the clobber.
   PERF: verbatim raw on escaped_long 1140→1846 MB/s (SWAR clean-run skip), now
   BEATS jsontext (1661) with fewer allocs (352B/4 vs old 968B/6, no decode buffer).
   VERIFIED: build/vet/`-race` green across token/default-lexer/default-writer/
   default-store + json/nodes; generate idempotent; semantic core unchanged.

### 9.5 Publish the benchmark chart ✅ (done 2026-07-02)
New self-contained module `json/lexers/benchmark` (mirrors `json/writers/benchmark`):
4-way input-throughput comparison per corpus — L (semantic), VL (verbatim),
easyjson jlexer, json/v2 jsontext. `benchviz/throughput.png` rendered with Fred's
benchviz (vintage theme, median of 6 runs); `benchviz/README.md` carries the
allocation table (ours: doc-size-independent few-alloc; easyjson: 17k–102k
allocs/op; jsontext: 26–262). Registered in go.work; standalone go.sum. Findings:
L leads twitter/golang/canada (and citm after §9.2 — 1315 vs jsontext 1172); VL
trades throughput for round-trip fidelity (blanks + line/col + raw strings).

### 9.6 VT-only de-pressuring stream ⏸️ PARKED (resume later)
The §9.1 register saturation is resolved for the SEMANTIC path (the load-bearing
one); the VERBATIM cores remain register-starved (pull 130 spills/$256, push 208/
$336) because position + blanks tracking is live across the loop. These levers
would speed VL and de-pressure its cores. Parked by Fred 2026-07-02 — VL is the
round-trip lane (formatters/linters/byte-faithful re-emit); pursue only if VL
throughput is on a hot path. Scope: 1/2/4 are VT-only (semantic byte-identical);
3 reaches the shared core (would put a call into scanTokenSemantic too → must be
re-validated on semantic). Tier 1 = the perf win, Tier 2 = the register fix.
- **Tier 1 (speed; also shrinks the loop body):**
  1. Alias the blanks in whole-buffer mode (`data[blankStart:i:i]`) instead of the
     current per-byte `append` — the push core already does this; bring it to pull.
     Removes a growing live slice + the per-byte copy. (Streaming still copies.)
  2. Batch-skip the whitespace run while counting newlines in ONE pass — a
     `consumeWhitespaceCounting(data) (n, newlines, lastNLoffset)` (SWAR popcount of
     newlines over the same words). Updates l.line/l.lineStart once per run, not per
     byte. This is the VL analog of §9.2; expected to close most of the citm-VL gap
     (524 → toward L's level).
- **Tier 2 (actual register de-pressuring):**
  4. Non-inline the VT emit: move `AsVerbatim().WithPosition()` behind a
     //go:noinline shim so the wider-token construction leaves the core (same lever
     as §9.3 "asm never inlines → shrinks the caller"). One-line change, measurable
     frame delta — do this BEFORE #3 to see if shrinking suffices.
  3. Split the inline value arms (number fast path, delimiter/structure) out of the
     core into methods so they stop sharing registers with the position state.
     SHARED-core change (hits semantic too — re-validate). Precedent: the string
     fast/slow split fixed FirstByte + devirt-pull.
- NOT viable: fully-lazy line/col — `line` is a cumulative running sum across
  tokens (can't be made loop-free without O(n) rescan); column could derive from
  blanks but line can't. Position stays a running counter; Tier 1 makes it cheap.

### Parked / not chasing — the two understood floors (2026-07-02, CLOSED)
Extended-corpus study (16 workloads, simdjson-go set) confirmed we generalize:
**L beats jsontext on 13/16**, including untrained string-heavy payloads
(update-center 1.62×, gsoc-2018 1.52×). The 3 losses are BOTH deliberate design
choices, fully diagnosed — accepted, not chased (Fred: "we're good"):

- **The tiny-token floor** (mesh/mesh.pretty 0.93×; also bools/nulls ~36%, single-
  char strings). ROOT CAUSE nailed by categorize→profile: mesh is 56% short
  integers at 2× canada's number density; profiling the push core shows ~18% of
  time is the by-value 32-byte token being constructed (MakeWithValue 280ms) +
  copied 3× through emit/yield (360ms) PER token — poorly amortized when tokens are
  1–3 bytes. NOT a missing integer fast-path (that exists, generic.go:435 — valid
  ints emit without touching consumeNumberWhole). It's the PRICE of the zero-alloc
  by-value token model. Fixes rejected: by-pointer emit → token escapes to heap,
  destroys zero-alloc (worse); 16-byte token (§5.3) → −17–26% on canada/citm/
  twitter. The two token verdicts RECONCILED: small token moves cost from copy
  (dominates on dense tiny tokens) to value-access (dominates on value-heavy
  corpora we win) — a net trade, not a win. The floor is what BUYS us 13/16 + near-
  zero allocs (jsontext 26–262 allocs/op, easyjson 17k–102k). Not chasing.
- **Dense escaped strings** (twitterescaped 0.93×): eager-unescape is the feature
  (Go-consumable string values, no re-reading RFC 8259 escaping). Mostly synthetic
  in real data. Not chasing.

### AVX2 string-scan — ✅ SHIPPED 2026-07-20 (was proven-but-unshipped 2026-07-02)
No longer parked — revived and wired in as the `guessLong=16` gate; see §9.3 for the
shipped design, results (+5.8% geomean), and the two heuristic lessons. The original
parked note is kept below for the reasoning that led here.

Isolated experiment (avo → AVX2, runs on this Zen 3 box; kernel correct vs oracle
incl. high-byte safety): string-stop scan is **6–14× vs SWAR for strings ≥64 B**,
loses <32 B (broadcast setup), crossover at 32 B → length-gated. Prize-sizing:
marginal on the parity 4-set (only twitter has long strings), but the extended
corpus flips it — gsoc-2018 80% / github_events 52% / update-center 37% of bytes in
string values ≥32. HOWEVER we ALREADY win those without it (gsoc 1.52×), and the
§9.1 register-relief rationale does NOT apply to strings (string scan is already
out of the main loop). So AVX2 is a MARGIN-WIDENER on payloads we already lead, not
a necessity + a permanent amd64 asm/CPU-gate/fallback surface. PARKED; pull off the
shelf only for a big-text-document-lexing consumer. Fred's adaptive gate (SWAR-
probe first, AVX2 only if the first word finds no closing quote) is the right shape
if revived. Experiment code was throwaway (avo examples dir).

---

## 10. Stream/buffer pull-core split ⏳ (2026-07-20 — streams catch-up)

**Decision (Fred):** the semantic pull core `scanTokenG` is split into two
policy-parameterized cores so the stream lane can be optimized *without perturbing
the benchmarked whole-buffer champion*. The buffer speed-up from stripping the
streaming-generality is expected to be *small* (the `l.wholeBuffer` runtime branches
predict near-perfectly and are per-token, not per-byte); **the prize is decoupling /
insulation**, not a buffer win. Sizing the buffer bonus is skipped — the decoupling
alone justifies the split (§9.1 discipline: don't edit the delicate loop that
produces the 16/18 wins).

### 10.1 Shape
Fork `scanTokenG` → two generic cores, both still fed through lexgen:

- **`scanTokenBufferG[T,P]`** — whole-buffer lane. Modeled on `scanPushG` (the
  proven buffer shape): **local cursor `i`, NO `readMore`**, whole-buffer value
  consumers, inline number fast path *unconditional* (no `l.wholeBuffer &&` gate),
  and **zero-copy preceding blanks** (`data[blankStart:i:i]`, like push) instead of
  the byte-by-byte `l.blanks` append. Returns one token; resumes from `l.consumed`
  next call; `continue`s only past elided separators + whitespace. Trailing blanks
  for the verbatim EOF token are sliced zero-copy before `errCheckG(io.EOF)`.
- **`scanTokenStreamG[T,P]`** — io.Reader lane = today's `scanTokenG`, but with the
  dead whole-buffer number fast path removed (always `consumeNumberStreaming`), so
  it is the small, clean surface we iterate on. Keeps the `for{ readMore(); … }`
  outer loop, struct cursor, streaming consumers.

`NextToken` (L) / `VL.NextToken` dispatch **once** on `l.wholeBuffer` — one
predictable branch per token, one `L` type, pooling/Reset untouched.

lexgen `cores`: `scanTokenG` entry → `scanTokenBufferG` + `scanTokenStreamG`
(`scanPushG`, `errCheckG` unchanged) ⇒ generated `scanTokenBuffer{Semantic,Verbatim}`,
`scanTokenStream{Semantic,Verbatim}`. `devirt_test.go::nextTokenGeneric` dispatches
the same way so the generic↔devirt equivalence guard covers both lanes.

### 10.2 Correctness net
`TestDevirtEquivalencePull` already runs each input in **both** whole-buffer and
tiny-buffer streaming mode; plus push/pull equivalence, conformance, security,
zerocopy, race. The buffer core is a yield→return transform of `scanPushG`, so
`TestDevirtEquivalencePush` (push vs whole-buffer pull) is an extra cross-check.

### 10.3 Stream lane optimization — DESIGN AGREED (2026-07-20, debated w/ Fred)

**End state:** *streaming = the same fast scanner run over a sliding window.* The
byte-by-byte streaming consumers become the thing we delete, not maintain.

**Cost model.** Two copies today: (1) the Read copy (reader→buffer) — unavoidable;
(2) the value copy (buffer→currentValue, byte-by-byte in
`consumeStringStreaming`/`consumeNumberStreaming`) — avoidable for most tokens. Plus
two speed gaps vs buffer mode: per-byte end-of-buffer checks + struct cursor, and no
SWAR/AVX2/bulk-copy on the value consumers. Key fact: most tokens sit entirely inside
the current buffer (token ≪ window), so for them streaming should behave *identically*
to buffer mode.

**Central lever — optimistic in-buffer scan.** Scan `l.buffer[:l.bufferized]` with a
local cursor and the buffer-mode fast paths. The ONLY difference from buffer mode is
the terminal condition: reaching `bufferized` means *"maybe need more"* (refill+
resume), not end-of-input. Tokens that complete in-buffer take the fast path and
ALIAS `l.buffer` zero-copy (valid until next refill — same lifespan as the
currentValue-reuse contract; Fred confirmed aliasing is within the contract). End-of-
buffer is then checked once per SWAR word + once per token, not per byte. This is
jsontext's `*Resumable` idea (§6 flagged it). The subtlety: `bufferized ≠ EOF`, so we
canNOT reuse `consumeStringWhole` verbatim (it errors ErrUnterminatedString at buffer
end) — the fast path needs a "reached bufferized → refill/delegate" branch.

**Buffer-size guard rail (Fred):** round `WithBufferSize` capacity UP to a multiple of
the max scan horizon (32 B — covers the 8 B SWAR word and the 32 B AVX2 stride) for
clean tiling + horizon-aligned slide arithmetic. Note: this pins CAPACITY; `bufferized`
(what a given Read returned) is reader-decided and not horizon-aligned, so the fast-path
scalar tail stays.

**Two models for a token that spans a refill (starts at `s`, scan hits `bufferized`):**
- *copy-into-current (A, today):* copy `buffer[s:bufferized]` OUT into currentValue,
  refill from 0, continue BYTE-BY-BYTE appending; value = a copy in currentValue. Two
  scanners (fast + streaming). Spanning tokens never zero-copy, never fast. Trivial refill.
- *slide+grow (B, chosen end state):* memmove `buffer[s:bufferized]` to the FRONT
  (overwriting already-emitted `[0:s]`), Read AFTER it, continue the SAME fast scan over
  the now-contiguous token; value = buffer alias (zero copy), fast-path throughout, even
  a `\uXXXX` split across the boundary becomes contiguous. ONE scanner. Grow only when a
  single token exceeds capacity. Per-spanning move cost ≈ same as A (both move the partial
  once); B's prize is architectural (one scanner; retire the streaming consumers) + stays
  fast on small buffers.

**THE INVARIANT (Fred — prevents an unbounded/ratcheting buffer):** capacity and read
size are INDEPENDENT. Read/window size stays = `bufferSize` (what we request per Read,
the working set between tokens). Capacity = spare headroom that grows ONLY when one
token's contiguous span outruns it, retained across refills/Reset, NEVER the amount we
read to fill. **Grow is triggered by a token outrunning capacity — never by the buffer
merely being full after a read** (else "full" fires every refill → unbounded). Windowing
reads to `bufferSize` is what makes "full" a reliable "this token needs more room" signal.
⇒ steady-state residency ≈ `bufferSize` + at most one in-flight oversized token; capacity
converges to the LARGEST SINGLE TOKEN, not the stream length. During a normal refill:
slide the small unconsumed remainder to front, read `bufferSize` into the free space;
only while consuming one over-window token does the window extend into spare (grow if it
exceeds cap); on token completion the window snaps back to `bufferSize`.

**Phasing:**
- **Phase 1 (delegation, low risk, ship+measure):** add the optimistic in-buffer fast
  path; fast-path only the CLEAN, in-buffer common case (plain string w/ closing quote in
  buffer → SWAR/AVX2 + alias; number terminated in buffer → inline + alias; true/false/
  null via existing consumeN). ANY escape, or scan reaches `bufferized` → delegate to the
  EXISTING streaming consumers (Model A). No rewrite of the hard spanning logic.
- **Phase 2 (slide+grow, Model B, under the invariant):** replace the delegation fallback
  with slide+grow so spanning tokens stay fast+zero-copy; then retire
  `consumeStringStreaming`/`consumeNumberStreaming`. Earns its way in only if measured
  spanning / small-buffer cost is real.

VL symmetry (§9.6) folds in — buffer core already gives verbatim zero-copy blanks; a
spanning blank run is the verbatim analogue of a spanning value (slide handles it too).

**Baseline gap (measured 2026-07-20, BenchmarkStreamGap, L.NextToken, stream as % of
buffer throughput — lower = bigger gap):**

    STRING-HEAVY (the prize, = go-openapi target):
      gsoc-2018       6.9%  (14.5x slower!)   update-center  25%   github_events 29%
      azure_swagger   29%   azure.pretty      32%   numbers 32%   apache 37%
    STRUCTURE/NUMBER (less bad, still ~2x):
      citm 60%   marine_ik 52%   instruments 50%   mesh 42%   canada 40%   twitter 37%

**Two diagnoses that sharpen the plan:**
1. **4 KB ≈ 64 KB throughput everywhere** (e.g. azure 29% vs 28%, citm 60% vs 61%). The
   gap is NOT refill frequency — it is the PER-TOKEN in-buffer work (byte-by-byte value
   consumers + struct cursor + the avoidable copy), which is paid regardless of buffer
   size. ⇒ **Phase 1 (optimistic in-buffer fast path) is the prize; bigger buffers and
   Phase 2 slide+grow are secondary** (spanning tokens are not the main cost).
2. **String-heavy bleeds most** — buffer mode runs SWAR/AVX2 + zero-copy alias; streaming
   crawls byte-by-byte + copy. gsoc buffer 2631 MB/s → stream 181. Phase 1 fast-pathing
   clean in-buffer strings should recover the bulk, exactly on the azure/go-openapi target.

Measurement harness: `json/lexers/benchmark/streamgap_bench_test.go` (THROWAWAY, imports
the lab; kept to measure Phase 1 before/after; delete before any retrofit to reference).

**Status:** ✅ split landed in the lab (build/vet/test/-race green, generate
idempotent, all equivalence guards pass — buffer-pull vs generic vs push, both
lanes in `TestDevirtEquivalencePull`). `maxValueBytes` parity restored in the
buffer core (numbers route to the bound-enforcing streaming consumer; verbatim
blanks-flood checked once at the token boundary + at EOF — the zero-copy-blanks
equivalents of the stream core's per-byte checks).

**Measured (unexpected upside):** the buffer-pull lane came out *faster* than the
old combined core, not merely insulated — interleaved+warmed A/B (lab devirt/pull
vs reference devirt/pull, Micro workloads): ints_pos −8.6%, decimals −7.0%,
strings_unicode −2.8%, strings_plain −2.1%, nulls +0.9% (noise). The number wins
exceed the ~6% cross-package alignment floor and match the mechanism (buffer lane
drops the per-token readMore + the `l.wholeBuffer` runtime gate + the per-byte
struct cursor); strings within noise. 0 allocs throughout. Push path unchanged.

**Next:** iterate on the stream lane (§10.3) — first size the actual stream-vs-buffer
gap on the corpus, then apply refill-side levers there without touching this core.

### 10.3a Phase 1a RESULTS — streaming clean-string fast path (2026-07-20)

Implemented `consumeStringStreamFast` (lab, semantic L): optimistic in-buffer SWAR/
AVX2 scan of the window; a CLEAN string that completes inside the window aliases
`l.buffer` zero-copy; ANY escape or window-end (span) → delegate to the existing
`consumeStringStreaming`. Relative-offset advances (streaming `l.offset` is absolute,
`l.consumed` is the window index). Correctness: new `TestStreamFastEquivalence`
(buffer-vs-stream, kinds+values, 16 buffer sizes 1..1024) + `TestStreamFastAliasesWindow`
(proves the alias); full suite + -race green.

Surfaced + fixed a PRE-EXISTING `consumeN` bug (unrelated to strings): the partial-
refill branch advanced `l.consumed` by the whole window `delta` instead of the bytes
actually needed, dropping the surplus — a literal (`true`/`null`) read through a window
SMALLER than the literal lost its trailing separator. Only reachable at bufsize<literal
(≤5); found by testing bufsize 2. Fixed (`delta < need`, advance by `need`).

Gap recovery (stream-4k as % of buffer throughput, baseline → Phase 1a):

    azure_swagger      29% → 89%  (+60pp)   ← the go-openapi target, gap nearly closed
    azure.pretty       32% → 83%  (+52pp)
    apache_builds      37% → 84%  (+47pp)
    update-center      26% → 64%  (+38pp)   payload-large 39%→76%   random 45%→81%
    github_events      29% → 66%  (+37pp)   twitter 37%→67%   twitterescaped 62%→87%
    citm 60%→73%  instruments 50%→72%  golang 40%→61%

Laggards, decomposed (each a known next lever, NOT a mystery):
  - NUMBER-heavy barely moved (strings-only lever): numbers 32%→37%, canada 40%→36%,
    mesh 42%→41%, marine_ik 52%→51%. → Phase 1b: streaming number fast path.
  - gsoc-2018 6.9%→9.2% (64k also 9.7% → NOT spanning). Escaped: 0.45% backslashes (vs
    azure 0.01%). Hypothesis: its long strings each carry an escape → Phase 1a delegates
    the WHOLE string to the byte-by-byte path (one escape in a 1 KB string = 1 KB crawled).
    → Phase 1c: streaming escaped-string fast path that bulk-copies clean runs between
    escapes (the streaming analogue of consumeStringEscaped).

Phase 1 remaining: 1b numbers, 1c escaped-string bulk runs. Then Phase 2 slide+grow
(spanning — secondary per the 4k≈64k finding). Measurement harness kept: streamgap_bench.

### 10.3b Phase 1b RESULTS — streaming number fast path (2026-07-20)

`consumeNumberStreamFast` (number.go): mirror of the string fast path — runs the
whole-buffer inline number scan over the window; aliases zero-copy when the number's
TERMINATOR is visible inside the window (end < bufferized ⇒ known-complete; unlike
whole-buffer, end==bufferized is NOT EOF, so it delegates). Delegates to
consumeNumberStreaming on: spans-window, bail forms (leadingZero/trailing-dot/
malformed-exponent/ambiguous), or maxValueBytes set. Relative offsets; l.consumed
untouched until alias so a delegate re-scans from the number start.

Recovery (stream-4k % of buffer, base→1a→1b): canada 40→36→75, numbers 32→37→71,
mesh 42→41→76, marine_ik 52→51→71, mesh.pretty 42→45→70, golang 40→61→83,
instruments 50→72→88, citm 60→73→86, twitterescaped 62→87→94. String-heavy held
within noise (Phase 1b doesn't touch the string path).

Correctness: extended TestStreamFastEquivalence with number boundary + bail + long-
spanning cases. Surfaced a PRE-EXISTING (not Phase-1b) buffer-vs-stream divergence on
the folded-look-ahead form "1.2.3": whole-buffer emits "1.2" then defers the error to
the rejected ".3"; consumeNumberStreaming rejects inline (repeated separator). BOTH
reject the document — only the token prefix + error code differ. The fast path
delegates that form, so it matches pre-Phase-1b streaming exactly. Test now pins
"both reject" for malformed input (strict identity for well-formed). This divergence
DISSOLVES in Phase 2 when the byte-by-byte consumers are retired (streaming would use
the same window scanner as whole-buffer). Full suite + -race green; generate idempotent.

**After Phase 1a+1b:** streaming is 70–94% of buffer for all workloads EXCEPT gsoc-2018
(8.8%, escaped long strings — untouched, delegates wholesale). → Phase 1c: streaming
escaped-string fast path (bulk clean runs between escapes). The ~65–70% cluster
(github, twitter, update-center) likely also has moderate escape density → 1c helps too.
