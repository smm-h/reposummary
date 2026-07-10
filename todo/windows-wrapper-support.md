# npm/PyPI wrappers: add Windows support

## Context

goreleaser builds Windows artifacts (`reposummary_<v>_windows_{amd64,arm64}.zip`) and they
ship on every GitHub release, but both distribution wrappers only map linux/darwin:

- `npm/install.js` — PLATFORM_MAP/ARCH_MAP cover linux/darwin; win32 hits the "unsupported
  platform" fallback (manual download / go install message). The map was copied verbatim
  from the reference template project, which also lacks Windows.
- `pypi/reposummary/__init__.py` — same platform mapping and tar.gz-only extraction.

So `npm i -g reposummary` / `pip install reposummary` on Windows installs a package whose
binary bootstrap refuses to run, despite a perfectly good zip existing on the release.

## Problem

Windows users get a working-looking install with a dead binary path and a manual-steps
error message. Either support the platform or don't build/publish the artifacts that imply
we do.

## Proposed solutions

1. **Add win32 to both wrappers (recommended).** Map `win32`→`windows`, handle the `.zip`
   archive format (npm: unzip without native deps — Node has no stdlib unzip; smallest
   path is `zlib`-based minimal reader or a tiny dependency, weigh against the
   zero-dependency stance; python: `zipfile` stdlib, trivial), name the binary
   `reposummary.exe`, skip chmod on Windows. Test the mapping logic in the existing
   wrapper test files.
   - Pros: completes the distribution story the release artifacts already promise.
   - Cons: npm-side unzip needs care (dependency vs hand-rolled); CI has no Windows
     smoke test today.
2. **Drop Windows artifacts from goreleaser instead.** If Windows is genuinely out of
   audience, stop building the zips so the wrappers' fallback message is honest.
   - Pros: zero code; removes the false promise.
   - Cons: closes the platform for `go install`-averse users.

## Affected files

- `npm/install.js`, `npm/install.test.js`, `pypi/reposummary/__init__.py`, possibly
  `pyproject`/`package.json` metadata (os/cpu fields), README install section,
  `.goreleaser.yml` (only if option 2).

## Effort

Small-medium (the npm unzip decision is the only real design point). Changelog: user-facing
feature ("Windows support for npm/PyPI installs") if option 1.
