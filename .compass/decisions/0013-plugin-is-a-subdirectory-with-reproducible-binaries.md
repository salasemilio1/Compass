---
claim: the plugin is ./plugin (launcher in bin/, binaries in libexec/), built reproducibly so CI byte-checks the committed binaries
date: 2026-10-21
premises:
  - goal:disposable-sessions
refs:
  - decisions/0010-go-binary-as-claude-code-plugin.md
  - Makefile
---
# the plugin is ./plugin (launcher in bin/, binaries in libexec/), built reproducibly so CI byte-checks the committed binaries

## Context
0010 planned the repo as its own marketplace with `"source": "./"` and binaries in `bin/`. Building
it showed three problems: the plugin cache would copy the whole repo (Python tools, tests, docs);
`bin/` is appended to the Bash tool's PATH, so per-platform binaries there would all be on PATH;
and "prebuilt binaries are committed" needs a way to know they match the source.

## Decision
- The marketplace (`.claude-plugin/marketplace.json`) points at `./plugin`, which holds only the
  manifest, `hooks/hooks.json`, `bin/compass` (a POSIX sh launcher that resolves symlinks and picks
  the OS/CPU build) and `libexec/compass-{darwin,linux}-{arm64,amd64}`.
- Builds are reproducible (`-trimpath -buildvcs=false`, `CGO_ENABLED=0`, go.mod's toolchain via
  `GOTOOLCHAIN`). CI rebuilds and byte-compares the committed binaries (`make verify-plugin`);
  verified identical from a macOS and a Linux build host.
- Releases attach per-platform tarballs and checksums; `go install` works with no dependencies.
- The launcher exits 0 for Claude Code hook calls when no binary fits the host, so a missing
  platform never blocks the agent; CLI calls fail.
- `compass init` writes a one-line pointer into CLAUDE.md, never `@AGENTS.md`: ebay-flipper keeps
  AGENTS.md as a full copy of CLAUDE.md, and the import would load it twice per session.
- The git pre-commit hook uses the Go compass on PATH, else the plugin's cached copy (skipping
  orphaned versions), else skips with a warning; a Python compass-cli first on PATH can't fail
  commits.

## Consequences
- About 11.6 MB of binaries per release in git history (0010's option A).
- Uninstall verified: Claude Code's plugin registries return byte-identical; the cache copy is
  marked `.orphaned_at` and purged by Claude Code later.
