# POC — composable on-the-fly validator of JSON Schema

Status: **DESIGN / for reflection** — nothing implemented. This captures the framing we converged on so it
can be reviewed before any code. Author: design dialogue (Fred + Claude), 2026-06-27.

---

## 0. Goal & UX

Build **specialised versions of a `json.Document`** (each *embeds/contains* a `json.Document`) that use the
decode **hooks** to compose a **parser-validator**. The user experience collapses the classic two stages
into one:

```
// today (go-openapi/spec style — what we move away from):
json.Unmarshal(&spec)        // unmarshal into Go types
spec.Validate()              // then validate

// target:
var spec Spec               // Spec embeds json.Document (jsonschema | openapi | overlay | …)
err := spec.DecodeJSON(r)   // decode + validate in ONE pass
// err == nil  ⟺  the document is well-formed AND valid against its grammar
```

Key properties:

- **No Go-type unmarshalling.** The immutable `json.Document` *is* the model; we don't mirror the spec into
  Go structs (the go-openapi/spec approach we are leaving behind). After a green decode you have **both**
  "it's valid" **and** a queryable, immutable Document.
- **Enrichment rides the same stream.** Hooks may collect decisive information and hydrate derived
  accessors (e.g. `OpenAPI.Operations()`), stored as **side-indexes** on the specialised Document. The
  Document stays immutable; derived data is redundant — that's fine.
- **Scope boundary.** This is for *plugging a validation layer into an arbitrarily complex document we do
  not want to unmarshal* (jsonschema, openapi, overlays, …). It is **NOT** a general-purpose
  "define a JSON Schema for a Go type and validate instances against it" engine — that is the **OP-program
  / `validations` Analyzer** track (compiled, strategy-driven), a separate design. `jsonschema` is the
  hardest case and exhibits all the hurdles, so it is the POC target.

---

## 1. Why NOT the meta-schema approach

We have done meta-schema validation before (go-openapi/v1: a draft-4 validator equipped with the
meta-schema; openapi-v2 = the openapi meta-schema + extra out-of-schema rules). Fred's verdict — the
meta-schema route is enticing but, in practice:

| meta-schema pain | the composable micro-parser model |
|---|---|
| slow / resource-intensive | single pass over the token stream; a small frame stack; no generic schema-matching machinery |
| poor error messages | each micro-parser knows its exact context → e.g. "`required` must be an array of unique strings, got number at `/properties/x/required`". Specificity is *free* because the parser is specific |
| multi-pass (resolve **and** expand every `$ref`) | validation never expands `$ref`. Structural check now, record the ref, resolve lazily/later — decoupled |
| rules that live *outside* the schema (even jsonschema has "unwritten" rules) | first-class: an `OnExit` check or a bespoke micro-parser, not an awkward bolt-on |
| cannot select the dialect **dynamically from the input** (the dream) | a discriminator-sniffing root that *binds* the grammar mid-parse (§5) — impossible with a fixed meta-schema |

Philosophy: we already have a **production engine** (the lexer → tokens; the node decoder → walks them).
The remaining problem is **how to compose the rules** as composable micro-parsers riding that engine.

---

## 2. The core hurdle — context-sensitivity over a context-free stream

The hook stream hands us `(key, token | node, depth)` per value. But JSON Schema is **context-sensitive**:
the meaning of a node depends on the *production* it sits in, not on its key or depth.

- value under `properties/X` → **a schema**
- value under `properties` → **a map of schemas**
- value under `required` → **an array of strings**
- value under `type` → **a string or an array of type-names**
- `$ref`'s value → **a reference string**

Same token kinds, different meanings. So the validator is a **pushdown automaton (PDA)**: it tracks "which
production am I inside" and pushes/pops as it descends/ascends.

The enabling fact: **the decoder already recurses, and `OnEnter`/`OnExit` ride that recursion.** The PDA is
just a frame stack pushed on container-`OnEnter` and popped on container-`OnExit`; its depth mirrors the
decoder's. State lives in `ctx.X` (the per-decode scratch designed for this; the pilot `constrained`
already used it for `octx`).

Micro-parsers **observe** the decode stream. They never drive the lexer — the deliberate move away from the
botched `jsonschema.beforeKey`, which sub-decoded inside the hook.

---

## 3. The composition model

The central split: **immutable grammar graph** vs **per-activation state**.

```
Parser     (immutable; wired once into a graph)
           - expected token kind(s) at this node
           - a Dispatcher for its children
           - zero or more OnExit checks (required keys, mutual exclusion, $ref-sibling rule, …)

Dispatcher (routes a node's children to parsers)
           - object: keyword → Parser map + fallback (unknown → anyValue | error)
           - array : one Parser for every element
           - "all values are schemas" (properties, $defs, patternProperties, items, allOf elements)

Frame      (per-activation; pooled; lives on the engine stack in ctx.X)
           - the active Parser + accumulated state (keys seen, counts, collected refs)

Engine     (IS the OnEnter/OnExit hook)
           - enter: parent frame's Dispatcher picks the child's Parser; validate token kind; push a Frame
           - exit : run the frame's Parser's OnExit checks; pop

Vocabulary (a bundle of keyword → Parser bindings + checks, merged into a Dispatcher)
Document   (embeds json.Document; wires engine + root Parser + merged vocabularies + observers; DecodeJSON)
```

Composition then has crisp answers:

- **Dispatch** = the Dispatcher (keyword map / uniform). Vocabularies *merge* their keyword maps
  (core ∪ applicator ∪ validation ∪ metadata ∪ format). openapi reuses the jsonschema `schema` Parser under
  its schema-shaped keywords, plus its own top-level Dispatcher.
- **Recursion** = a cycle in the Parser graph: `schema → properties → mapOfSchemas → schema`. Built once,
  reused everywhere a schema may appear. `mapOfSchemas` is one Parser shared by `properties` / `$defs` /
  `patternProperties`.
- **Cross-member rules** = the Frame accumulates "keys seen" / counts; the Parser's `OnExit` checks fire
  with that state (`$ref`-sibling rule, `if`/`then`/`else`, `dependentRequired`, `properties` ×
  `additionalProperties`).
- **Document-global rules** (operationId uniqueness, every `$ref` resolves) = enrichment collects into the
  Document during the walk; the **root `OnExit`** validates invariants over the collection. This is where
  validation and enrichment **unify**: enrichment gathers, root-exit checks.

Why error quality is *free*: a specific micro-parser emits a specific message with the JSON-Pointer path
(CTX-1 already attaches the path to hook errors). No expansion, no generic "does not match" diffs.

---

## 4. The hurdles, and where each lands

| # | hurdle | disposition |
|---|--------|-------------|
| H1 | context-sensitivity (the role problem) | PDA frame stack in `ctx.X` — the core mechanism |
| H2 | `$ref` / forward & lazy resolution | validate the ref *string* on the fly; **record** refs; resolve targets at root-`OnExit` (whole `$defs` known) or defer to the lazy-resolution session. Never expand inline |
| H3 | draft/dialect variance (4 → 2020-12, OAS dialect) | version-parameterised vocabularies; dynamic detection in §5; POC pins one draft |
| H4 | cross-member rules | Frame state + Parser `OnExit` checks |
| H5 | enrichment on the same stream (`Operations()`) | side-indexes on the Document, built by observers; immutable doc, redundant derived data is fine |
| H6 | allocation | `PoolRedeemable` frames; state in `ctx.X` |
| H7 | error model | **decision** — collect-all vs fail-fast (§6) |
| H8 | unknown keywords / `x-*` extensions | keyword miss → `anyValue` (lax) or error (strict), vocabulary-configurable; `x-*` captured as extensions |

---

## 5. The dynamic-dialect dream — believed reachable

Discriminators (`$schema`, `openapi`, `swagger`) are top-level scalar keys.

1. The root Parser starts in **detect** mode with a tiny dispatcher that recognises the discriminator.
2. When the discriminator value arrives, it **binds**: selects the dialect's vocabulary set and installs
   that Dispatcher on the root frame for subsequent members.
3. Members that arrived *before* the discriminator (their Nodes are already in the tree) are **buffered**
   `(key, node)` and **replayed** through the bound grammar once bound.

For well-authored specs the buffer is empty (`$schema` / `openapi` come first). For adversarial ordering we
validate a few already-built sub-Nodes after binding — correct, just slightly less streaming for those.
Net: **detect → bind → replay-the-prefix**; everything after the discriminator streams. It falls out
naturally because the Document is being built regardless.

---

## 6. Open design decisions (these shape the types — settle before code)

1. **`Parser`: interface vs struct-descriptor?** Leaning **struct descriptor** (`{kind, dispatcher,
   checks}`) — *data, not code* — so it is composable/mergeable and could later be generated from a source
   or compiled to the OP-program. Bespoke parsers slot in as a descriptor carrying a custom check func.
2. **Error model: collect-all vs fail-fast.** For specs, **collect-all** (hooks return `Continue`,
   accumulate path-tagged errors in the engine, `DecodeJSON` returns a joined set, with a cap) is far more
   useful; fail-fast as a mode. Ripples through every micro-parser, so decide early.
3. **Enrichment: observers vs woven into parsers?** Leaning a small **observer list** the engine fans out
   to (separate from the grammar) so `Operations()` hydration doesn't tangle with validation; document-
   global *checks* read the enrichment at root-exit.
4. **Frame state & pooling** — `PoolRedeemable` frames; state = keys-seen set + collected refs + counters.
5. **Vocabulary merge policy** — keywords are mostly disjoint; define union with last-wins or
   error-on-conflict.

The **`Parser` / `Dispatcher` / `Frame` triad is the whole ballgame** — iterate on these type signatures on
paper before writing the engine.

---

## 7. Proposed POC scope

Prove the engine + composability + the one-pass UX on a real-ish schema, minimally:

1. The PDA engine (frame stack in `ctx.X`; push/pop on container enter/exit) as the `OnEnter`/`OnExit` hook.
2. A small vocabulary subset: `type`, `properties`, `items`, `required`, `enum`, `minimum`/`maximum`,
   `$ref`, `$defs` — enough to exercise schema ↔ mapOfSchemas ↔ arrayOfSchemas ↔ stringArray recursion.
3. A `Schema` specialised Document with `DecodeJSON(r) error` (nil ⟺ well-formed).
4. `$ref` recorded + structurally checked; target resolution at root-`OnExit`.
5. One enrichment example proving the dual use (e.g. expose `Defs()` / collected `$ref` targets via a method).
6. Negative tests with **precise messages + paths**: wrong `type` value, `required` with non-strings,
   a `properties` value that isn't a schema, a dangling `$ref`.

Pin draft 2020-12 for the POC (defer dynamic dialect §5 to a later step, but keep the engine able to rebind
a Dispatcher so §5 is not designed out).

---

## 8. Relationship to other tracks

- **OP-program / `validations` Analyzer** (separate): compiled, strategy-driven validators for instances
  against a schema. This POC is *spec well-formedness*, not instance validation. The micro-parser grammar
  here could *later* be compiled toward an OP-program — keep the engine grammar-agnostic so that door stays
  open, but it is out of scope for the POC.
- **Lazy `$ref` resolution session** (store/document level): the validator only *records* refs; resolution
  is the session's job (one store per working session: spec + overlays + resolved refs).
- **Depends on** the decode-hook redesign (`OnEnter`/`OnExit` + `HookEvent` + `Action`), currently on the
  `nodes/inspection` branch (HEAD `d52cac0`, not yet merged).

---

## 9. Next steps (basecamp — pending Fred)

- Merge the completed `nodes/inspection` work to master (it carries the hooks the POC needs).
- Open a `jsonschema/poc` worktree/branch.
- Iterate on the `Parser` / `Dispatcher` / `Frame` type signatures **on paper** (this doc), then build the
  engine, then the first vocabulary slice, then the `Schema` Document + tests.
- This doc is the reviewable starting point; revise it as the type design firms up.
