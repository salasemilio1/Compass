# compass index — compass

> An in-repo agent-memory protocol for Claude Code: stores curated project context (`.compass/`) in the
> repo so sessions start informed instead of cold, plus skills to load/save it (`warm-up`, `wrap-up`,
> `init-compass`) and a `SessionStart` hook that nudges agents to use it.

**Stack:** bash (install script, hook, checks), Python (learnings dedup), Markdown (skills, spec, template)   ·   **Mode:** committed   ·   **Updated:** 2026-07-06

## Start here
- New to the project? Read: [context/overview.md](context/overview.md), [SPEC.md](../SPEC.md)
- Resuming? Read: [context/current-state.md](context/current-state.md)
- Common tasks: adding/changing a skill → edit `skills/<name>/SKILL.md`, then update this INDEX line
- A `graphify-out/graph.json` structural graph is available for this repo (via `graphify query`) —
  see [context/overview.md](context/overview.md) for the three-layer model.

## context
<!-- one line per entry; this is a retrieval hint, not a summary -->
- [current-state.md](context/current-state.md) — piloting graphify-integration branch, unmerged, next steps to decide merge/revert
- [overview.md](context/overview.md) — what compass is, the three-layer model (compass/context-mode/graphify), repo layout

## architecture
- _none yet_

## decisions
- [0001-graphify-deterministic-only.md](decisions/0001-graphify-deterministic-only.md) — graphify wiring uses only its deterministic mode (no LLM/backend), for zero-cost always-fresh navigation
- [0002-close-audited-gaps-with-deterministic-scripts.md](decisions/0002-close-audited-gaps-with-deterministic-scripts.md) — closed 5 audited bloat/staleness gaps with compass-check.sh + a 3-stage learnings-dedup filter; rejected Caveman/Headroom/semantic-tool-selection as the wrong layer

## experiments
<!-- newest first; one line: model + headline result -->
- _none yet_

## logs
<!-- newest first -->
- [2026-07-06-close-audited-gaps.md](logs/2026-07-06-close-audited-gaps.md) — deterministic checker/fixer + install guard + staleness hook + learnings-dedup filter, closing 5 audited gaps
- [2026-07-06-graphify-integration.md](logs/2026-07-06-graphify-integration.md) — wired graphify in as a deterministic-only navigation layer, auto-refreshed via the SessionStart hook

## learnings
- [bash-pipefail-empty-substitution.md](learnings/bash-pipefail-empty-substitution.md) — guard `var=$(grep|awk)` with `|| true` under `set -euo pipefail`, or an expected "not found" kills the script
- [test-helper-naming-must-avoid-test-glob.md](learnings/test-helper-naming-must-avoid-test-glob.md) — a shared test helper named `test_*.sh` gets silently "run" (and passes) by a glob-based test runner
- [plan-approval-does-not-blanket-authorize-actions.md](learnings/plan-approval-does-not-blanket-authorize-actions.md) — approving a plan doesn't pre-authorize every action in it; the permission classifier can still deny individually
- [graphify-cli-quirks.md](learnings/graphify-cli-quirks.md) — Ollama backend needs `graphifyy[ollama]` extras; `--backend` flag syntax differs by subcommand (not currently used — deterministic-only)
- [verify-flagged-installs-directly.md](learnings/verify-flagged-installs-directly.md) — when an install is flagged as a possible typosquat, verify via the registry's own API, not just the project's docs

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
