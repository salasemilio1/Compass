# verify compass CLI, build presentation diagrams, audit the dedup embedder and machine footprint

Date: 2026-07-10
Commits: 76b4d9c..HEAD (AGENTS.md change uncommitted at time of writing)

## Why
User needed to (1) confirm `compass init` actually works end-to-end before relying on it, (2) build
diagrams explaining compass's workflow for a presentation, and (3) sanity-check the system's own
claims of technical elegance and machine footprint under skeptical questioning.

## What changed
- Verified `compass` CLI end-to-end via `ctx_execute`: 12/12 unit tests pass, scaffold/append/
  interactive-prompt/`--path` behaviors all work correctly, globally-installed binary (`uv tool`)
  is byte-identical to repo source. No bugs found.
- `AGENTS.md` — documented `session_status.py` (the deterministic token-usage tool) so any agent
  working in this repo reaches for it instead of guessing/deflecting when asked about token usage.
- Built presentation diagrams (not committed to the repo — rendered inline in conversation) showing
  the session workflow (start/work/wrap-up unchanged for the user, hooks and skills running
  underneath) and the learnings-dedup cascade with its offline embedding stage.
- User pressure-tested the dedup cascade's embedding stage (whole-document embedding, no chunking)
  and asked for evidence it has ever fired. Investigation found it hasn't — see
  `learnings/dedup-stage3-never-verified-with-real-embedder.md`.
- Audited the machine footprint of a compass install (sizes and paths under `~/.claude/`,
  `~/.local/`, and per-repo `.compass/`) and found four stray `*.backup-20260704100310` symlinks in
  `~/.claude/skills/` and `~/.claude/hooks/`, left over from before decisions/0006 switched
  install.sh from symlinking to copying. One of them (`compass-session-start.sh.backup-...`) still
  points at a different, older compass checkout. Not referenced by `settings.json`, so not a live
  risk, but dead weight the user wants cleaned up next session.

## Assumptions changed
None — no design decisions were made this session, only verification and audit.

## Learnings
- `learnings/dedup-stage3-never-verified-with-real-embedder.md`

## Breaking changes
None.
