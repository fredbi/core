# String workloads — baseline (go1.26.4, Ryzen 7 5800X, count=10 medians)

Raw: `2026-06-strings-baseline-go1.26.4.txt`. MB/s = median of 10. Two new
workloads added this round: `strings_escaped_long`, `strings_uescaped`.

| workload | our `bytes` (pull) | our `tokens` (push) | jsontext | push vs jsontext |
|---|---|---|---|---|
| `strings_plain`        | 958 | 1041 | 1397 | 75% |
| `strings_escaped`      | 571 | 614  | 797  | 77% |
| `strings_escaped_long` | 664 | 675  | **1779** | **38%** |
| `strings_uescaped`     | 875 | 978  | 607  | **161%** |
| `strings_unicode`      | 872 | 1018 | 641  | **159%** |

## Findings — the new workloads earned their place immediately

1. **`strings_escaped_long` is the real wound: 38% of jsontext (2.6× slower).**
   The short `strings_escaped` workload *hid* this (showed a benign 77%) because
   its elements are escape-dense and short — the byte-by-byte tail never gets
   long. With a long clean tail after a couple of escapes, our per-byte
   `append` loop is murderous while jsontext just runs a `noEscape` scan and
   *doesn't even unescape* (validate-only at tokenize time). **This is THE target
   for clean-run batching (§4.2–4.3).**

2. **We already BEAT jsontext on both Unicode workloads — by 1.6×.**
   `strings_uescaped` (\u + surrogate pair): us 978 vs jsontext 607.
   `strings_unicode` (literal multibyte): us 1018 vs jsontext 641. jsontext's
   tokenizer pays heavy canonical-form validation on `\u` (case checks, control
   vs non-control, surrogate validation, the `stringNonCanonical` flag joins) and
   `utf8.DecodeRune`s every multibyte rune; we decode `\u` cheaply and *alias*
   literal multibyte without decoding. **Conclusion: our `\u`/surrogate path is
   not a liability — it's an advantage.** This vindicates keeping the
   `unescapeUnicodeSequence` inlining work parked (§4.6) — there's no deficit to
   chase there.

3. **`strings_plain` (75%) and `strings_escaped` short (77%)** are dominated by
   per-token fixed overhead (dispatch, token build), not the scanner. Clean-run
   batching will nudge `strings_escaped` a little; `strings_plain` is road-b
   territory (de-genericization), out of scope.

## Net steer

- Clean-run batching is now justified by a **2.6× gap on `strings_escaped_long`**,
  not a vague 17% — the new workload converted a fuzzy hunch into a sharp target.
- The Unicode/`\u` optimization sub-thread (§4.6) is **de-prioritized**: we're
  already ahead. Revisit only if a real-world `\u`-dense corpus says otherwise.
- Regression canaries for the implementation round: `strings_plain`,
  `strings_unicode`, `strings_uescaped` must NOT regress (they're at/above
  parity); `strings_escaped` + `strings_escaped_long` must climb.
