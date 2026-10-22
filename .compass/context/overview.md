---
claim: "what compass is (v3: one Go binary plus the .compass/ format) and the repo layout"
---
# overview

Date: 2026-10-22

compass keeps a project's working memory in the repo, in `.compass/`, so agent sessions are
disposable: a fresh agent runs `compass resume` (or gets the brief from the SessionStart hook) and
continues from current-state instead of from conversation history. v3 is one static Go binary; the
contract is the `.compass/` format in `SPEC.md` Part A (Part B is the v2 history). Positioning and
scope: [positioning.md](positioning.md); the binary and plugin: decisions 0013 and 0014.

## Repo layout
- `cmd/compass/` — the CLI entry point: argument parsing and exit codes, nothing else
- `internal/compass/` — the core, the only code that touches `.compass/`: `check`, `index`,
  `record`, `resume`, `init`, `doctor`, current-state freshness (`state.go`)
- `internal/hook/` — Claude Code hook adapters (SessionStart brief or stub, Stop, PreCompact,
  git pre-commit) over the core
- `plugin/` — what the Claude Code marketplace installs: manifest, `hooks/hooks.json`, the sh
  launcher in `bin/`, committed reproducible binaries in `libexec/` (decision 0013)
- `.claude-plugin/marketplace.json` — the repo as its own marketplace, pointing at `plugin/`
- `Makefile` — `test`, `build` (to `bin/`), `plugin`, `verify-plugin`, `dist`, `smoke`
- `.github/workflows/` — CI (Go tests, verify-plugin, smoke, `compass check`; Python tests for the
  v2 leftovers) and the tag-triggered release
- `tests/` — `smoke.sh` runs the shipped binaries through the launcher with no Go on PATH; the
  Python tests cover the v2 leftovers below
- `SPEC.md`, `README.md`, `AGENTS.md` — the format contract, install/use, the agent contract

v2 leftovers, kept but installed by nothing (their tests still run in CI):
- `skills/` — `init-compass`, `warm-up`, `wrap-up` markdown skills (they still name the removed
  v2 checker script)
- `hooks/` — v2's SessionStart nudge, context-usage nudge and tool-routing guard
- `tools/` — `learning-dedup/`, `session-context/`, `usage-retro/`

## Neighbours
compass is the curated "why and where are we" layer. context-mode keeps large tool output out of
the session window; graphify (optional add-on, deterministic mode only, decision 0008) maps code
structure. compass never reads or writes `graphify-out/`.
