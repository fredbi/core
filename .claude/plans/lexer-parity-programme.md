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
2. **16-byte token is a live candidate (un-rejected).** `*[]byte`(8) + kind +
   delim + bool (3) + 5 pad = **16 B** — and the union-`uint64` is *unnecessary*,
   padding already absorbs the three small fields (Fred's doubt confirmed). The
   pointer is consistent with the documented "valid only until next NextToken"
   contract (`lexer.go:153-158`); the slice header lives in a stable lexer field
   so `&l.value` costs **zero per-token alloc**, and the only price is one deref
   per *consumed* value (invisible to throughput benches). Validate by micro-bench
   (§5.3). One API wrinkle: `token.MakeWithValue` must bind a stable field, not a
   param local (`&param` would escape → alloc).
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

### 4.1 Numbers ⬜
Current `number.go` validates the full grammar with scalar digit scans. Candidate
fast paths, cheapest probe first:
- all-digits → already short-circuited ✅
- `digit + (digits | '.')`, reject trailing `.` (a lone-trailing `.` is invalid) ⬜
- `'-'` + any of the above ⬜
- general scientific notation → slow path ⬜

Open question to settle *with the micro-bench*, not by intuition: most JSON
numbers are short (< 8 bytes), so a SWAR digit scan may not pay versus the scalar
loop. `decimals`/`exponential`/`ints_neg` micro-benches decide it.

### 4.2 Strings ⬜ — minimize the eager-unescape tax
**Decided: keep eager unescape** (the "ready-to-use value" is the feature). The
game is using cheap SWAR probes to *avoid* unescape work, not to defer it. Keep
the zero-copy alias path (unaltered string) ✅. Split:
- pure ASCII / pure unescaped UTF-8 → SWAR finds the closing quote first → alias,
  no unescape (already done) ✅
- dense escapes + clean tail → clean-run batching shipped ✅
- general randomly-positioned escapes / surrogates → slow path ✅
- **widen zero-copy to "alter but do not grow"** ⬜ — an unescaped string is always
  ≤ source length (every escape shrinks), so we could unescape *in place* over the
  aliased region instead of into `currentValue`, dodging the scratch copy. Needs
  care: the input buffer must be writable — **verify buffer ownership first**
  (whole-buffer mode aliases the *caller's* data per `lexer.go:221`, so in-place
  mutation of the alias is NOT safe without an opt-in/own-the-buffer mode). This
  caveat likely demotes the idea unless we own the buffer.

Lazy/raw-token deferral is **out of scope** — it would shed the eager tax by
pushing work onto every consumer, which contradicts the contract we keep.

---

## 5. Tricks — inlining, switch cost, token size

### 5.1 Devirtualization by codegen (PREP) 🚧
The tax: `scanTokenG[T,P]`/`scanPushG[T,P]` call `p.emit/p.none/p.eof` through the
generics dictionary (indirect, non-devirtualized) — ~5% on `L`. The generic cores
are ~350 lines and will *never* inline; the point is **not** to inline them but to
**remove the indirect dictionary call** so the tiny `emit` body becomes a direct,
inlinable construction.

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

### 5.2 Dispatch: classification table vs binary-search switch ⬜
**Verified (asm probe):** our switch on the lead byte does **not** become a jump
table — the JSON byte set is sparse (`"`=34 … `}`=125), so the compiler emits a
**binary search of ~4 `CMPB`** (one `JMP` for the contiguous digit range). So real
dispatch headroom exists.
- ❌ `[]func(*L) T` table — O(1) index but an **indirect call per token**: blocks
  inlining and re-adds the exact tax §5.1 removes. Don't.
- ⬜ **classification table** `class [256]uint8`: one indexed load → a *compact
  dense* switch on the contiguous class enum (this the compiler *can* jump-table)
  → direct, inlinable arms. Gives Fred's "compact + inlinable" goal without
  indirection, and composes with §5.1 (arms call the devirt funcs).
- Probe both on `separators`/`nulls`/`bools` (dispatch-dominated micro-benches);
  adopt the classification table only if it beats the binary-search switch.

### 5.3 Token size ⬜ (un-rejected — 16-byte candidate)
- `T` today = **32 B** = 24 (`[]byte` header) + `valueDelimiter`+`kind`+`valueBool`
  (3 B) + **5 B trailing padding**. `VT` = **72 B** → would become **56 B**.
- **Union `uint64` idea: dropped as pointless** — the three small fields already
  fit the 8-byte tail; padding is *after* them, so hiding them in a `uint64` frees
  nothing. Keep plain fields.
- **`*[]byte` idea: viable, 16 B.** `*[]byte`(8) + kind + delim + bool (3) + 5 pad
  = 16. Earlier rejection was **wrong**: returning `[]byte(*t.value)` copies only
  the header, never the bytes; and zero-alloc is preserved by pointing every token
  at a **stable lexer field** (`&l.value`, a pointer into the heap `*L`), never at
  a parameter local. Consistent with the documented short-lifespan contract
  (`lexer.go:153-158`); `Clone` already deep-copies for callers who keep tokens.
- Cost: one pointer deref per *consumed* value (strings/numbers only) — invisible
  to throughput micro-benches (tokens discarded), paid by real consumers. Benefit:
  halves the by-value token copy on every `NextToken` return and every push
  `yield`, which *does* show in throughput benches.
- One real wrinkle: `token.MakeWithValue([]byte)` must change to bind a stable
  backing field, not `&param`. Audit callers that (against contract) rely on a
  fast-path value surviving past the next token.
- **Validate by micro-bench** (`strings_plain`, `ints`, `separators` — copy-heavy
  paths) before retrofit. Tentatively the most promising single throughput lever
  after devirt.

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
4. ⬜ `lexgen` devirt generator (§5.1) — body-agnostic, so build it now; re-gen cheaply after each core change.

**Experiments (each: win micro-bench → no regressions → hold on corpora):**
4. ⬜ Token: 16-byte `*[]byte` candidate (§5.3) — likely top throughput lever after devirt.
5. ⬜ Numbers fast paths (§4.1), decided on micro-benches.
6. ⬜ Strings: in-place no-grow unescape (§4.2), gated on buffer ownership.
7. ⬜ Dispatch: classification table vs binary-search switch (§5.2).

**Later:**
8. ⏸️ Streams (§6).
9. ⬜ Gate review + retrofit to master (§7).
