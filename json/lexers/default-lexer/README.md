# Default implementation of a JSON lexer

A low-level JSON building block that **lexes without evaluating**. Concretely:

1. **No value evaluation.** Numbers and strings stay as `[]byte`; no numeric
   conversion, so no loss of resolution. Comparable in spirit to the experimental
   `encoding/json/v2` + `encoding/json/jsontext`.

2. **Zero unamortized allocation, bounded peak memory.** Everything lives in short-lived,
   recycled buffers. The only hard-coded literals are `null`/`true`/`false`.
   Peak memory ≈ max(read-buffer window, longest single value).

3. **Pluggable behind a common interface** (`json/lexers.Lexer`), so alternative
   implementations may be injected (incl. one on top of `encoding/json/v2`).

4. **Two flavors:**
   - **Semantic** (`L` → `token.T`): drops insignificant whitespace, normalizes UTF8 pairs.
   - **Verbatim** (`VL` → `token.T`): preserves whitespace/escapes; for
     linters / LSPs / formatters sensitive to exact positions.

5. **Streaming or buffered:** `io.Reader` (internally buffered) or `[]byte`.

## Getting started

TODO

## Design goals

We want a fast JSON lexer to feed our [json.Document] with the following requirements:

* support oldstable go - no GOEXPERIMENT, GODEBUG etc toggles
* zero allocation
* high throughput optimized for large strings (e.g. in GB/s, not MB/s)
* bounded memory (up to the largest ingested token)
* support for streams
* accurate error reporting and location (offset, error with surrounding text)
* short lived token, recycle any memory
* token knows its kind and value - other information (location, pointer, leading blanks...) are stored
  in the lexer's state
* JSON string escaping and UTF8 normalization:
  * `L`: the caller doesn't have to know these rules - strings are directly usable
  * `VL`: the caller wants unaltered strings, including escapes
* no loss of numeric precision: no conversion to native types
* push & pull iterators

Since we want it to be flexible, there are a few available options:

* optional knobs: some are available a toggles, some result from the choice of lexer `L` vs `VL`
  * security guards against overflows (max. depth, max. token length)
  * in streaming mode, ability to set the memory window being used
  * ability to elide semantically redundant separators (",", ":") from iterated tokens
  * ability to switch to "verbatim" mode, preserving non-significant blank, not escaping strings (that's our `VL` lexer)
  * verbatim mode tracks a token's line and column in the input text
  * option to tolerate numbers such as `+1` and `01`
  * option to track the json pointer of the current token **TODO: NOT IMPLEMENTED YET**
  * tunable context window for reporting errors

Additional objectives:

* reusable internal core for scanning tokens

Non-goals / out-of-scope:

* non-UTF8 encoding
* JSON canonicalization (RFC 8785)
* full SIMD implementation (à la simd-json)

## Design

  * push vs pull loops
  * fast paths
  * heuristics
  * inlining
  * generics
  * devirtualization
  * SWAR scanners
  * SIMD acceleration (AVX2): usage limited to fast-parse large strings (amd64 arch)

Differences with `encoding/json/v2`

* ❌ : no, never
* ⏸️ : yes, always
* ✅ : opt-in, enabled by default
* ⬜ : opt-in, disabled by default
 
|                | L   | VL  | jsontext |
|----------------|-----|-----|----------|
| token size     | 32b      ||  16b?    |
| only UTF8      | ⏸️  |  ⏸️ |  ⏸️      |
| number as bytes| ⏸️  |  ⏸️ |  ⏸️      |
| token has value| ⏸️  |  ⏸️ |  ⏸️      |
| sep. elide     | ✅  |  ⬜ |  ⏸️      |
| string escape  | ⏸️  |  ❌ |  ❌      |
| track ns space | ❌  |  ⏸️ |  ❌      |
| track line/col | ❌  |  ⏸️ |  ❌      |
| track pointer  | ❌       ||  ⏸️      |
| AVX2 acceler.  | ✅       ||  ❌      |
| push iterator  | ✅       ||  ❌      |    
| pull iterator  | ✅       ||  ⏸️      |
| limit stack    | ✅       ||  ✅      |
| limit tok size | ✅       ||  ❌      |    

Trade-offs when comparing to `github.com/go-json-experiment/json/jsontext` (stdlib `json/v2`).

* our token is larger (more memory traffic) but cheaper to consume (no extra indirection or escpaping)
* our decision to escape strings involves more work
* actual token usage is lighter (no indirection, no escaping: all done in the `L` lexer most efficiently)
* our fast-path is zero-alloc, zero-copy
* our heuristics are less efficient for:
  * small values (single digits, booleans only...)
  * densely escaped strings 
* other workloads generally show higher performances, sometimes much higher

Our lexer's fastest path is to use the push iterator (`Tokens()`) from a buffer of bytes.

Streaming mode degrades speed by 15-20%.

Pull iterator (`NextToken`) mode degrades speed by 10-15%.
  
## Conformance tests

Our implementation of the JSON lexers pass the full JSON conformance suite. No compromise on strictness.

## Examples

## Benchmarks

See ...

## Roadmap

* AV2 support is currently provided as assembly kernels for amd64 only
* AVX512 is likely overkill for our usage and I don't have the hardware to test it thoroughly
* this will be eventually replaced by go native support for AV2 & AVX512 (currently experimental)
