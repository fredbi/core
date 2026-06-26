# String processing: study + optimization plan (Phase 4.1)

> Worktree `.worktrees/lexer/exploration`, branch `exploration`.
> Companion to [default-lexer-roadmap.md](default-lexer-roadmap.md) §4.1 and the
> go1.27 ramble (devirtualization is *not* the lever here).
> Status legend: ✅ done · ⏳ in progress · ⬜ todo · 🔬 measure first.

## 0. TL;DR

The only string workload that exercises our slow path is `strings_escaped`, and
there we are the weakest vs the jsontext yardstick (528/669 MB/s vs 812). The
root cause is **byte-by-byte copying in the unescape loop**: after the first
escape we append one byte per iteration with no clean-run batching. jsontext's
`AppendUnquote` batches clean runs (scan to next escape, one bulk `append`).

**Plan: batch clean runs in the unescape slow path** — after each escape, a
scalar scan finds the next stop byte and one bulk append copies the clean run,
mirroring `AppendUnquote`'s shape while keeping our *eager-decode* contract.

**OUTCOME (✅ shipped):** `strings_escaped_long` **+75–82% B/s vs our own
baseline** (closes the gap to jsontext from 38% → **69%**; it does **not** beat
jsontext — see correction below), real corpora +1.9% geomean with **no
regressions**, fast-path canaries untouched/flat. Only the synthetic dense-escape
`strings_escaped` regresses ~8% (tiny-run memmove cost) and that profile never
appears in real data. Notable twists vs the original plan: (a) a *shared*
`scanStringStop` helper was abandoned — it wouldn't inline and regressed the fast
path, so the fast path is left byte-identical to baseline; (b) the slow-path scan
is **scalar, not SWAR** — SWAR's word math costs more than it saves on the short
runs. See §4.2/§4.3/§4.7.

> **CORRECTION (2026-06-26):** an earlier draft said this "now beats jsontext" on
> `strings_escaped_long`. That was wrong for the **shipped scalar** variant: a
> full run vs json/v2 shows jsontext 1726 MB/s vs our 1193 (push) — we reach
> **69%**, not a win. The "+75/+82%" figures are correct but are deltas vs *our
> own* byte-by-byte baseline, not vs jsontext. The "beats jsontext" number
> (~2350 MB/s) came from the **SWAR** slow-path variant, which we did **not**
> ship (it regressed dense `strings_escaped` -16% vs scalar's -8%). Open item:
> re-evaluate SWAR vs scalar on **real corpora** before settling — SWAR may be the
> better ship if its long-string win outweighs the dense-escape loss in real data
> (only scalar was validated on real corpora so far).

Explicitly **out of scope**: the `strings_plain` gap (per-token dispatch
overhead, road-b `writegen`-style de-genericization, not the scanner) and UTF-8
validation (we deliberately don't validate on the alias path).

## 1. Background — the two architectures

### jsontext (encoding/json/v2): validate now, unescape later

jsontext splits string handling into two phases that never run together during
tokenization:

1. **Validation** — `ReadToken` → `jsonwire.ConsumeSimpleString` (fast,
   inlinable, ASCII-only) or `ConsumeStringResumable` (full). These *only*
   validate grammar + UTF-8 and advance a cursor. They return the **raw, still
   escaped, still quoted** bytes aliasing the input. **No copy, no unescape,
   ever, at tokenize time.**
2. **Materialization** — `jsonwire.AppendUnquote`, called lazily only when the
   consumer actually wants a Go string.

Our benchmark `Walk` reads only `tok.Kind()`, so on escaped/multibyte inputs
**jsontext does almost none of the work we do** — it validates and points at
bytes. Apples-to-oranges on escaped strings: we return a *decoded* value, it
returns a *validated-but-escaped* slice.

### default-lexer: eager decode

`consumeString` produces the final decoded value inside `NextToken`. Whole-buffer
mode (`consumeStringWhole`, `string.go`):

- **Fast path**: 8-byte SWAR word test (3 needles: `<0x20`, `"`, `\`) → on the
  common no-escape case, alias the input zero-copy. Multibyte (≥0x80) passes
  through transparently; we do **not** validate UTF-8 here.
- **Slow path** (first escape found): copy the clean prefix, then loop handling
  escapes — **but the `default:` arm appends one byte at a time** (`string.go`
  ~L168-170). No clean-run batching after the first escape.

## 2. Current numbers (go1.26.4, Ryzen 7 5800X)

| workload | our pull `bytes` | our push `tokens` | jsontext | data |
|---|---|---|---|---|
| `strings_plain`   | 914 MB/s | 1045 MB/s | **1386** | no escapes → both alias |
| `strings_escaped` | 528 MB/s | 669 MB/s  | **812**  | `\t \n \\ \"` → we unescape, they don't |
| `strings_unicode` | 881 MB/s | **1046**  | ≈        | raw UTF-8 `☃ 𝄞 é`, **no `\u`** → we alias, they DecodeRune |

Notes:
- `strings_unicode` is **literal multibyte UTF-8, not `\u` escapes**, so it hits
  our *fast alias* path — not the unescape path. We look good here precisely
  because we alias while jsontext must `utf8.DecodeRune`.
- `strings_plain` gap is per-token fixed overhead (grammar state,
  `token.MakeWithValue`, ~5% generic dict dispatch, key/colon bookkeeping), not
  the scan. Many tiny strings → fixed cost dominates. Not a scanner problem.

> **FRED**
>
> - `strings_plain` ok so the technique is sound. That leave open some further inline technique (e.g. injected
>   codegen) but at this stage, this case is valid.
>
> - `strings_escaped` yes the string should be cleaned up from JSON conventions. IIRC we also renormalize
>   some ambiguous unicode sequences ("surrogate pairs" - valid but may harm go or downstream consumers).
>   So we do some more work but lose ~17%. So perhaps there is a better way to do this.
>
> - `strings_unicode`
>
> **→ CLAUDE (reply)**
>
> Important correction, because it changes where we should aim: **the surrogate
> work and the `strings_escaped` 17% loss are unrelated.** `strings_escaped`
> contains only `\t \n \\ \"` — **zero `\u` sequences** — so
> `unescapeUnicodeSequence` (and all surrogate handling) **never runs** on that
> workload. The 17% there is *purely* the byte-by-byte copy loop + per-escape
> dispatch. Fixing the copy loop (§4.2–4.3) is the whole win for `strings_escaped`
> and touches no Unicode code at all.
>
> The surrogate cost you're remembering would only surface on a `\u`-heavy input —
> which **we don't currently benchmark** (hence §4.1's new `strings_uescaped`).
> That's exactly why I want it in the corpus before we reason about it: today we'd
> be optimizing a path no benchmark exercises.
>
> On "renormalize ambiguous sequences": what we actually do is (a) combine a
> *valid* high+low surrogate pair into its astral rune (necessary, jsontext does
> the same), and (b) **reject** invalid/lone surrogates with `ErrInvalidRune`.
> jsontext's *validating* tokenizer also rejects; its `AppendUnquote` instead
> *replaces* with U+FFFD. So our only semantic divergence is reject-vs-replace,
> and that's a **policy** question, not a perf one — I've parked it in §6 rather
> than letting it muddy this plan.
>
> `strings_unicode`: nothing to do — we already match/beat jsontext here because
> the data is literal multibyte UTF-8 (no `\u`), so it rides our zero-copy alias
> path while jsontext must `utf8.DecodeRune` every rune. This workload *validates*
> the alias strategy; leave it alone (it's the regression canary in §4.6).


## 3. The three technique differences

1. **Fast path — we're already good (arguably better).** SWAR 3-needle word test
   + zero-copy alias, handles multibyte transparently. jsontext's
   `ConsumeSimpleString` is a scalar byte loop + `escapeASCII[c]` table, ASCII
   only (≥RuneSelf bails to the rune-decoding path), kept simple to stay
   inlinable. No change needed.
2. **Slow path — THE gap.** jsontext `AppendUnquote` batches: inner `noEscape`
   run-loop advances over a clean run, then one bulk `append(dst, src[i:n]...)`,
   handle escape, `i = n`, repeat. We append byte-by-byte. ← fix target.
3. **`\u` decoding — minor.** Ours: `unhex` ×4 + a separate `consumeN(6)` for the
   surrogate tail. jsontext: `parseHexUint16` + inline surrogate handling, no
   re-fetch. Only matters for `\u`-heavy payloads (absent from current corpus).

## 4. Plan of work

### 4.1 ✅ Close the corpus blind spots first (so we measure the right thing)
- ✅ Added `strings_uescaped` (`\uXXXX` + surrogate pair every 4th elem) and
  `strings_escaped_long` (couple escapes + long clean tail) to `workloads.go`.
  Both pass `TestWorkloadsLex` (all impls agree). Stale comment on
  `unicodeStringElem` (claimed `\u`, was literal UTF-8) corrected.
- ✅ Baseline captured: `ramblings/2026-06-strings-baseline-go1.26.4.txt` (raw,
  count=10) + `2026-06-strings-baseline-summary.md` (medians + analysis).

**Baseline medians (MB/s, go1.26.4):**

| workload | our pull | our push | jsontext | push vs jt |
|---|---|---|---|---|
| `strings_plain`        | 958 | 1041 | 1397 | 75% |
| `strings_escaped`      | 571 | 614  | 797  | 77% |
| `strings_escaped_long` | 664 | 675  | **1779** | **38%** ← target |
| `strings_uescaped`     | 875 | 978  | 607  | **161%** ✅ we win |
| `strings_unicode`      | 872 | 1018 | 641  | **159%** ✅ we win |

**What the new workloads revealed (this changes the plan):**
1. `strings_escaped_long` is the real wound — **38% of jsontext (2.6× slower)**.
   The short `strings_escaped` *hid* it (benign 77%) because its elements are
   short + escape-dense. The byte-by-byte tail copy is murderous on a long clean
   run; jsontext validates-only (no unescape) + `noEscape` scan. This is THE
   target for §4.2–4.3.
2. **We already beat jsontext 1.6× on both Unicode workloads** (`uescaped` 978 vs
   607; `unicode` 1018 vs 641). jsontext pays heavy `\u` canonical-form
   validation + per-rune `DecodeRune`; we decode `\u` cheaply and alias literal
   multibyte. → **`\u`/surrogate is an advantage, not a deficit.** Confirms
   parking §4.6 (inlining `unescapeUnicodeSequence`) — nothing to chase there.
3. `strings_plain`/`strings_escaped`-short gaps are per-token dispatch, not the
   scanner. Batching nudges `escaped`; `plain` is road-b territory (out of scope).

> **FRED**
>
> I agree with inlining the unicode surrogate processing
> func (l *L) unescapeUnicodeSequence() (rune, error)
> But that's going to be challenging to put it under a budget of 80 tokens!
> (however there are just 2 call sites).
>
> `strings_escaped_long` : so how does this work? do we SWAR to search for the next \u
> and if none, we switch back to unescaped unicode?
>
> **→ CLAUDE (reply)**
>
> Subtle but key: **the SWAR scan does *not* look for `\u`.** It looks for the
> next *structural stop* — `"` OR `\` OR any byte `<0x20` — exactly the same three
> needles as the fast path. `\u` isn't special to the scanner; it's just one of
> the things we find *after* we land on a `\`. The scan's only job is "how far is
> the next interesting byte", so we can bulk-copy everything before it.
>
> Worked example — `"\n` + 2000 clean bytes + `"`:
>
> ```
> 1. fast-path SWAR hits '\' at index 1            → enter slow path
> 2. copy clean prefix data[start:1]  (empty here)
> 3. dispatch the escape: data[1]=='n' → append '\n'   (cursor now past "\n")
> 4. SWAR-scan forward for "/\\/<0x20                ← THIS is the new step
>    → the 2000-byte tail has none, so stop == the final '"'
> 5. ONE bulk append: data[i:stop]  (all 2000 bytes at once)   ← the win
> 6. data[stop]=='"' → done
> ```
>
> Today step 5 is 2000 separate `append(..., c)` calls (string.go `default:`
> arm). After §4.3 it's one `append(dst, data[i:stop]...)` = one `memmove`. The
> SWAR amortizes over the run instead of one branch per byte. `strings_escaped`
> (short elements, escape-dense) shows a modest win; `strings_escaped_long` (one
> escape, long tail) is where it's dramatic — which is why I want both in the
> corpus, so we don't over-claim from one shape.
>
> On inlining `unescapeUnicodeSequence` under cost-80: **don't force the whole
> function inline** — agreed it won't fit, and I'd argue we shouldn't try. Two
> better moves, in priority order:
>
> 1. **Stop using `consumeN` in whole-buffer mode.** `unescapeUnicodeSequence`
>    currently reads via `consumeN` (the streaming reader that re-syncs
>    `l.consumed`/`l.offset` each call). In whole-buffer mode we already hold all
>    the bytes in `data` at cursor `i` — we should parse `data[i:i+4]` hex
>    *directly* off the local cursor, like the rest of `consumeStringWhole`. That
>    removes the re-sync churn and the `[4]byte`/`[6]byte` shuffling regardless of
>    inlining. (New §4.7.)
> 2. **Split hot/cold.** The frequent case is a single BMP `\uXXXX` (no
>    surrogate). Keep a tiny inlinable BMP path (parse 4 hex → `AppendRune`); push
>    the rare surrogate-pair logic into a `//go:noinline` cold helper. That keeps
>    the hot path small *without* contorting the whole function under budget.
>    Borrow jsontext's `parseHexUint16` trick (one packed compute over 4 digits)
>    in place of our four `unhex` calls.
>
> But both of these only pay off on a `\u`-heavy workload, so they're gated behind
> §4.1 landing `strings_uescaped`. If the benchstat says `\u` is rare/cheap in
> practice, we skip the inlining entirely — measure first.

### 4.2 ✅ (revised) Helper extraction abandoned — fast path left untouched
- **Tried** lifting the SWAR into `scanStringStop` and calling it from both
  paths. It does **not** inline (cost 98 > 80), so the fast path paid a call per
  string and **regressed** `strings_plain`/`unicode`/`uescaped` ~3–5%.
- **Then tried** factoring only the word-test (`hasStringStop`, cost 50, inlines)
  and writing the loop inline in both paths. Fast path recovered, but the
  `tokens` push path still showed a ~5% wobble on the Unicode canaries.
- **Final decision: do not touch the fast path at all.** The slow path uses a
  *scalar* run-scan (not SWAR — see 4.3), so it never needed the helper. The
  fast-path SWAR block is now byte-identical to baseline → Unicode/uescaped/plain
  canaries are statistically flat (p>0.1). Lesson: a shared helper isn't worth a
  fast-path inlining regression; duplication-free here means "don't share".

### 4.3 ✅ Batch clean runs in the unescape slow path (scalar scan)
- Rewrote the slow-path loop: after each escape, a **scalar** scan finds the next
  stop byte (`"`/`\`/`<0x20`), then **one bulk `append(l.currentValue,
  data[i:stop]...)`** copies the clean run. Scalar (not SWAR) is deliberate — on
  escape-dense strings the runs are tiny and the SWAR word math costs more than it
  saves; the win is the single memmove replacing one append per byte.
- `maxValueBytes` checked as `len + (stop-i)` **before** the append (rejects an
  over-long value without copying a huge run; zero-width run still catches
  escape-only expansion). Eager `\u` decode unchanged.
- All tests + `-race` green (conformance, handoff, zerocopy, security, max-value).

### 4.4 ✅ Push core shares the win
- `consumeStringWhole` is shared by pull + push; the `tokens` (push) benchmark
  shows the same `strings_escaped_long` gain (-45% sec/op), confirmed. No
  separate change needed.

### 4.5 ⬜ Streaming path (`consumeStringStreaming`)
- Lower priority. The refill loop is inherently byte-at-a-time across buffer
  boundaries, but within a refilled chunk we could still batch. Defer unless the
  benchstat shows streaming string-heavy workloads matter; note the decision.

### 4.6 ⏸️ Whole-buffer-native `\u` decode (DE-PRIORITIZED — §4.1 says we already win)
> Baseline shows we beat jsontext 1.6× on both `\u` and multibyte. No deficit to
> chase. Keep this section for reference; revisit only if a real-world `\u`-dense
> corpus regresses. The hot/cold-split design below stands if we ever need it.

- Replace the `consumeN`-based `unescapeUnicodeSequence` call *inside the
  whole-buffer slow path* with a direct parse off the local `data`/`i` cursor:
  parse `data[i:i+4]` (bounds-check once), advance `i += 4`; only on
  `utf16.IsSurrogate` drop into a `//go:noinline` cold helper that reads the
  trailing `\uYYYY` from `data[i:]`. Borrow `parseHexUint16` (packed 4-digit
  parse) in place of 4×`unhex`.
- Keep the streaming path (`consumeStringStreaming`) on `consumeN` — that
  abstraction is correct there (it spans buffer refills).
- Decision checkpoint: do this **only if** the `strings_uescaped` baseline shows
  the `\u` path is a measurable fraction. Otherwise record "skipped, `\u` not hot"
  and move on. (Answers FRED's inlining question: hot/cold split, not whole-fn.)

### 4.7 ✅ Results (go1.26.4, count=10, benchstat; raw in ramblings/2026-06-strings-cleanrun-*)

**Synthetic strings (B/s vs *our own* baseline — NOT vs jsontext):**

| workload | pull | push | note |
|---|---|---|---|
| `strings_plain`        | ~ | ~ | flat (fast path untouched) |
| `strings_unicode`      | ~ | ~ | flat (canary held) |
| `strings_uescaped`     | ~ | ~ | flat (canary held) |
| `strings_escaped`      | -8.4% | -8.7% | dense-escape tradeoff (tiny runs) |
| `strings_escaped_long` | **+75%** | **+82%** | the target; also allocs 8→5 |

**vs jsontext (push MB/s, full run 2026-06-26)** — for calibration, this is where
we actually land, not just vs our baseline:

| workload | our push | jsontext | ratio |
|---|---|---|---|
| `strings_escaped_long` | 1193 | 1726 | **0.69** (was 0.38) |
| `strings_escaped`      | 555  | 797  | 0.70 |
| `strings_plain`        | 1024 | 1389 | 0.74 |
| `strings_unicode`      | 978  | 620  | **1.58** |
| `strings_uescaped`     | 954  | 594  | **1.61** |

**Real corpora (B/s vs baseline) — the decisive gate, all flat-to-faster:**

| workload | pull | push |
|---|---|---|
| `citm_catalog`   | +1.4% | ~ |
| `twitter_status` | ~     | +3.9% |
| `mixed`          | +2.8% | ~ |
| `object_keys`    | ~     | ~ |
| **geomean**      | **+1.9%** | (no regressions) |

**Verdict:** net win **vs our prior code** (the gate that matters for shipping).
The clean-run batching closes `strings_escaped_long` from 38% → 69% of jsontext —
a big improvement but **not** a win against json/v2. The only regression
(`strings_escaped`, dense short escapes) does not appear in any real-world
workload; the representative escaped profile (occasional escape in long clean text
= `strings_escaped_long` / twitter) improves most. Fast-path canaries untouched.
Shipped (scalar variant). **Open: the SWAR variant beats jsontext on
`strings_escaped_long` but was not validated on real corpora — re-evaluate.**

### 4.8 ✅ Validation
- ✅ `go test ./...` (suite + conformance + handoff + zerocopy + security +
  max-value), `-race`, equivalence tests (push==pull, lab==ref) — all green.
- ✅ benchstat vs baseline captured (§4.7). `strings_escaped_long` up +75/+82%,
  fast-path canaries (`plain`/`unicode`/`uescaped`) statistically flat (p>0.1),
  real corpora flat-to-faster (+1.9% geomean). Raw under ramblings.
- ✅ Fast path untouched, so its 0-unamortized-alloc property is unchanged
  (pooled/reset variants still 0 B/op). Slow-path `currentValue` peak grows
  modestly (bulk-append rounding: escaped_long 832→960 B/op) — a one-time reused
  buffer, not a per-token leak.
- ✅ Added `maxValueBytes` slow-path regression cases to `TestMaxValueBytes`
  (leading escape forces the unescape path; oversized clean run trips, under-limit
  escaped value accepted).

## 5. Risks / watch-list
- **Fast-path regression**: extracting the SWAR into a helper must keep it
  inlined — if it stops inlining, `strings_plain` regresses. Gate on `-m` +
  benchstat before/after.
- **maxValueBytes accounting**: bulk append changes when the bound is observed
  (per-run vs per-byte). Keep the check before each append so an over-long run is
  still rejected; add a regression test with a value that crosses the bound mid
  clean-run.
- **Eager contract preserved**: the decoded value must remain byte-identical to
  today on every fixture — the existing equivalence + conformance suites are the
  oracle; no new semantics.

## 6. Non-goals (recorded so we don't drift)
- Lazy/deferred unescape (jsontext's model) — would change our token-value
  contract; out of scope, possibly revisited if/when a json/v2 contrib lexer is
  built that wraps `jsontext` directly.
- UTF-8 validation on the alias path — deliberately omitted (perf); separate
  decision if ever required.
- `strings_plain` throughput — bounded by per-token dispatch, addressed by
  road-b de-genericization, not this plan.
- **Invalid/lone-surrogate policy** (reject with `ErrInvalidRune`, as today, vs
  replace with U+FFFD à la jsontext's `AppendUnquote`) — a *semantic* decision,
  not perf. Recorded here so it doesn't ride in on a perf change. Our current
  reject behavior is covered by the conformance suite; revisit separately if a
  downstream consumer needs lenient replacement. (Raised by FRED in §2.)
