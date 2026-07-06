# overview

Date: 2026-07-04

compass is an in-repo agent-memory protocol for Claude Code. A repo "has a compass" when it contains a
`.compass/` directory following `SPEC.md`. The goal: orient any agent cheaply from the repo instead of
from conversation history, so sessions can be disposable.

## Repo layout
- `SPEC.md` — the protocol contract (categories, INDEX.md format, lifecycle, sharing model)
- `template/` — the `.compass/` scaffold `init-compass` copies into a target repo
- `skills/{warm-up,wrap-up,init-compass}/SKILL.md` — the three slash commands
- `hooks/compass-session-start.sh` — SessionStart hook, nudges agents to warm up when `.compass/` exists
- `install.sh` — symlinks skills/hook into `~/.claude/`, idempotent, optionally installs context-mode
  and graphify

## Three-layer model (on the `graphify-integration` branch, unmerged as of 2026-07-06)
This repo is piloting [graphify](https://github.com/safishamsi/graphify) as an optional third layer
alongside compass and context-mode:
- **graphify** — auto-derived code structure ("what calls what, where does X live"), regenerated from
  the code itself. compass only uses graphify's deterministic mode (`graphify update <path>`): tree-sitter
  for code, regex for markdown structure/links. No LLM, no API key, no network call. See `decisions/0001-graphify-deterministic-only.md` for why.
- **context-mode** — sandboxes large tool output (diffs, test runs) so it doesn't consume the session
  window.
- **compass** (this repo) — curated, human-reviewed narrative memory: why decisions were made, what
  changed and why, project status.

compass never reads or writes `graphify-out/`; it's optional reference material an agent may consult for
mechanical lookups, same trust tier as reading source directly (see `SPEC.md`).
