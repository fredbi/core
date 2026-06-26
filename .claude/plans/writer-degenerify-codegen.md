# writegen — concretize the generic writer onto leaf writers

> Status: **DRAFT for review** — iterate before building the tool.
> Scope: `json/writers/default-writer` + new `internal/writegen` generator.
> Last updated: 2026-06-25

## Progress

- ✅ **Investigation** — root-caused the two inlining gates; proved the fix; benchmarked.
- ✅ **Build `writegen`** — AST lifter (receiver-swap only), dependency-free imports, in-place
  injection, idempotent (stable `markerDetect="writegen"` token, decoupled from display text).
- ✅ **Wire `//go:generate`** into `buffered.go` + `unbuffered.go`; 30 methods lifted each;
  prototype removed. Build + suite + `-race` + round-trip ×4 green; 0-alloc preserved; inlining
  confirmed (17 `writeSingleByte` inline sites). **Perf: on par with easyjson at 0 alloc** (steady
  state, ±~10%, trading the lead per workload) — up from ~20–45% behind.
- ✅ **Lint — `dupl`**: resolved with a file-level `//nolint:dupl` (+ explanation) on the three
  files (`base_writer.go` + the two targets) — the triplication is by design. godoc/err113/mnd
  fixed. Package lints clean (only pre-existing yaml/indented/gocyclo remain, untouched).

**Complete.** `go generate ./...` regenerates idempotently; the directive survives regeneration.

## Legend

- ✅ done · 🚧 in progress · ⏳ planned · 🔬 needs decision

---

## 1. Why

`commonWriter[T wrt]` wraps the byte primitives generically. Calls like `w.jw.writeSingleByte(comma)`
compile to a **type-parameter dictionary call** (`(&.dict[0])(w.jw, …)`) that the compiler does **not**
devirtualize or inline (gate 2). So the structural hot path (delimiters, string quotes, `Token`
dispatch) never inlines `writeSingleByte`, even after it was made cheap enough to inline (gate 1).

Measured: lifting the structural ops onto the concrete `*Buffered` (receiver swap) made `writeSingleByte`
inline and pushed **our-buffered past easyjson on all 4 corpus datasets at 0 alloc** (canada 779→1112
MB/s vs easyjson 924; citm +24%; golang +18%; twitter +19%). String-heavy sets have more headroom once
`writeText` is concrete too — hence **lift everything**, not a subset (nested generic calls cascade:
generic `String` calls generic `writeText` regardless of a lifted `writeText`).

## 2. The mechanism (validated)

A lifted method is the generic body **verbatim**, with **only the receiver type changed**:

```go
// commonWriter (source of truth)
func (w *commonWriter[T]) Comma() { w.jw.writeSingleByte(comma) }

// injected into buffered.go
func (w *Buffered) Comma() { w.jw.writeSingleByte(comma) }   // identical body
```

No body rewriting. On `*Buffered`, the embedded `commonWriter[*buffered]` field `jw` is statically typed
`*buffered`, so `w.jw.writeSingleByte` re-resolves to a concrete (inlinable) call. Confirmed with
`-gcflags=-m`: 12+ `inlining call to (*buffered).writeSingleByte` sites, none for the generic version.

`commonWriter[T]` **stays embedded** (provides the `jw` field, remains the single source of truth). The
lifted concrete methods shadow the generic ones; the unused generic instantiation methods are
dead-code-eliminated.

## 3. Decisions (settled)

- **Lift-all**: every `*commonWriter[T]` method is stamped onto each target (incl. unexported helpers
  `writeText`, `writeTextString`, `append`, `appendFloat`).
- **Targets**: `Buffered`, `Unbuffered` only. `Indented`/`YAML` keep their hand-written structural
  methods (formatting) and are excluded; they already benefit by routing through the lifted `*Buffered`
  ops.
- **In-place injection** into the target's own file (`buffered.go`, `unbuffered.go`). No separate
  generated file, no file-level "DO NOT EDIT" banner.
- **Per-method marker**: each injected method carries a one-line generated notice, e.g.
  `// writegen: lifted from commonWriter.Comma — edit the source there, regenerate here.`
  Idempotent regen = strip every `*Target` func bearing the marker, re-inject the fresh set.
- **Trigger**: `//go:generate go run ./internal/writegen -target Buffered` atop `buffered.go`
  (and `-target Unbuffered` atop `unbuffered.go`).
- **No drift**: bodies are byte-for-byte the generic ones; editing `commonWriter` + `go generate` is the
  only way to change them.

## 4. Tool design — `internal/writegen/main.go`

1. Parse the package (`go/parser`, `ParseDir` on the package dir; the source-of-truth file is
   `base_writer.go`).
2. Collect every method whose receiver is `*commonWriter[T]` (FuncDecl with that receiver). Keep AST +
   the original source text span for verbatim bodies.
3. Parse the target file (`-target Buffered` → its declaring file, found by locating the `type Buffered
   struct`).
4. Strip previously-injected funcs (receiver `*Target` + marker comment).
5. For each collected method, rewrite the receiver type `*commonWriter[T]` → `*Target`, prepend the
   marker comment. Body unchanged.
6. **Imports**: scan the lifted bodies for package-qualified selectors (`pkg.Ident`), map each to its
   import path from `base_writer.go`'s import block, and union the missing ones into the target file's
   imports. (Dependency-free; avoids adding `x/tools` to `json/go.mod`. See 🔬 below.)
7. Append the methods, `format.Source`, write the file back.

Idempotent: running twice yields no diff (strip-then-reinject + stable ordering = method order in
`base_writer.go`).

## 5. Verification

- `go build` + `-gcflags=-m`: lifted methods inline `writeSingleByte`/`writeEscaped`; no `&.dict[…]`
  calls for the lifted set.
- `go generate ./...` twice → `git diff` empty (idempotency).
- Full `default-writer` suite + `-race`; `TestReplayRoundTrip` ×4 corpus in the benchmark module.
- Benchmark: expect string-heavy datasets (citm/golang/twitter) to rise further now that `writeText`
  quotes are concrete; canada already ~1.1 GB/s.
- Delete `delimiters_proto.go` (superseded).

## 6. Risks / edge cases

- **Generic refs in bodies**: `commonWriter` bodies use `T` only via `w.jw` — no other `T` usage to
  rewrite. Tool asserts this (error if a body references the type param otherwise).
- **Name collisions**: hand-written `*Buffered`/`*buffered` methods (`Reset`, `Flush`, `Size`, `flush`,
  `redeem`, `writeSingleByte`, `writeBinary`, `writeEscaped`, `borrowBuffer`) are disjoint from the
  `commonWriter` method set → no clash. Tool errors on any overlap.
- **Import removal**: tool only *adds* imports; a regen that drops the last user of an import could leave
  it unused. Mitigation: also drop imports no longer referenced by file (full union recompute), or accept
  and rely on `go vet`/lint. 🔬
- **Interface satisfaction** unchanged (same method sets, just concrete).

## 7. Open decision

- 🔬 **Imports**: dependency-free AST scan (§4.6, recommended) vs. running the output through
  `golang.org/x/tools/imports` (cleaner unused-handling, but adds a dep — would put `writegen` in its
  own nested module or tolerate the dep in `json/go.mod`). Leaning dependency-free.
