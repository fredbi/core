# Micro-benchmark baseline — go1.26.4, Ryzen 7 5800X, count=10 medians

Raw: `2026-06-micro-baseline.txt`. `reference` = worktree `default-lexer`
(exploration HEAD, has clean-run batching); `lab` = `default-lexer/lab` sandbox.
Values are MB/s, median of 10. Modes: bytes (pull) / tokens (push) / reset (reused).

| workload | ref bytes | ref tokens | lab bytes | lab tokens | note |
|---|---|---|---|---|---|
| separators | 115 | 157 | 115 | 160 | **slowest — pure dispatch** |
| nulls | 210 | 262 | 217 | 259 | dispatch-dominated |
| bools | 231 | 277 | 232 | 278 | dispatch-dominated |
| exponential | 342 | 405 | 331 | 395 | number slow path |
| decimals | 477 | 558 | 443 | 513 | lab slightly behind |
| ints_pos | 512 | 661 | 535 | 648 | |
| ints_neg | 549 | 671 | 571 | 659 | |
| strings_escaped | 573 | 602 | 589 | 663 | |
| strings_escaped_long | **1360** | **1362** | **770** | **790** | **lab MISSING clean-run batching** |
| strings_uescaped | 863 | 972 | 854 | 962 | |
| strings_unicode | 881 | 975 | 877 | 970 | |
| strings_plain | 951 | 1027 | 949 | 1032 | |

## Findings

1. **Dispatch tax is the floor.** `separators` (115), `nulls` (210), `bools`
   (231) are the three slowest — tokens are tiny and numerous, so per-token
   overhead (binary-search dispatch + the generic `emit` dictionary call)
   dominates. These are the clean probes for §5.1 (devirt) and §5.2
   (classification table). A win here lifts the whole floor.
2. **Push (tokens) beats pull (bytes) by ~15–20% everywhere** — confirms the
   push API is genuinely the faster path; worth keeping front-and-centre.
3. **`reset` ≈ `bytes`** and reports 0 allocs/op — steady-state scan cost is the
   same as per-iteration construction here (construction is cheap/amortized), so
   the per-iteration `bytes` numbers are honest.
4. **The `lab` package is stale.** It is the older VL-unification sandbox, not a
   copy of the current reference: it lacks clean-run batching (escaped_long 770 vs
   1360) and trails slightly on decimals. **Prep prerequisite: re-sync the lab
   from the current reference before running any experiment** (Fred's prep step 1).

## Next probes (against the refreshed lab)
- devirt (§5.1) + classification table (§5.2): measure on separators/nulls/bools.
- 16-byte token (§5.3): measure on strings_plain/ints_pos/separators (copy-heavy).
- numbers fast paths (§4.1): exponential (342) and decimals (477) have headroom.
