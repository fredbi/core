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

- ✅ **D2 — `Builder.Reset` drops the children/index capacity — and that is correct, not a leak
  (resolved by the C5 COW model).** Original worry: `b.n = nullNode` throws away the slice + map every
  Borrow→Reset cycle. But under copy-on-write the builder is `aliased` as soon as it hands out a
  `Node()`, so by the time `Reset` runs those backing arrays belong to the returned snapshot; reusing
  them would corrupt an already-published node. So `Reset` **must** drop them. The lost capacity is the
  irreducible cost of immutability and cannot be reclaimed by tweaking `Reset`.

  **Bigger picture (Fred, parked):** a *node-granularity* pool is unsound by construction — because
  aliasing can share a node's `children`/`keysIndex` into any number of clones that outlive the builder,
  "safe to redeem" is not locally decidable; it needs whole-document reachability. The only viable
  model is a **document-scoped allocator/arena** (an external oracle that frees all of a document's node
  structure in one shot), mirroring how `stores.Store` already arena-allocates scalar values. Individual
  nodes never self-redeem. Deferred until the document layer drives it; per-node pooling is considered
  moot. See [[nodes-inspection]] memory.

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

- ✅ **M1 — doc-comment typos (fixed):** "the alue of" → "the value of", "use cases hat" → "use cases
  that", and `decodeArray`'s mislabeled "empty object" comment → "empty array".
- ✅ **M2 — kind-guard + dup-check boilerplate extracted (fixed).** Added `requireObject(action)` /
  `requireArray(action)` and `rejectDuplicateKey(key, ik)` helpers; the 8 key/elem mutators now use
  them, removing the repeated `fmt.Errorf` blocks while preserving the exact error messages.
- ✅ **M3 — `Dump` use-after-redeem (real bug, fixed).** `Dump` called `RedeemUnbuffered(jw)` *before*
  `Encode(ctx)` used `jw` — the writer was returned to the pool while still being written to (a
  concurrent `Borrow` could hand out the same writer). Now a `defer RedeemUnbuffered(jw)` redeems only
  after `Encode` completes. New `TestNodeDump` loops to exercise pooled borrow/redeem reuse.

### Encoding path (E-series)

- ⏳ **E1 — unbounded recursion / no depth guard (TODO, delegated).** `encode` recurses one frame per
  nesting level. Per Fred: depth is bounded upstream by the **lexer's max-depth option** (the caller
  injects the lexer), so a decoded hierarchy is safe; programmatically built trees are the builder
  caller's responsibility. Valid but not urgent — left as a `TODO(fred)` on `encode`, no encode-side
  guard added.
- ✅ **E2 — children loops now re-check `w.Ok()` (fixed).** They previously checked only after
  `StartObject`/`StartArray`, so a mid-stream writer error still walked the whole subtree issuing
  writer calls — wasteful, and unsafe against the non-sticky buffered writer. Now each loop iteration
  (and the closing `End*`) bails on `!w.Ok()`. Test `TestEncodeWriterError` asserts the underlying
  writer is hit only a few times after a failure (no per-node write storm) and the error surfaces.
- ✅ **E3 — nil-safety (fixed).** `Encode` no longer derefs a nil `ctx.W` in its defer (guards up
  front); `encode` bails on a nil writer and raises a clean error on a nil store at a scalar leaf
  instead of nil-panicking. Tests `TestEncodeNilWriter` / `TestEncodeNilStore`.
- ⏳ **E4 — object encode trusts child keys (TODO, upstream).** Per Fred, key validity is guaranteed
  upstream by the builder/decoder, not re-checked by the consuming encoder; documented as a comment on
  the object branch, no encoder-side guard.
- ↩️ **E5 — REVERTED (Fred's call). `WriteTo` keeps the panic-on-corruption contract.** We briefly
  routed `WriteTo` corruption (out-of-range offset, invalid header) through the writer's `SetErr` instead
  of panicking. Fred rejected it: the `Store` presents a *closed, error-free* interface by design — store
  methods never return errors; on failure they yield a zero `Handle` and the caller guards against it.
  Singling out `WriteTo` for error-routing doesn't fit that contract (and corrupted handles are a
  hand-crafted/internal-corruption fault, which is exactly what panics are for). Reverted `WriteTo` to
  `assertOffsetInArena` / `assertValidHeader`, removed `writeToOffsetInArena` / `writeToInvalidHeader`
  and `TestWriteToCorruptHandle`. The store-wide panic contract stands for all paths.
- ℹ️ **E6 — `encode` now uses a pointer receiver + index-based loops** (Fred: "slightly faster"),
  avoiding the per-recursive-call Node copy and the per-child range copy. No behaviour change (encode
  never mutates).
- ℹ️ **Cross-ref (default-writer, out of scope here):** the *buffered* writer is not sticky on error —
  `buffered.writeSingleByte`/`flush` don't check `w.err`, and `flush` reassigns `w.err` unconditionally,
  so a later successful flush can clear an earlier error. File as a default-writer finding.

---

## 3. Proposed sequencing (for review — not started)

1. **Phase A — Correctness + tests.** Fix C1–C3, settle D1 then fix C4. Add table tests covering
   object index integrity across Append/Prepend/Insert/Remove/Swap + AtKey/KeyIndex round-trips, and
   null round-trips. Target the index ops specifically (today's gap). Aim to lift coverage past ~80%.
2. **Phase B — API/representation.** Resolve D1/D4 (null model + options), then D2/D3 pooling/alloc.
3. **Phase C — Docs + cosmetics.** M1–M3, doc pass, decide VerbatimNode (D5) fate.

## 3b. Target architecture (future) — `DocumentFactory` arena

The settled direction for node memory management (Fred), to revisit when the document layer drives it.

**Why it pays off — the uniform `Node` makes the pool *fungible*.** Every JSON value of every shape
decodes into the same `Node` type, so a node freed by one document can back a totally unrelated
structure on the next cycle. This is strictly stronger than a typed `sync.Pool[*T]`, which is
monomorphic and *fragments by type* (a thousand idle `*User`s don't help you allocate an `*Order`). A
node reservoir never fragments by schema because there is no schema at the node level → it approaches
~100% reuse across arbitrary traffic. Requirements/seams this implies:
- **Zero residual type identity on reset.** A recycled node must return as a neutral cell. Ours already
  is: `kind`, `children`, `value` (handle), `keysIndex`, `key`, `ctx` — all generic, nothing
  schema-shaped clings.
- **The fungible unit is the node *slot*; sub-resources are semi-specialized.** Object cells carry a
  `keysIndex` map, array cells a `[]Node`, scalars neither. So the arena is three reservoirs (node
  slots, `[]Node` backings, index maps) reassembled per shape — a node recycled object→array reuses the
  slot/slice but returns its map unused. (Same "maps aren't values" seam as everywhere else.)
- Same bet as the rest of the stack: tokens (lexer) → handles (store) → nodes (document) — a uniform,
  schema-agnostic, reusable currency, paid for with a heavier API than codegen'd typed structs.

Mechanics:

- **A `DocumentFactory` one level above is the allocator and the redeem oracle.** It spawns documents
  and their nodes from a region it owns; recycling the factory bulk-recycles *every* document and node
  it produced. The factory is the unique root, so "safe to redeem" becomes decidable at its boundary —
  which a single node can never decide locally (COW aliasing shares its `children`/`keysIndex` into
  clones with unknown lifetime). Region/arena allocation, not per-node reference pooling.
- **Unifies the two arenas under one lifetime.** `stores.Store` is already the value arena; the factory
  owns (or is) both the store and a node-structure arena. One `factory.Recycle()` resets both. The
  store's region model is the template.
- **Keeps the C5 COW model sound — the factory boundary is its safe scope.** Clones may share backing
  within one factory generation; the whole region drops together. Hard contract: **a node must not
  escape its factory** (never shared across a recycle).
- **Maps can't be arena'd.** Slab-allocate the `[]Node` child backings; pool-and-`clear()` the
  `map[InternedKey]int` index (Go maps are runtime-managed). Same "maps are references" property that
  forced per-mutation index copies forces the index to be pooled, not arena'd.
- **Integration seam = `cloneForWrite`.** It currently does `slices.Clone`/`maps.Clone` against the Go
  heap; to capture those in the factory the Builder needs an allocator handle (thread it like the store,
  via the builder's store ref or `ParentContext`) so COW draws from the factory, not the heap.
- D2 (`Builder.Reset` dropping capacity) is the local symptom of this: only the factory arena can
  reclaim that capacity; `Reset` itself must keep dropping.

## 4. Open decisions to confirm with Fred

- D1: single null model — fold `KindNull` into a null-handle scalar, or keep `KindNull` and make all
  predicates honour it? (affects `Kind`, `Value`, `IsNull`, builder `Null`, encode).
- D4: keep & modernize `DecodeOptions`/`EncodeOptions` to the zero-alloc functional-option pattern, or
  strip to the minimum actually used (`tolerateDuplKey` + hooks)?
- Scope: is this inspection meant to stay within `nodes/light`, or also sweep the `json.Document` layer
  that wraps it (where these bugs ultimately surface)?
