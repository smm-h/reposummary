---
title: README.md
---
# reposummary

:-: var key="project.description"

Git commits **are** the journal: everything that can be extracted deterministically from git is extracted for free, and the LLM prose layer is entirely optional.

## Install

- Go: `go install github.com/smm-h/reposummary@latest`
- npm: `npm i -g reposummary`
- PyPI: `pip install reposummary`

(The npm and PyPI packages download the prebuilt binary.)

## Usage

`reposummary summarize <repo> --window <window> --synthesis <backend>`

`<repo>` defaults to the current directory. `--window` and `--synthesis` are required (no default).

```
# Deterministic journal for today, no LLM
reposummary summarize . --window today --synthesis none

# A fixed one-month span from a date
reposummary summarize . --window 2026-06-11+1month --synthesis none

# The whole history, narrated via the local claude CLI
reposummary summarize . --window all --synthesis claude-cli --model haiku
```

## Commands

:-: table-commands path="."

## Windows

- `today` — commits since local midnight today
- `yesterday` — the previous calendar day
- `week` / `7d` — the last 7 days
- `month` / `30d` — the last 30 days
- `all` / `start` / `start..now` — the full history
- `<YYYY-MM-DD>+<N><unit>` — a fixed span from a date (e.g. `2026-06-11+1month`)
- `<refA>..<refB>` — an explicit git revision range (e.g. `v0.1.0..HEAD`)

An unrecognized window is a hard error, never a silent fallthrough. See the usage guide (docs/guide.md) for details.

## Synthesis backends

`--synthesis` selects the prose backend explicitly — there is **no silent fallback**. A chosen backend that fails is a hard error, not a quiet downgrade.

- `none` — no LLM; the journal is the deterministic extraction only.
- `claude-cli` — shells out to `claude -p --model <model>` (no API key needed).
- `anthropic-api` — calls the Anthropic Messages API; requires `ANTHROPIC_API_KEY` in the environment (no implicit default).

`--model` selects the model id (default `haiku`).

## Caching

A journal for a fixed `(firstSHA, lastSHA, synthesis, model, version)` tuple is deterministic, so identical windows reuse cached output on disk. The cache key is a sha256 of those inputs; entries are plain Markdown files under `$XDG_CACHE_HOME/reposummary` (or `~/.cache/reposummary`). Repeated summaries therefore cost O(new commits), not O(window size). Disable with `--no-cache`, or point elsewhere with `--cache-dir`.
