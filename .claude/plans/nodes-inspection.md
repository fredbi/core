# nodes package — inspection & hardening roadmap

> Status: **Phase A in progress** — C1–C4 fixed + index-integrity tests added (cov 42.3% → 52.0%).
> Scope: `json/nodes` (+ `json/nodes/light`, `json/nodes/error-codes`). The `light.Node` is the base
> building block for `json.Document`, `json.Collection`, constrained docs and JSON-pointer support.
> Branch: `nodes/inspection` (worktree `.worktrees/nodes/inspection`), off `master` @ `1d1589c`.
> Last updated: 2026-06-26

## Legend

- ✅ done
- 🚧 in progress
- ⏳ planned / not started
- 🔬 needs design decision
- 💭 stretch

---

## 0. Context

- Library is **WIP, unpublished** → fix aggressively, no backward-compat burden.
- `nodes/light` is "the light version": values are `stores.Handle` references into an external
  `stores.Store` injected via `ParentContext`; the node holds only structure + a handle. Compact, but
  the API is heavier than a plain document (caller threads store/lexer/writer through `ParentContext`).
- Consumers (within `json/`): `document.go`, `collection*.go`, `constrained/constrained.go`,
  `pointer.go`, `builder.go`, `options.go`. So bugs here surface as Document bugs.
- Baseline: builds, vets, tests green. **Coverage 42.3%** — the Builder's object-index operations and
  most error paths are untested, which is exactly where the bugs below live.

## 1. Package map

| file | LOC | role |
|------|-----|------|
| `light/node.go` | 638 | `Node` type, accessors, `Decode`/`Encode` (lexer→node→writer) |
| `light/builder.go` | 670 | fluent `Builder` to construct/mutate nodes programmatically |
| `light/context.go` | 133 | `Context` (offset), `ParentContext` (DI), `Path` (JSON-pointer trail) |
| `light/pools.go` | 46 | `Borrow/Redeem` for Builder, ParentContext, Path |
| `light/hooks.go` | 28 | decode hook callbacks |
| `light/options.go` | 13 | `DecodeOptions` / `EncodeOptions` (skeletal) |
| `light/verbatim.go` | 6 | `VerbatimNode` — empty stub (TODO) |
| `nodes/nodes.go` | 32 | `Kind` enum |
| `error-codes/errors.go` | 19 | `ErrNode`, `ErrBuilder` sentinels |

---

## 2. Findings

### Correctness (confirmed by probe tests)

> ✅ **C1–C4 fixed** on branch `nodes/inspection`. New tests: `light/builder_index_test.go`
> (`TestBuilderObjectIndex` invariant-checks index/children sync across Append/Prepend/Insert/Remove/Swap
> + dup-key rejection on all paths; `TestNodeIsNullAndEncodeHandles` covers IsNull + the zero-handle
> encoder guard). Coverage 42.3% → 52.0%, vet + golangci-lint clean.

- ✅ **C1 — `Builder.AppendKey` writes an off-by-one key index (primary bug).**
  `builder.go:176` sets `b.n.keysIndex[value.key] = len(b.n.children)` *after* the append, so the index
  points one past the element (decode path does it right at `node.go:350`: `len(children) - 1`).
  Consequence: **every** index-based op on an object built via `AppendKey` is corrupted —
  `AtKey`/`AtInternedKey` returns the wrong child or **panics** with index-out-of-range, and `KeyIndex`,
  `Swap`, `RemoveKey` all read the poisoned index. Probe: building `{a,b}` yields `keysIndex={a:1,b:2}`
  for `children` of length 2 → `AtKey("b")` panics. Latent because the only object test appends a
  single key and never looks it up by key (Encode walks `children` directly).
  Fix: `len(b.n.children) - 1`.

- ✅ **C2 — `Builder.RemoveKey` does not re-index keys after the removed position.**
  `builder.go:284-285` deletes the map entry and `slices.Delete`s the child, but every key whose index
  was `> removed` still maps to its old (now shifted) position. `InsertKey`/`PrependKey` correctly
  shift; `RemoveKey` doesn't. Independent of C1; both compound to fully desync `keysIndex` from
  `children`. Fix: decrement `keysIndex[k]` for all `k` with index `> removed`.

- ✅ **C3 — `Builder.InsertKey` duplicate-key check tests an unset key.**
  `builder.go:240` reads `b.n.keysIndex[value.key]` but `value.key` is only assigned at line 249 — so
  the dup check always runs against the zero `InternedKey`, never detecting a real duplicate, and on the
  fast paths it delegates to Append/Prepend which check correctly only by luck. Fix: move the
  `value.key = MakeInternedKey(key)` assignment above the check (mirror AppendKey).

- ✅ **C4 — `Node.IsNull` returns false for an actual null node.**
  `node.go:114` short-circuited `if n.kind != KindScalar { return false }`, but decode tags JSON `null`
  as `KindNull` (`node.go:375`), not scalar. So `IsNull` was false on the very nodes it should flag, and
  only ever true for a *scalar* holding `HandleZero` — which is wrong (HandleZero is absence, not null).
  **Decision from Fred (resolves D1):** the null model is settled — JSON null is a `KindNull` node
  carrying a *dedicated, non-zero* null handle (`Store.PutNull()`); `HandleZero` means "no value" and
  signals handle corruption. Fixes applied: `IsNull` now returns `n.kind == nodes.KindNull`; and the
  encoder now **errors** on a `KindScalar` node whose value `IsZero()` (previously `WriteTo(HandleZero)`
  silently emitted nothing → invalid JSON).

- ✅ **C5 — `Builder` clone-and-mutate was not copy-on-write; mutating a clone corrupted the original
  (fixed).** `Node` is meant to be immutable and cheap to clone, with `From(n)` yielding a mutable
  derivative that leaves `n` intact. `From` was a **shallow struct copy** (`b.n = n`), so `b.n.children`
  aliased `n`'s backing array and `b.n.keysIndex` aliased the *same map*, and no mutation did
  copy-on-write. Proven (probes): `From(obj).AppendKey` mutated the original's index in place
  (`{a,b}`→`{a,b,c}` over a length-2 children → self-inconsistent, `AtKey` can panic); `From(arr).
  RemoveElem(0)` ran `slices.Delete` in place (`[e0 e1 e2]`→`[e1 e2 …]`); `Swap` reordered the shared
  backing. So **no** mutation reliably left the original unaltered.

  **Fix — guarded copy-on-write.** Added an `aliased` flag + `cloneForWrite()` that `slices.Clone`s the
  children and `maps.Clone`s the index on the *first* mutation, then clears the flag. `From` and `Node`
  both set `aliased = true` (so a seeded clone *and* every handed-out snapshot are protected). The flag
  is what avoids over-copying along a fluent chain: a chain copies the slice+index **at most once**, not
  per step; further ops mutate the now-owned structures in place. `resetNode` is COW-aware (drops the
  shared refs instead of reusing them). Allocation profile (measured): clone-without-mutation **0**;
  one object mutation **4**; a **five**-mutation chain **5** (≈ one mutation, not 5×) — the index is
  copied once. Tests: `builder_cow_test.go` asserts the original is byte-for-byte unaltered after every
  object/array op, that two snapshots from one builder diverge independently, and the alloc profile.
  Race-clean; coverage 53.0% → 57.8%.

### Design / API smells

- ✅ **D1 — Null representation (decided).** JSON null = `KindNull` node holding a dedicated non-zero
  null handle (`PutNull()`); `HandleZero` = absence/corruption, never null. `IsNull` + encoder aligned
  in C4. Remaining consistency follow-up (low priority): `Value()` returns `(NullValue,false)` for a
  `KindNull` node — revisit whether a null node should yield `(NullValue,true)` so callers can
  distinguish null from absence via the accessor too.

- 🔬 **D2 — `Builder.Reset` discards pooled capacity.** `builder.go:46` does `b.n = nullNode`, dropping
  the `children` slice and `keysIndex` map. Since `Builder` is pooled (`poolOfBuilders`), Borrow→Reset
  loses the per-builder allocations every cycle — partly defeating the pool. Consider reusing via
  `resetNode()` semantics (keep backing arrays, clear length) like the decode reset paths.

- ✅ **D3 — `Swap` allocated a `keysIndex` for arrays (fixed).** `Swap` called `ensureIndex()`
  unconditionally, creating an unused map for array nodes; removed — only the object branch touches the
  index now, arrays stay `keysIndex == nil`. Also added the missing bounds check (was an unguarded
  out-of-range panic; now returns an `ErrBuilder` like `RemoveElem`). Covered by two new subtests in
  `TestBuilderObjectIndex`.

- 🔬 **D4 — `options.go` is skeletal / dead-ish.** `DecodeOptions` embeds hooks + `tolerateDuplKey`;
  `EncodeOptions` is empty with a commented field; file header literally says "I don't think we really
  need options here." Decide whether these stay (and adopt the new zero-alloc functional-option pattern
  used in writers/stores) or get trimmed.

- 💭 **D5 — `VerbatimNode` is an empty stub** (`verbatim.go`) referenced from `node.go` docs as the
  verbatim-reconstruction path. Either schedule it or mark clearly as not-yet.

### Minor / cosmetic

- ⏳ **M1 — typos in exported doc comments:** `node.go:59` "the alue of", `node.go:30` "use cases hat",
  `decodeArray` comment "empty object" should read "empty array" (`node.go:523`).
- ⏳ **M2 — `AppendKey`/`PrependKey` duplicate the object-kind-guard + dup-check boilerplate** across 5
  methods; candidate for a small helper once behaviour is fixed.
- ⏳ **M3 — `Dump` redeems the unbuffered writer (`node.go:271`) *before* calling `Encode` with it**
  (`node.go:272`). Works only because RedeemUnbuffered doesn't invalidate the writer; still reads as a
  use-after-redeem and should be reordered.

---

## 3. Proposed sequencing (for review — not started)

1. **Phase A — Correctness + tests.** Fix C1–C3, settle D1 then fix C4. Add table tests covering
   object index integrity across Append/Prepend/Insert/Remove/Swap + AtKey/KeyIndex round-trips, and
   null round-trips. Target the index ops specifically (today's gap). Aim to lift coverage past ~80%.
2. **Phase B — API/representation.** Resolve D1/D4 (null model + options), then D2/D3 pooling/alloc.
3. **Phase C — Docs + cosmetics.** M1–M3, doc pass, decide VerbatimNode (D5) fate.

## 4. Open decisions to confirm with Fred

- D1: single null model — fold `KindNull` into a null-handle scalar, or keep `KindNull` and make all
  predicates honour it? (affects `Kind`, `Value`, `IsNull`, builder `Null`, encode).
- D4: keep & modernize `DecodeOptions`/`EncodeOptions` to the zero-alloc functional-option pattern, or
  strip to the minimum actually used (`tolerateDuplKey` + hooks)?
- Scope: is this inspection meant to stay within `nodes/light`, or also sweep the `json.Document` layer
  that wraps it (where these bugs ultimately surface)?
