# reposummary

`reposummary` generates a Markdown "journal" of a git repository's activity over
a time window. The core idea: git commits **are** the narrative. Everything that
can be extracted deterministically from git is extracted for free; LLM tokens are
spent only on the final prose layer, if you ask for one.

## What it does

Given a repository and a time window, `reposummary`:

1. Reads commit metadata, file churn, and tags directly from git.
2. Classifies commits (breaking / feature / fix / other), sums churn, tallies
   authors, per-directory activity, issue references, and per-day counts.
3. Renders a Markdown journal: an at-a-glance summary, grouped change sections,
   active areas, and a timeline.
4. Optionally prepends an LLM-written prose summary of what happened.

## Usage

```
reposummary summarize <repo> --window <w> --synthesis <s>
```

`<repo>` defaults to the current directory. `--window` and `--synthesis` are
required.

### Window forms

| Form | Meaning |
| --- | --- |
| `today` | commits since local midnight today |
| `yesterday` | commits during the previous calendar day |
| `week` or `7d` | the last 7 days |
| `month` or `30d` | the last 30 days |
| `all` (or `start`, `start..now`) | the full history |
| `<YYYY-MM-DD>+<N><unit>` | a fixed span from a date; unit is `d`/`w`/`m`/`y` (e.g. `2026-06-11+1month`) |
| `<refA>..<refB>` | an explicit git revision range (e.g. `v0.1.0..HEAD`) |

An unrecognized window is a hard error, never a silent fallthrough.

### Synthesis backends

`--synthesis` selects the prose backend explicitly. There is **no silent
fallback**: the chosen backend must work, or the command errors.

| Backend | Behavior |
| --- | --- |
| `none` | no LLM; the journal is the deterministic extraction only |
| `claude-cli` | shells out to the `claude` CLI (`claude -p ... --model ...`); if it is not logged in or fails, that is a hard error |
| `anthropic-api` | calls the Anthropic Messages API; requires `ANTHROPIC_API_KEY` in the environment (no implicit default) |

`--model` selects the model id (default `haiku`, which maps to a current Anthropic
Haiku model; full model ids are passed through unchanged).

### Other flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `--branch` | `HEAD` | git ref to treat as the tip |
| `--output` | (stdout) | write the journal to a file instead of stdout |
| `--cache` / `--no-cache` | on | use the on-disk journal cache |
| `--cache-dir` | (XDG) | override the cache directory |

## Caching and incremental efficiency

A journal for a fixed `(firstSHA, lastSHA, synthesis, model, version)` tuple is
deterministic, so identical windows reuse cached output. The cache key is a
sha256 of those inputs; entries are plain Markdown files under the cache
directory (`$XDG_CACHE_HOME/reposummary` or `~/.cache/reposummary`). This makes
repeated summaries cost O(new commits), not O(window size). Disable with
`--no-cache`.

## Build and install

```
go build -o reposummary .
go install .
```

Zero external dependencies except the CLI framework (`strictcli`); everything
else is the Go standard library.

## Examples

```
# Everything, no LLM prose
reposummary summarize . --window all --synthesis none

# Last week, narrated via the Anthropic API
ANTHROPIC_API_KEY=... reposummary summarize ~/code/myproj --window week --synthesis anthropic-api

# A release range, narrated via the local claude CLI, written to a file
reposummary summarize . --window v0.1.0..HEAD --synthesis claude-cli --output CHANGES.md
```
