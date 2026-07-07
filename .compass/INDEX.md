# compass index — compass

> An in-repo agent-memory protocol for Claude Code: stores curated project context (`.compass/`) in the
> repo so sessions start informed instead of cold, plus skills to load/save it (`warm-up`, `wrap-up`,
> `init-compass`) and a `SessionStart` hook that nudges agents to use it.

**Stack:** bash (install script, hook, checks), Python (learnings dedup, session-context), Markdown (skills, spec, template)   ·   **Mode:** committed   ·   **Updated:** 2026-07-07 (context-usage nudge)

## Start here
- New to the project? Read: [context/overview.md](context/overview.md), [SPEC.md](../SPEC.md)
- Resuming? Read: [context/current-state.md](context/current-state.md)
- Common tasks: adding/changing a skill → edit `skills/<name>/SKILL.md`, then update this INDEX line
- A `graphify-out/graph.json` structural graph is available for this repo (via `graphify query`) —
  see [context/overview.md](context/overview.md) for the three-layer model.

## context
<!-- one line per entry; this is a retrieval hint, not a summary -->
- [current-state.md](context/current-state.md) — piloting graphify-integration branch, unmerged; token-based context-usage nudge now installed; awaiting a fresh session + a few real sessions of data before deciding merge/revert and building decisions/0004's efficiency report
- [overview.md](context/overview.md) — what compass is, the three-layer model (compass/context-mode/graphify), repo layout

## architecture
- _none yet_

## decisions
- [0001-graphify-deterministic-only.md](decisions/0001-graphify-deterministic-only.md) — graphify wiring uses only its deterministic mode (no LLM/backend), for zero-cost always-fresh navigation
- [0002-close-audited-gaps-with-deterministic-scripts.md](decisions/0002-close-audited-gaps-with-deterministic-scripts.md) — closed 5 audited bloat/staleness gaps with compass-check.sh + a 3-stage learnings-dedup filter; rejected Caveman/Headroom/semantic-tool-selection as the wrong layer
- [0003-learnings-dedup-always-runs-stage3.md](decisions/0003-learnings-dedup-always-runs-stage3.md) — dropped the confidently-novel lexical short-circuit after real-content calibration found true paraphrases it silently missed
- [0004-transcript-based-tool-efficiency-measurement.md](decisions/0004-transcript-based-tool-efficiency-measurement.md) — measure context-mode/graphify's real token savings by attributing per-turn token deltas to tool calls in the transcript, deterministic, no synthetic benchmark

## experiments
<!-- newest first; one line: model + headline result -->
- _none yet_

## logs
<!-- newest first -->
- [2026-07-07-context-usage-nudge.md](logs/2026-07-07-context-usage-nudge.md) — built a token-based Stop hook (session_status.py + session-context-nudge.py) replacing ADE's turn-count nudge, swapped jq for python3 in install.sh
- [2026-07-06-dedup-calibration-and-wrapup-ux.md](logs/2026-07-06-dedup-calibration-and-wrapup-ux.md) — calibrated learnings-dedup against real content, closed the residual paraphrase gap (decisions/0003), streamlined wrap-up's review flow
- [2026-07-06-close-audited-gaps.md](logs/2026-07-06-close-audited-gaps.md) — deterministic checker/fixer + install guard + staleness hook + learnings-dedup filter, closing 5 audited gaps
- [2026-07-06-graphify-integration.md](logs/2026-07-06-graphify-integration.md) — wired graphify in as a deterministic-only navigation layer, auto-refreshed via the SessionStart hook

## learnings
- [hooks-require-session-restart.md](learnings/hooks-require-session-restart.md) — Claude Code loads hook config once at session start; changes need a new session, can't hot-swap
- [claude-project-dir-slug-replaces-non-alphanumeric.md](learnings/claude-project-dir-slug-replaces-non-alphanumeric.md) — the `~/.claude/projects/<slug>` slug replaces every non-alphanumeric char with `-`, not just `/`
- [test-assertions-must-encode-intent-not-observed-output.md](learnings/test-assertions-must-encode-intent-not-observed-output.md) — decide the correct expected value from spec/intent before running the code; never shape an assertion (or a mock feeding one) to match observed output
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
