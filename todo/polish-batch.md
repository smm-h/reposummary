# Polish batch: header glyph, empty-window cache key, model alias pass-through

Three small approved fixes to land together in the next working batch on this repo.

## 1. Inconsistent arrow glyphs in the journal header

The date-window form's label renders ASCII `->` (e.g. `**2026-06-11 -> 2026-07-11**`)
while the same header line's date-range subtitle uses `→`. One-line fix in
`internal/window` (the label builder for the `<date>+<duration>` form): use `→`.

## 2. Zero-commit windows share one cache key

`cache.MakeKey(firstSHA, lastSHA, synthesis, model, version)` receives empty strings for
both SHAs whenever a window contains no commits, so ALL empty windows collapse to a single
cache key. Harmless today (every empty window renders the identical "No activity" body
except the window label — which means the label can actually be WRONG on a cache hit:
a cached "Today (...)" journal served for an empty `--window week`). Fix: include the
window label (or raw spec) in MakeKey's hash input. Update `internal/cache` tests
(key-difference cases) and add one asserting two different empty windows produce
different keys.

## 3. anthropic-api backend hardcodes a dated model id

`internal/synth` maps the alias `haiku` to a pinned dated id. The Anthropic API accepts
alias ids directly, so the mapping ages for no benefit. Fix: delete the alias map and pass
the user's `--model` string through verbatim; document in README/guide that both aliases
and full ids work. Adjust the synth tests that assert the mapping.

## Notes

- All three are user-visible enough for one small changelog entry (`--type fix`).
- Ship as a patch release when the batch lands (release account note: pushes for this
  repo run under the account that owns it — verify `gh api user` before releasing).

## Effort

Under an hour total including tests.
