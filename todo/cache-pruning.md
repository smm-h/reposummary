# Cache pruning: the journal cache grows without bound

## Context

`internal/cache` stores each rendered journal as a content-addressed file under the
platform cache dir (`~/.cache/reposummary/<sha>.md`), keyed on
(firstSHA, lastSHA, synthesis, model, version). Entries are written on every uncached run
and never removed. Deliberately deferred at review time — growth is tiny markdown files —
but unbounded is unbounded.

## Problem

Every distinct (window-tip, backend, version) combination adds a file forever. Heavy use
across many repos/windows (or scripted daily use) accretes stale entries: version bumps
alone orphan every prior entry (version is part of the key). Nothing ever cleans up.

## Proposed solutions

1. **Age-based prune on write (recommended).** After a successful cache write, opportunistically delete entries with mtime older than N days (e.g. 90). Zero new commands, self-maintaining, bounded cost per run.
   - Pros: invisible, no new surface; mtime is refreshed on cache hits if we touch on read (do that too, so hot entries survive).
   - Cons: a prune pass on every write does a dir scan (cheap at realistic sizes).
2. **Explicit `cache prune` subcommand** (`--older-than`, `--all` with confirmation).
   - Pros: user-controlled, no background behavior.
   - Cons: nobody runs manual maintenance commands; the problem persists in practice.
3. Both: opportunistic prune + the command for manual full clears.

## Affected files

- `internal/cache/cache.go` (+ touch-on-read in Get, prune helper), tests for prune ages
  and hot-entry survival; README caching section.
- If a subcommand: the strictcli registration + schema regen (`--dump-schema`) + selfdoc.

## Effort

Small — an hour or two with tests.
