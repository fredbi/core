# Default implementation of a JSON lexer

A low-level JSON building block that **lexes without evaluating**. Concretely:

1. **No value evaluation.** Numbers and strings stay as `[]byte`; no numeric
   conversion, so no loss of resolution. Comparable in spirit to the experimental
   `encoding/json/v2` + `encoding/json/jsontext`.
2. **Near-zero allocation, low peak memory.** Everything lives in short-lived,
   recycled buffers. The only hard-coded literals are `null`/`true`/`false`.
   Peak memory ≈ longest single string/number value.
3. **Pluggable behind a common interface** (`json/lexers`), so alternative
   implementations (incl. one on top of `encoding/json/v2`, and a SIMD one) drop in.
4. **Two flavors:**
   - **Semantic** (`L` → `token.T`): drops insignificant whitespace, normalizes.
   - **Verbatim** (`VL` → `token.VT`): preserves whitespace/escapes; for
     linters / LSPs / formatters sensitive to exact positions.
5. **Streaming or buffered:** `io.Reader` (internally buffered) or `[]byte`.

## Getting started

## Examples


## Extra features

-  **3.1 Optional input normalization.** Make UTF-8 / escape processing
  toggleable (sanitizer hooks for strings and numbers, per old TODO).
- JSON canonicalization (RFC 8785).** Opt-in transform for strings and
  numbers. ⚠️ Number canonicalization (ECMAScript double) conflicts with the
  no-resolution-loss invariant — must be explicit opt-in, likely a higher layer. 🔬
- NDJSON support - Line-delimited JSON, especially for streaming `io.Reader`;
  top-level value sequence separated by `\n`.
- SIMD accelaration

## Conformance tests

## Benchmarks

## Alternate implementation
