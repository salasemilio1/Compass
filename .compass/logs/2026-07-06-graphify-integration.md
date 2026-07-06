# graphify integration (deterministic-only)

Date: 2026-07-06
Commits: feb8cc1..f01c3c9

## Why
Pilot graphify (github.com/safishamsi/graphify) as an optional structural-navigation layer for compass —
reduce agent token cost when navigating a codebase — while keeping compass's "easy, orthogonal" design
goal: no forced complexity, string together small deterministic tools.

## What changed
- `install.sh` — optional `install_graphify()` step (uv/pipx/pip fallback), mirrors the existing
  `install_context_mode()` pattern.
- `README.md` / `SPEC.md` — documented the three-layer model (compass = curated narrative,
  context-mode = output sandboxing, graphify = structural navigation) and compass's read-only
  relationship to `graphify-out/`.
- `skills/warm-up/SKILL.md` — prefer `graphify query` over grepping when `graphify-out/graph.json` exists.
- `hooks/compass-session-start.sh` — mentions graphify when present; fires `graphify update .`
  (deterministic, zero LLM) in the background on every session start once a graph exists in the repo.
- `.compass/` scaffolded for this repo itself via `init-compass` (`committed` mode) — first time this
  repo dogfoods its own protocol.
- `AGENTS.md`, `CLAUDE.md` created, pointing at compass.
- `.compass/decisions/0001-graphify-deterministic-only.md` — see below.

## Assumptions changed
Originally planned to run graphify's semantic extraction (LLM-based concept/relationship extraction for
`.md`/`.txt`) against a local Ollama model (`qwen2.5-coder:14b`) to avoid cloud cost. Verified it works
end-to-end (accurate `graphify query` results, $0 cost) but also hit a real data-quality issue (a dropped
required field on one extracted edge) and a real refresh-latency question (LLM extraction isn't
instant even at $0). Decided against it — see ADR 0001. Landed on deterministic-only
(`graphify update <path>`: tree-sitter + markdown regex), which needs zero config and is safe to
auto-refresh unconditionally.

## Learnings
See `learnings/graphify-cli-quirks.md` and `learnings/verify-flagged-installs-directly.md`.

## Breaking changes
None — all additive, all gated on `.compass/` and `graphify-out/` existing.
