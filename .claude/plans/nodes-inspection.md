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
- ✅ **E7 — caller-side zero-handle guards on the `Put` path (added).** The counterpart to the E5
  revert: since the store reports a rejection as a zero `Handle` (not an error), the *callers* must
  guard. Both store-writing paths now do. **Decode** (`node.go`): new `Node.putValue(ctx, h)` — a zero
  handle routes `nodecodes.ErrNode` through the lexer's `SetErr` and the caller `return`s (stops). All
  five `decodeToken` sites (`PutNull` for object/array containers, `PutNull`/`PutBool`/`PutToken` for
  scalars) go through it. **Build** (`builder.go`): new `Builder.setValue(h)` — a zero handle records
  `nodecodes.ErrBuilder` so the chain short-circuits on the next `Ok()`. All eight builder sites
  (`StringValue`/`BytesValue`/`BoolValue`/`NumberValue` + the `buildFrom{Float,Integer,Uinteger,TextAppender}`
  helpers) go through it. Tests: `TestBuilderZeroHandleGuard` / `TestDecodeZeroHandleGuard` drive a
  `zeroStore` fake that rejects every value; previously a rejection left a node silently bound to
  `HandleZero`.
- ℹ️ **Cross-ref (default-writer, out of scope here):** the *buffered* writer is not sticky on error —
  `buffered.writeSingleByte`/`flush` don't check `w.err`, and `flush` reassigns `w.err` unconditionally,
  so a later successful flush can clear an earlier error. File as a default-writer finding.

### Context machinery (`context.go`) — combed before the decode path

The `ctx.P` `Path` is a single shared, mutable, growable slice. Each decode level captures its base
length, appends one slot for the first child, **overwrites** it for each sibling, and truncates back on
exit. Verified correct: the helpers compare only **lengths** (`len(p) == len(original)`), never
dereference `original`, so the logic is robust to slice reallocation; each depth exclusively owns its
slot. JSON-Pointer escaping in `String()` (`~`→`~0`, `/`→`~1`, single-pass `Replacer`) is correct.

- ✅ **CTX-1 — the JSON Pointer path is now attached to decode errors (Fred: "definitely").** The intent
  of `ctx.P` is to pinpoint *where* an error occurred. It was built at per-token cost but **never
  surfaced**: `ErrContext` had no path field and `Decode`'s defer ignored `ctx.P`. Added a `Path` field
  to `codes.ErrContext` (forward-compatible with the planned move of JSON-Pointer tracking into the
  lexer) and `Decode`'s defer now sets it from `ctx.P.String()` in both branches. The error-path
  truncations in `decodeObject`/`decodeArray` are guarded by `if l.Ok()`, so on error `ctx.P` still
  points at the failing node. Tests `TestDecodeErrorPath` (nested `/a/c`, array `/2`, escaped
  `/a~1b~0c`, happy-path nil context). `Pretty()` left unchanged (golden tests).
- ✅ **CTX-2 — removed the dead `EmptyPath`.** Exported but unused, and unusable externally anyway
  (`stringOrInt` is unexported).
- 🔮 **CTX-3 (future, Fred) — JSON-Pointer feature is unfinished and slated to move into the lexer** as
  a lexer option. The `stringOrInt` struct may be simplified (stringify the int eagerly), and the bare
  `panic("assert")` in the path helpers is a known unpolished edge. Left as-is for now; revisit when the
  feature moves. Possibly extend path tracking to the encode side too.
- ✅ **CTX-4 — fixed the hook `skip` lexer desync + made `skip` semantics uniform.** Decided semantic:
  **skip = discard this value from the result but still consume its tokens** (drain a composite),
  staying in sync; a skip is silent (no nested hook fires for drained tokens). Implementation:
  - new `drainValue(ctx, tok)` consumes the rest of a value whose opening token is already read
    (balances start/end container tokens — separators are elided; scalar = no-op).
  - `BeforeKey` skip now reads the pending value token and drains it before continuing (was: left it
    unconsumed → next read misparsed it as a key → `ErrMissingKey`).
  - `BeforeElem` skip now drains the element (was: composite bodies leaked into the next read).
  - `NodeHook` skip now drops the node *consistently*: `decodeToken` returns `produced bool`, and the
    object/array loops `continue` (drop) when a value was skipped — previously a `NodeHook` skip left an
    empty placeholder node because the parent still yielded it. (`NodeHook` skip was unused in practice —
    the constrained validators only ever return `err` — but the asymmetry is now closed.)
  - `AfterKey`/`AfterElem` skip were already correct (value fully consumed; just not yielded).
  - hardening bonus: `decodeToken`'s EOF-as-value case now sets an error instead of returning silently
    (top-level `decode()` already filters EOF, so reaching it means an unterminated container).
  - hook `skip`/`err` contract documented on the hook types in `hooks.go`.
  - Tests: `TestDecodeSkip` (Before/After Key/Elem + NodeHook, scalar + composite + deeply nested
    drains) and `TestDecodeSkipDoesNotMaskErrors`; unterminated input verified to error, not hang. Green
    under `-race`.
- ✅ **CTX-5 — documented `ParentContext`'s single-goroutine contract.** Fred: the node machinery is
  single-goroutine *by design* (decode pulls from a stateful lexer, encode pushes into a stateful
  writer; the whole logic is not thread-safe). Godoc now states it's not safe for concurrent use (one
  per goroutine per document), annotates the terse single-letter fields, and notes `ctx.P` is only valid
  during a callback. Fields remain in flux (kept light per Fred).

### Decode path — container assembly (`decodeObject` / `decodeArray` / `decodeToken`)

- ✅ **DEC-1 — duplicate-key default mode lost data and reported the wrong error (BUG, fixed).**
  `{"a":1,"a":2,"b":3}` with the default (`tolerateDuplKey=false`) used to `return false` on the
  duplicate **without setting an error**: the object iterator stopped, leaving the rest unconsumed, the
  result was silently truncated to `{"a":1}` (both `a:2` *and* `b:3` dropped), and the leftover tokens
  reparsed at top level surfaced a misleading "invalid JSON token". Now it sets a clean
  `nodecodes.ErrDuplicateKey` naming the key; CTX-1 attaches the JSON Pointer (the iterator is suspended
  at the offending key's `yield` and skips its truncation on error, so `ctx.P` pinpoints it). Tolerate
  mode is unchanged (last-wins, all siblings kept). Tests: `TestDecodeDuplicateKey` (default error +
  path, nested path `/x/a`, tolerate last-wins incl. trailing duplicate).
- ✅ **DEC-2 — `decodeObject`/`decodeArray` returned a nil iterator on the `!l.Ok()` short-circuit
  (latent panic, hardened).** Ranging a nil `iter.Seq`/`iter.Seq2` panics. The path is unreachable today
  (the only caller checks `l.Ok()` first), but it's a footgun; both now return an empty iterator.
- ✅ **DEC-3 — empty containers** (`{}`, `[]`, nested/array-of empties) confirmed correct and pinned by
  `TestDecodeEmptyContainers`.
- ℹ️ **DEC-4 — `n.ctx.offset` is captured *after* the node's opening token** (`l.Offset()` runs once the
  `{`/`[`/scalar has been read), so it marks just past the start, not the node's first byte. Minor and
  consistent across kinds; left as-is (the JSON-Pointer/offset feature is in flux, slated to move into
  the lexer — see CTX-3). Noted, not changed.
- ✅ **DEC-5 — the missing-key guard** (`!tok.IsKey()` → `ErrMissingKey`) and the `IsEndObject`/
  `IsEndArray` empty-exit checks reviewed: correct given the semantic lexer elides separators.
- ✅ **DEC-6 — top-level `decode()` loop reviewed: correct, contract documented + pinned.** The loop
  re-invokes `decodeToken` on the same `n`, so multiple top-level values would last-win — but the
  injected lexer rejects a second top-level value *at tokenization* (`1 2` → "value should follow a
  delimiter"; `{} {}` → "missing comma"; trailing garbage → "invalid JSON token"), so the overwrite is
  unreachable and the first value is never silently replaced. Empty/whitespace-only input →
  `lexcodes.ErrNoData`. The node layer correctly leans on the lexer for grammar rather than
  re-implementing it. Documented on `Decode`; pinned by `TestDecodeTopLevel` (single value accepted;
  trailing/multiple and empty inputs error; trailing whitespace tolerated).

### Read/accessor API (`node.go` query methods) — Tier-2

- ✅ **ACC-1 — `Pairs`/`Elems`/`IndexedElems` returned a nil iterator on a kind-mismatch (public-API
  panic, fixed).** Same footgun as DEC-2 but on the *public* surface: `for range scalar.Pairs()` panicked
  (ranging a nil `iter.Seq` is a nil-func call). All three now return an empty iterator on the wrong
  kind, so ranging is always safe. Pinned by `TestAccessorIteratorsNeverPanic` (every kind × all three).
- ✅ **ACC-2 — `KeyIndex` lacked the kind guard** the other object accessors have (it was safe via
  nil-map read, but inconsistent). Added `kind != KindObject → (0,false)`.
- ✅ **ACC-3 — documented the not-found contract.** `KindNull` is the zero kind, so the zero `Node`
  returned by `AtKey`/`Elem`/`AtInternedKey` on a miss is indistinguishable from a JSON null. Callers
  must test the `bool`, not infer absence from kind — now stated on `AtKey`.
- ℹ️ **ACC-4 — `Value`/`Handle` are scalar-only by design** (return false for null and containers); a
  null value is reached via `IsNull`, not `Value`. Confirmed intentional (KindNull ≠ KindScalar); pinned
  by `TestAccessorScalarKindGuards` (null reports false from Value/Handle; containers too).
- Coverage: `Elem` bounds (incl. negative), `AtKey`/`KeyIndex` hit & miss, `Len`/`Is*` across kinds,
  order-preserving `Pairs`/`Elems`, and wrong-kind safety — all pinned (`TestAccessorObject`,
  `TestAccessorArray`, `TestAccessorScalarKindGuards`). Index integrity itself (keysIndex↔children) was
  the C1–C4 builder track, already fixed; the accessors trust that invariant.

### Pools (`pools.go`) — combed before the decode path

Per Fred: standardize on the `pools.PoolRedeemable` variant (cached, alloc-free, built-in redeemer that
detects double-redeem) and simplify the API to a uniform `Borrow…() (*T, func())` shape.

- ✅ **POOL-1 — `poolOfParentContexts` → `pools.NewRedeemable`; uniform borrow-with-redeem API.**
  `BorrowParentContext() (*ParentContext, func())` and `BorrowPath() (Path, func())` (renamed from
  `BorrowPathWithRedeem`); dropped `RedeemParentContext`. `poolOfPaths` was already redeemable-backed
  (`PoolSlice`). Callers adapted across both modules: `json/document.go` (decode + encode),
  `json/constrained/constrained.go` (5 sites), `jsonschema/overlay.go`, `jsonschema/schema.go`.
- ✅ **POOL-2 — removed the dead, pooling-incompatible `Builder` pool.** `light.BorrowBuilder` /
  `RedeemBuilder` had no callers, and `Builder.Reset` deliberately keeps `b.s` (the reuse benchmark
  `builder_cow_test.go` does `rb.Reset()` then keeps building, so the store must survive `Reset`). Since
  `PoolRedeemable`'s only cleanup hook is `Reset`, a pooled `Builder` would pin its store while idle —
  exactly the leak the redeem-reset contract forbids. Rather than ship that, the unused pool is removed.
  If builder pooling is wanted later, resolve the `Reset`-keeps-store vs pool-clears-store tension first.
- ✅ **POOL-3 — added the missing pool tests** (`pools_test.go`, none existed): `ParentContext.Reset`
  drops every injected reference (no idle pin), borrow/redeem round-trip is clean, and the double-redeem
  guard panics — for both the context and path pools. Green under `-race` and `-tags poolsdebug`.
- 🐞 **Pre-existing, out of scope (jsonschema WIP, not from this change):** `jsonschema` does not compile
  on its own today — `schema.go:106/108 undefined: meta.Data`, and `octx declared and not used` in
  `core.go:185` / `applicator.go:209` (files untouched here). Flagged for the owner.

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
