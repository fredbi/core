// Package simd holds a PROVEN-BUT-UNSHIPPED AVX2 string-stop scanner (§9.3).
//
// stringStopIndexAVX2 finds the first byte that ends a JSON string body — a
// control char (< 0x20), a double quote (0x22) or a backslash (0x5c) — scanning
// 32 bytes per iteration in a YMM register. In isolation it is 6–14x faster than
// the 8-byte SWAR scan for strings >= 64 bytes, but loses below 32 bytes (vector
// setup overhead), so any use must be length-gated.
//
// It is PARKED (see the plan §9.3 / "Parked" section): the corpus study showed the
// win is a margin-widener on string-heavy payloads we already lead, not a
// necessity, and it would add a permanent amd64 asm + CPU-gate + fallback surface.
// Kept here — as its own module so the avo generator dependency does not leak into
// the lexer — so the working experiment is not lost. amd64-only by design.
//
//go:generate go run asm.go -out stringstop_amd64.s -stubs stub_amd64.go -pkg simd
package simd
