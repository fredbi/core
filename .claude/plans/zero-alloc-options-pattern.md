# Zero-alloc options pattern — standardize across the repo

> Status: **APPLIED to writer + store — for review.** Pattern decided (variant C); both
> `default-writer` and `default-store` converted on branch `writer-options-zeroalloc`. Lexer is being
> reworked separately and is explicitly out of scope for now. Other `func(*options)` packages: later.
> Origin: the `default-writer` options machinery allocates and was worked around with
> per-type option pools; `stores/default-store` already solved this with an immutable
> value-builder. Goal: extract the canonical pattern, refine it, deploy across the board.
> Last updated: 2026-06-26

## 1. The problem with `type Option func(*options)`

The classic functional-option pattern allocates, for two compounding reasons:

1. **Each `With*` call builds a closure** that captures its argument (`func(o *options){ o.x = n }`).
2. **The mutable `*options` escapes.** The constructor does `var o options; for _, a := range opts { a(&o) }`
   — `&o` is handed to opaque function values, so escape analysis gives up and heap-allocates it.
   The variadic `[]Option` backing array escapes too.

Measured (self-contained micro-bench, `go1.25`, amd64):

| pattern                          | ns/op | B/op | allocs/op |
|----------------------------------|------:|-----:|----------:|
| `func(*options)` functional opts | 29.1  | 24   | **1**     |
| immutable value-builder          | 0.46  | 0    | **0**     |

The writer tried to dodge the allocation by **pooling the options struct itself**
(`poolOfBufferedOptions`, `poolOfYAMLOptions`, …). That works but is the "too complex play"
noted in the brief: every constructor borrows an `*options` from a pool, applies the closures,
stores the pointer on the writer, and must later `Reset()` + `redeem()` + `Redeem()` it — plus a
`runtime.AddCleanup` finalizer in the `New*` path purely to return the pooled options to its pool.
And it still allocates the closures on every `With*` call.

## 2. The fix (store precedent): immutable value-builder

`stores/default-store/options.go`:

```go
type Options struct{ resolved options }          // exported builder wrapping unexported config
func DefaultOptions() Options { return Options{resolved: options{ /* non-zero defaults */ }} }
func (o Options) WithArenaSize(n int) Options { o.resolved.minArenaSize = n; return o } // value rx
func New(opts ...Options) *Store { return &Store{options: resolveOptions(opts)} }
func resolveOptions(opts []Options) options {     // last wins; empty => defaults
	if len(opts) == 0 { return DefaultOptions().resolved }
	return opts[len(opts)-1].resolved
}
```

Why it is zero-alloc: `Options` is a plain value. The chain
`DefaultOptions().WithArenaSize(8).WithEnableCompression(false)` is all value copies on the stack —
**no closures, nothing escapes**. The variadic `[]Options` with a single element does not escape
(`resolveOptions` only reads `opts[len-1]` and returns a copy), so its backing array stays on the
stack too. No pool, no `Reset`, no finalizer.

## 2b. Recommended form: value-threaded functional option (keeps `options` private)

The store's value-builder works but forces exporting `Options` + `DefaultOptions`. The grudge with
that is real: `func(*options)` kept the entire config **unexported** (only the `WithX` funcs leak).
We can keep that encapsulation *and* be zero-alloc by threading the value instead of a pointer:

```go
// today: allocates — &o escapes, so the writer pools the struct to compensate
type BufferedOption func(*bufferedOptions)
func WithBufferSize(n int) BufferedOption {
	return func(o *bufferedOptions) { o.bufferSize = max(n, minBufferSize) }
}

// recommended (variant C): zero-alloc, bufferedOptions stays fully unexported, nothing new exported
type BufferedOption func(bufferedOptions) bufferedOptions
func WithBufferSize(n int) BufferedOption {
	return func(o bufferedOptions) bufferedOptions { o.bufferSize = max(n, minBufferSize); return o }
}
```

Call sites are unchanged: `NewBuffered(w, WithBufferSize(8192))` still compiles verbatim. Defaults stay
internal (set in the constructor). Measured: **0 B/op, 0 allocs/op** (4.6 ns).

**Why the captured closure does not allocate** — the question everyone asks. A closure capturing `n`
*would* heap-allocate if it escaped. It doesn't, because:
1. `WithBufferSize` is a one-liner → it **inlines** into the call site.
2. The closure literal is then created *at the call site*, not inside `WithBufferSize`.
3. Escape analysis there proves the constructor only applies (ranges + calls) and never stores the
   option → closure environment is **stack**-allocated.

`-gcflags=-m` evidence:

```
inlining call to WithSizeC        // step 1
... argument does not escape      // the variadic []Option: stack
func literal does not escape      // the closure: stack  → 0 allocs
```

vs. the current pointer form, which prints `moved to heap: c` — the escaping struct that is the 1 alloc.

**The one caveat:** zero-alloc requires the `With*` setter to be **inlinable**. One-line setters always
are. A setter too complex to inline (cost > 80) would build its closure internally and return it →
escape → 1 alloc. This is the single robustness difference vs. the store's value-builder, which has no
closures at all and is therefore zero-alloc regardless of inlining.

### Two skins of one principle

The portable principle is: **config is a private value, threaded — never a pointer handed to
closures — and never pooled.** Two API skins implement it:

| skin | exported surface | when to use |
|------|------------------|-------------|
| **C — value-threaded functional** (`Option func(o) o`) | only the `WithX` funcs | **default.** writer, lexer, the ~15 other `func(*options)` packages. Keeps config private. |
| **value-builder** (`Options` + `DefaultOptions` + `With*` methods) | `Options`, `DefaultOptions`, methods | only when callers need a **first-class, storable** config value (store: dict injection, gob round-trip, explicit reuse). |

Default to C. Reach for the value-builder only when the exported config value earns its keep. The store
stays as-is (its `Options` is deliberate public API).

## 3. The canonical recipe (what we standardize on)

For a component `Foo` with config:

```go
// exported immutable builder; unexported resolved config embedded into the component
type FooOptions struct{ resolved fooConfig }
type fooConfig struct { /* only comparable config: ints, bools, strings, caller-owned slices */ }

func DefaultFooOptions() FooOptions { return FooOptions{resolved: fooConfig{ /* defaults */ }} }
func (o FooOptions) WithBar(n int) FooOptions { o.resolved.bar = n; return o }   // value receiver

func resolveFooOptions(opts []FooOptions) fooConfig { /* last-wins / defaults */ }

func NewFoo(/*deps*/, opts ...FooOptions) *Foo { f := &Foo{cfg: resolveFooOptions(opts)}; … }
```

Rules that keep it zero-alloc and clean:
- **`With*` are value receivers** returning the copy. Never pointer receivers.
- **Config holds only plain values.** No `func` fields, no pooled handles — see §4.
- **Defaults live in `DefaultFooOptions()`**, not in zero values, when any default is non-zero.
- **Constructors take `opts ...FooOptions`, last wins.** Keeps `NewFoo(dep)` = defaults ergonomics.
- **The component embeds `fooConfig` by value** (`type Foo struct { fooConfig; … }`), so internal
  code reads `f.bar` directly and `Reset` is a single value assignment.

## 4. Refinements over the store's current form

Three improvements to bake into the standard, beyond a literal copy of the store code:

1. **Separate config from runtime state (the writer's core problem).** `bufferedOptions` today mixes
   `bufferSize` (config) with `redeemBuffer func()` (a *pooled* resource handle). The func field is
   exactly what forces pointer-sharing and pooling. In the new pattern the redeem handle moves to the
   writer instance (`buffered.redeemBuffer`), and `bufferedConfig` becomes a pure comparable value.
   (Contrast: the store keeps a lazily-built `cw flateWriter` *inside* resolved options — that is OK
   there because it is built once and lives and dies with the store, never returned to a pool. The
   writer's buffer handle is pool-borrowed, so it must live on the instance and be redeemed.)

2. **Prefer `string` over `[]byte` for textual config** (`indent`). Keeps the config comparable and
   dodges the `[]byte("  ")` default allocation; convert/﻿write as needed at the use site.

3. **Optional shared generic resolver** instead of copying `resolveFooOptions` into every package:
   ```go
   // package options (new, tiny) — or swag
   func Resolve[O any](opts []O, def func() O) O {
       if len(opts) == 0 { return def() }
       return opts[len(opts)-1]
   }
   ```
   `def` is passed by name (`DefaultFooOptions`), a static func reference — no closure alloc. This is
   sugar; per-package 4-line resolvers are equally fine. 🔬 decide whether the shared helper earns
   its keep.

Sharp edge to document loudly (inherited from the store): `New(dep, A, B)` **silently ignores A**.
Options: (a) document "last wins" and move on (store's choice), or (b) `panic` when `len(opts) > 1`
to catch misuse. Leaning (a) for consistency. 🔬

## 5. Application to `default-writer`

Four option types today, all pooled: `bufferedOptions`, `unbufferedOptions` (empty),
`indentedOptions`, `yamlOptions`. Target shape:

| today                                   | becomes                                                            |
|-----------------------------------------|--------------------------------------------------------------------|
| `BufferedOption func(*bufferedOptions)` | `BufferedOptions` value-builder; `bufferedConfig{ bufferSize int }` |
| `WithBufferSize(n) BufferedOption`      | `(BufferedOptions).WithBufferSize(n) BufferedOptions`              |
| `redeemBuffer func()` in options        | field on `buffered` instance (runtime state, not config)          |
| `WithIndentBufferedOptions(...Option)`  | `(IndentedOptions).WithBuffered(BufferedOptions)` (nested value)   |
| `unbufferedOptions struct{}` (+ pool)   | drop entirely — `NewUnbuffered`/`BorrowUnbuffered` take no opts¹    |

¹ `Unbuffered` has zero knobs; its option type/pool exist only to satisfy the old machinery.

**What gets deleted:** `poolOfBufferedOptions`, `poolOfUnbufferedOptions`, `poolOfIndentedOptions`,
`poolOfYAMLOptions`; every `*Options.Reset()` / `.redeem()` / nil-check; the `runtime.AddCleanup`
options-finalizer in `NewBuffered`; the recycled-vs-new option juggling in `BorrowIndented`/`BorrowYAML`.
The writer's real resource pools (`poolOfBuffers`, number/read/escaped buffers, and the writer-struct
pools `poolOfBuffered` …) **stay** — they pool genuine reusable buffers, not config.

**Borrow path becomes trivial:** `w.cfg = resolveBufferedOptions(opts)` (value assignment); then
`w.borrowBuffer()` using `w.cfg.bufferSize`; `redeemBuffer` recorded on the instance. `Reset` clears
`w.buf` and reassigns defaults — no pool traffic.

This is also a small **perf + correctness** win on the constructor path (drops the per-`New` options
alloc and the finalizer registration) on top of the simplification.

## 5b. Prototype results (branch `writer-options-zeroalloc`)

All four writer option types converted to variant C. Measured:

- **0 allocs** on the Borrow path *with* an option: `BorrowBuffered(w, WithBufferSize(8192))`.
- **The option adds 0 allocs on the New path too**: `NewBuffered(w)` and `NewBuffered(w, WithBufferSize(8192))`
  allocate the same count (the writer + working buffer are inherent to New; the option closure is free).
  This required keeping `bufferedOptionsWithDefaults` inlinable — dropping a redundant post-loop guard
  took it from cost 86 (not inlined → +1 closure alloc) to inlinable (→ 0). Lesson: **the WithDefaults
  resolver must stay under the inline budget**, same discipline as the `With*` setters.
- Throughput unchanged (canada ~1196 MB/s, citm ~544 MB/s; 0 allocs) — the hot write path never touches
  options, and `w.bufferSize` is now a direct field rather than a pointer deref.
- Suite + `-race` + benchmark round-trip ×4 green; `go generate` still idempotent.

**Deleted machinery:** all four `poolOf*Options`; every `*Options.Reset()`/`.redeem()`; the
`runtime.AddCleanup` options-finalizer in all three `New*` constructors; the `if opts == nil` /
nil-pointer juggling in the Borrow paths. `redeemBuffer` moved from `bufferedOptions` (config) onto the
`buffered` instance (runtime state) — the separation that made the config a plain value. Net **−108
lines** in `default-writer` (options + pools machinery alone: −114/+63).

**One gotcha found & fixed mid-prototype:** I initially nil-ed `redeemBuffered` in `Indented.redeem`/
`YAML.redeem` (looked tidy). That broke working-buffer recycling — a recycled Indented/YAML reuses the
same inner `*Buffered` across cycles, so the handle must persist; nil-ing it made every later Redeem
skip returning the buffer (4096 B leaked per cycle). The alloc-budget regression tests caught it.

## 5c. Store conversion (branch `writer-options-zeroalloc`)

`default-store` dropped the exported value-builder (`Options` + `DefaultOptions` + `With*` methods) for
variant C: `Option func(options) options` with free `WithArenaSize`/`WithEnableCompression`/
`WithCompression{Level,Threshold,Dict}` funcs. `options` is now fully unexported; nothing new is
exported beyond the `WithX` funcs. Constructors became `...Option`. Blast radius outside the package was
**zero** — the only `json`-package construction is `store.New()` (no args). gob is unaffected (it
serializes the unexported `options`, not the API type). Semantic change: options now **compose**
(apply-all, left to right) instead of the old last-wins; no caller relied on last-wins. Suite + `-race`
+ lint clean.

**Pointer/slice/interface inside the config — does it break inlining? No.** The store's `options`
embeds a `dict []byte` and a `cw flateWriter` interface. **Every option is zero-alloc**, including the
slice-bearing `WithCompressionDict` — copying a struct with pointer/slice/interface fields is a header
copy, no allocation, and the fields don't affect the setter's inline cost.

The one option that *initially* cost +1 alloc was `WithCompressionLevel`, and not because of any
pointer: its eager `panic(fmt.Errorf(...))` validation pushed the *outer* func over the inline budget, so
its closure was built inside it and escaped. **Resolved by replacing the panic with a silent clamp**
(`level = min(max(level, flate.HuffmanOnly), flate.BestCompression)`) — out-of-range levels are clamped
into the valid range instead of panicking. That keeps the func inlinable (→ 0 alloc) and removes the
`fmt` dependency. Lesson reinforced: **anything that blocks inlining (panic+fmt, oversized body) reintroduces
the allocation** — keep `With*` setters and the resolver trivial.

**Same discipline as the writer:** the resolver must stay inlinable. `optionsWithDefaults`'s large seed
struct literal initially blocked inlining (→ the variadic slice escaped, +1 alloc for *any* option). Lift
the seed to a package var (`defaultStoreOptions`, copied by value, never mutated) → resolver inlines →
non-validating options are all 0-alloc.

## 6. Migration surface

- Public API break: `NewBuffered(w, WithBufferSize(8192))` →
  `NewBuffered(w, writer.DefaultBufferedOptions().WithBufferSize(8192))` (and the Indented/YAML forms).
- Call sites in-repo are few: most consumers (`json/options.go`, `json/dynamic`, `json/nodes/light`)
  call `Borrow*` with **no** options and are unaffected. Only sites passing `WithBufferSize`/`WithIndent`
  change (~handful, mostly the writer's own tests). Full sweep needed before flipping.
- The repo is pre-1.0 (go-openapi v2 programme), so the break is acceptable; do it package-by-package.

## 7. Rollout order (smallest blast radius first)

1. `json/writers/default-writer` — the motivating case; validates the nested-options refinement.
2. `json/lexers/default-lexer` (+ `lab`) — same `WithBufferSize` shape; high-value, perf-sensitive.
3. The leaf config packages still on `func(*options)` (~15: `jsonschema/*`, `codegen/*`, `genmodels/*`,
   `spec/faker`, `json/options.go`, `json/dynamic`, …) — mechanical, lower stakes; can be scripted.

Each step: convert, delete the options pool(s), run suite + `-race` + the alloc tests, confirm 0
allocs on the constructor path with a `testing.AllocsPerRun` guard.

## 8. Open questions (for review)

- 🔬 Shared generic `Resolve` helper + a tiny `options` package, or per-package 4-liners?
- 🔬 `last-wins` silently, or `panic` on `len(opts) > 1`?
- 🔓 Keep the exported-wrapper / unexported-config split (`Options{ resolved }`) everywhere for
  encapsulation + embedding, or collapse to a single exported struct with unexported fields where a
  component doesn't need to embed the config? (Writer wants the embed → keep the split there.)
- 🔓 Naming: `DefaultBufferedOptions()` vs a package-level `BufferedDefaults` var (var would be mutable
  global — avoid; prefer the func). Confirm `FooOptions` vs `Options` naming when a package has several
  option families (writer has 4 → needs the `Buffered`/`Indented`/`YAML` prefixes; store has 1 → bare
  `Options`).
- 🔭 Scope check: deploy only to the hot/perf packages (writer, lexer, store ✓ done), or truly
  across all ~18 `func(*options)` sites for consistency even where allocs never mattered?
```
