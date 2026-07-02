// Own module to isolate the avo build-time dependency (needed only to regenerate
// the AVX2 asm via `go generate`; the generated .s + package have no avo import).
module github.com/fredbi/core/json/lexers/default-lexer/internal/simd

go 1.25

require github.com/mmcloughlin/avo v0.0.0

replace github.com/mmcloughlin/avo => /home/fred/src/github.com/fredbi/avo
