# Ramblings — the go-openapi v2 programme (why the lexer is shaped the way it is)

> Context captured 2026-06-23. The default-lexer is the lowest building block of
> a much larger programme; these constraints explain its design priorities and
> where our perf research re-applies later. Companion to
> [perf/paradigm](2026-06-perf-and-paradigm.md) and [codegen/asm/jit](2026-06-codegen-asm-jit.md).

## Primary use case

The lexer parses **OpenAPI spec documents**, which can be **enormous** (Google,
Azure APIs). Advanced in-memory processing of such docs is the real challenge —
hence the obsession with **low memory footprint and low GC pressure**, not raw
MB/s. Correctness and **no loss of accuracy** come first.

## What the lexer feeds (consumers in the suite)

- **A general-purpose "JSON Document"** — a hierarchical structure.
- **A JSON "store"** — a compact memory store backing the Document, with
  compacted/compressed data in a **memory arena** (conceptually like a thing
  buried deep in the Kubernetes machinery, but standalone).
- **A JSON "writer"** — streams JSON efficiently; optionally YAML (less
  efficiently), pretty-JSON, HTML-escaping, etc. (in the spirit of easyjson's
  jwriter).
- **The Verbatim lexer (`VL`)** exists because some tools — **TUIs, LSPs** —
  need *full* fidelity (positions, original bytes, blanks). That is why
  positions are always-on and why `VL` is a first-class flavour, not an option.

## Design priorities (in order)

1. Correctness + no accuracy/precision loss (we never convert numbers).
2. Low memory, low GC pressure (huge docs in memory).
3. Performance "as best we can"; **jsontext v2 is the yardstick** (non-SIMD).
4. No SIMD in the core. (Surprise of the project: it took the Go team 12+ years
   to ship a decent tokenizer — jsontext is genuinely good.)

If/when **jsontext leaves experimental**, a legitimate end-state is to **wrap it**
for the non-verbatim path and stop there. We are **not competing with serializers**
(jsoniter, goccy, sonic) — only the pure lexer/parser part.

## How serialization is offered to users (the codegen story)

The new codegen lets API authors choose their serialization mechanism:

- **Untyped**: our **Document** (API via JSON Pointer / JSON Path) — a better
  alternative to today's `map[string]any`-nesting `any` runtime
  (cf. go-openapi/runtime v1).
- **Generated serializers** (as go-swagger does today), built over a serializer
  of the user's choice: stdlib, easyjson, goccy, sonic, jsoniter. We already have
  useful wiring in **go-openapi/swag/jsonutils**: a runtime-registered multi-path
  for serialization (currently stdlib + easyjson). We may keep extending that or
  replace it.

A possible **SIMD-lexer** (and similarly **JSON-LD** support) is useful **only for
the untyped runtime usage**, not the core.

## Where our perf/asm/JIT research re-applies later

Among by-products of the JSON & JSON Schema infrastructure: a **schema
validator** that supersedes the venerable go-openapi/validate with a new concept —
an **"OP program" compiled from the schema** along an optimized path. Per our
research, that program could be:

- **pure Go codegen** (like goccy's unmarshaler, or the `regexp` package's
  program), or
- a **mix with JIT assembly** (like sonic).

This is exactly where the **Road 1 (generate-from-golden-source)** and even
**Road 3 (JIT)** ideas from the codegen ramble become live again — "compile a
schema into an optimized validation program" is the same family as "compile a
type into an optimized decoder." **Not there yet** — but the research is banked
for it.
