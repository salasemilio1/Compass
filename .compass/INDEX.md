# compass index — compass

> An in-repo agent-memory protocol for Claude Code: stores curated project context (`.compass/`) in the
> repo so sessions start informed instead of cold, plus skills to load/save it (`warm-up`, `wrap-up`,
> `init-compass`) and a `SessionStart` hook that nudges agents to use it.

**Stack:** bash (install script, hook), Markdown (skills, spec, template)   ·   **Mode:** committed   ·   **Updated:** 2026-07-06

## Start here
- New to the project? Read: [context/overview.md](context/overview.md), [SPEC.md](../SPEC.md)
- Common tasks: adding/changing a skill → edit `skills/<name>/SKILL.md`, then update this INDEX line
- A `graphify-out/graph.json` structural graph is available for this repo (via `graphify query`) —
  see [context/overview.md](context/overview.md) for the three-layer model.

## context
<!-- one line per entry; this is a retrieval hint, not a summary -->
- [overview.md](context/overview.md) — what compass is, the three-layer model (compass/context-mode/graphify), repo layout

## architecture
- _none yet_

## decisions
- [0001-graphify-deterministic-only.md](decisions/0001-graphify-deterministic-only.md) — graphify wiring uses only its deterministic mode (no LLM/backend), for zero-cost always-fresh navigation

## experiments
<!-- newest first; one line: model + headline result -->
- _none yet_

## logs
<!-- newest first -->
- _none yet_

## learnings
- _none yet_

---

## Folder guide

| Folder | Holds | Naming |
|---|---|---|
| `context/` | Domain facts, goals, constraints, glossary. `current-state.md` tracks where you left off. | `slug.md` |
| `architecture/` | Components, data flow, module boundaries. Start with `overview.md`. | `slug.md` |
| `decisions/` | ADRs: what was decided, alternatives, consequences. | `NNNN-slug.md` |
| `experiments/` | One file per ML run: config, results, observations, failures. | `YYYY-MM-DD-slug.md` |
| `logs/` | One file per completed task: what changed and why. | `YYYY-MM-DD-slug.md` |
| `learnings/` | Durable gotchas extracted from logs and experiments. | `slug.md` |
| `scratch/` | Ephemeral working notes — always gitignored, never authoritative. | any |
