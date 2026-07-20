// Generator-only module for the AVX2 string-stop kernel. It isolates the avo
// build-time dependency: only `go generate` (see ../scan_amd64.go) runs this, and
// the generated ../stringstop_amd64.s has no avo import, so the parent json module
// never pulls avo. Lives under a `_`-prefixed directory so the go tool ignores it
// when listing/building the parent module.
module github.com/fredbi/core/json/lexers/default-lexer/internal/strscan/_asm

go 1.25

require github.com/mmcloughlin/avo v0.0.0

require (
	golang.org/x/mod v0.27.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/tools v0.36.0 // indirect
)

replace github.com/mmcloughlin/avo => /home/fred/src/github.com/fredbi/avo
