# Vendored JSON benchmark corpus

These are canonical JSON benchmark documents, widely used to compare JSON
parsers (nativejson-benchmark, simdjson, easyjson, encoding/json/v2, ...).

They are vendored here gzip-compressed (decompressed in-memory via the standard
library `compress/gzip`, so no extra dependency is pulled in) from the
`encoding/json/v2` reference implementation test corpus:

- Upstream: https://github.com/go-json-experiment/json
  (`internal/jsontest/testdata`, originally `.zst`; re-compressed to `.gz` here)
- License: BSD-style, Copyright The Go Authors (see that repository's LICENSE).

| file | shape |
|------|-------|
| `canada_geometry.json.gz` | number-heavy (geo coordinates) |
| `citm_catalog.json.gz` | objects and strings (real-world catalog) |
| `twitter_status.json.gz` | unicode-rich strings (social API payload) |
| `golang_source.json.gz` | deeply structured (Go AST dump) |

## Extended corpus (simdjson-go)

To measure the lexer on "untrained" ground — payloads it was NOT tuned against,
especially string-value-heavy documents the parity 4-set under-represents — the
following are vendored from the [minio/simdjson-go](https://github.com/minio/simdjson-go)
testdata (originally `.zst`, re-compressed to `.gz`). The %≥32 column is the
fraction of bytes living in string *values* of length ≥ 32 (the AVX2-string prize
metric; see the plan §9.3 / ramblings).

| file | shape | %bytes in str-values ≥32 |
|------|-------|--------------------------|
| `gsoc-2018.json.gz` | huge text fields (GSoC dump) | 80% |
| `github_events.json.gz` | event payloads, long text/URLs | 52% |
| `update-center.json.gz` | Jenkins plugin metadata | 37% |
| `apache_builds.json.gz` | build logs (32–63 B strings) | 37% |
| `twitterescaped.json.gz` | twitter with escaped unicode | 29% |
| `payload-large.json.gz` | mixed API payload | 22% |
| `random.json.gz` | mixed random | 2.5% |
| `marine_ik.json.gz` | number/structure-heavy | ~0% |
| `mesh.json.gz` | number-heavy (3D mesh) | 0% |
| `mesh.pretty.json.gz` | mesh, pretty-printed (whitespace) | 0% |
| `numbers.json.gz` | pure number array | 0% |
| `instruments.json.gz` | small structured | 0% |

Excluded: `parking-citations` (NDJSON — multiple top-level objects, not a single
JSON document); `payload-small`/`payload-medium` (< 1 KB, too small to benchmark);
`canada`/`citm_catalog`/`twitter` (duplicates of the parity set above).

## Updating

Re-fetch from a fresh checkout of the upstream corpus and re-compress:

    zstd -dc <name>.json.zst | gzip -9 > <name>.json.gz
