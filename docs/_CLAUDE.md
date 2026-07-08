---
title: CLAUDE.md
---
# reposummary

:-: var key="project.description"

The usage guide is `docs/guide.md`. The CLI surface is generated from
`.strictcli/schema.json` (regenerate with `reposummary --dump-schema`).

## Project structure

Internal packages (each paragraph is the package's Go doc comment, extracted from source):

:-: prose-desc path="internal/window"

:-: prose-desc path="internal/gitdata"

:-: prose-desc path="internal/digest"

:-: prose-desc path="internal/render"

:-: prose-desc path="internal/synth"

:-: prose-desc path="internal/cache"

## Design rules (non-negotiable)

- No silent degradation / no fallback: the `--synthesis` backend is an explicit choice; a
  chosen backend that fails is a hard error, never a silent fallback.
- No implicit defaults for provider/billing values: `--window` and `--synthesis` are
  required (no default).
- Deterministic extraction is free (git plumbing); LLM tokens are spent only on the final
  prose synthesis.

## Build / test / release

- `go build ./... && go vet ./... && go test ./...`
- Regenerate the CLI schema after CLI changes: `reposummary --dump-schema` (commit
  `.strictcli/schema.json`).
- Release via rlsbl (`rlsbl release run`); docs regenerate via selfdoc at release.
