---
title: Usage guide
description: Window forms, synthesis backends, and caching for reposummary.
nav_group: Guide
nav_order: 1
---
# Usage guide

`reposummary summarize <repo> --window <window> --synthesis <backend>` extracts a git
repository's activity over a time window and renders a Markdown journal, optionally narrated
by an LLM. `<repo>` defaults to the current directory; `--window` and `--synthesis` are
required and have no default.

## Windows

`--window` accepts the following forms. An unrecognized window is a hard error, never a
silent fallthrough.

| Form | Meaning | Example |
| --- | --- | --- |
| `today` | commits since local midnight today | `--window today` |
| `yesterday` | the previous calendar day | `--window yesterday` |
| `week` / `7d` | the last 7 days | `--window 7d` |
| `month` / `30d` | the last 30 days | `--window month` |
| `all` / `start` / `start..now` | the full history | `--window all` |
| `<YYYY-MM-DD>+<N><unit>` | a fixed span from a date; unit is `d` / `w` / `m` / `y` | `--window 2026-06-11+1month` |
| `<refA>..<refB>` | an explicit git revision range | `--window v0.1.0..HEAD` |

The tip of the window is `--branch` (default `HEAD`).

## Synthesis backends

`--synthesis` selects the prose backend explicitly. There is **no silent fallback**: the
chosen backend must work, or the command errors. reposummary never quietly downgrades one
backend to another.

| Backend | Behavior |
| --- | --- |
| `none` | no LLM; the journal is the deterministic extraction only |
| `claude-cli` | shells out to the `claude` CLI (`claude -p --model <model>`); no API key needed. If `claude` is missing, not logged in, or fails, that is a hard error |
| `anthropic-api` | calls the Anthropic Messages API; requires `ANTHROPIC_API_KEY` in the environment (no implicit default). A missing key is a hard error |

`--model` selects the model id (default `haiku`, which maps to a current Anthropic Haiku
model; full model ids are passed through unchanged). The model is only consulted when the
backend is `claude-cli` or `anthropic-api`.

## Caching and incremental efficiency

A journal for a fixed `(firstSHA, lastSHA, synthesis, model, version)` tuple is
deterministic, so identical windows reuse cached output. The cache key is a sha256 over
those five inputs:

- `firstSHA` and `lastSHA` — the commit range actually covered by the window
- `synthesis` — the selected backend
- `model` — the model id
- `version` — the reposummary version (so a new release never serves stale prose)

Entries are plain Markdown files under the cache directory
(`$XDG_CACHE_HOME/reposummary`, or `~/.cache/reposummary`). Because the key is the resolved
commit range rather than the window text, two different window expressions that resolve to
the same range share a cache entry, and repeated summaries cost O(new commits), not
O(window size).

- `--no-cache` disables the cache entirely (always recompute, never read or write).
- `--cache-dir <path>` overrides the cache directory.

## Examples

```
# Deterministic journal for today, printed to stdout
reposummary summarize . --window today --synthesis none

# Last week, narrated via the Anthropic API
ANTHROPIC_API_KEY=... reposummary summarize ~/code/myproj --window week --synthesis anthropic-api

# A fixed one-month span from a date, no LLM
reposummary summarize . --window 2026-06-11+1month --synthesis none

# A release range, narrated via the local claude CLI, written to a file
reposummary summarize . --window v0.1.0..HEAD --synthesis claude-cli --output CHANGES.md

# The whole history, ignoring any cached output
reposummary summarize . --window all --synthesis none --no-cache
```
