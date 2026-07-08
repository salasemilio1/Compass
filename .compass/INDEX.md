# compass index — compass

> An in-repo agent-memory protocol for Claude Code: stores curated project context (`.compass/`) in the
> repo so sessions start informed instead of cold, plus skills to load/save it (`warm-up`, `wrap-up`,
> `init-compass`) and a `SessionStart` hook that nudges agents to use it.

**Stack:** bash (install script, hook, checks), Python (learnings dedup, session-context), Markdown (skills, spec, template)   ·   **Mode:** committed   ·   **Updated:** 2026-07-08 (confirmed guard deployment, warm-up health check)

## Start here
- New to the project? Read: [context/overview.md](context/overview.md), [SPEC.md](../SPEC.md)
- Resuming? Read: [context/current-state.md](context/current-state.md)
- Common tasks: adding/changing a skill → edit `skills/<name>/SKILL.md`, then update this INDEX line
- A `graphify-out/graph.json` structural graph is available for this repo (via `graphify query`) —
  see [context/overview.md](context/overview.md) for the three-layer model.

## context
<!-- one line per entry; this is a retrieval hint, not a summary -->
- [current-state.md](context/current-state.md) — piloting graphify-integration branch, unmerged; pre-commit backstop installed and tool_efficiency.py built (decisions/0004), which surfaced a Bash-truncation-vs-ctx measurement bias; next: design a deterministic (non-agent-judgment) routing rule
- [overview.md](context/overview.md) — what compass is, the three-layer model (compass/context-mode/graphify), repo layout

## architecture
- _none yet_

## decisions
- [0001-graphify-deterministic-only.md](decisions/0001-graphify-deterministic-only.md) — graphify wiring uses only its deterministic mode (no LLM/backend), for zero-cost always-fresh navigation
- [0002-close-audited-gaps-with-deterministic-scripts.md](decisions/0002-close-audited-gaps-with-deterministic-scripts.md) — closed 5 audited bloat/staleness gaps with compass-check.sh + a 3-stage learnings-dedup filter; rejected Caveman/Headroom/semantic-tool-selection as the wrong layer
- [0003-learnings-dedup-always-runs-stage3.md](decisions/0003-learnings-dedup-always-runs-stage3.md) — dropped the confidently-novel lexical short-circuit after real-content calibration found true paraphrases it silently missed
- [0004-transcript-based-tool-efficiency-measurement.md](decisions/0004-transcript-based-tool-efficiency-measurement.md) — measure context-mode/graphify's real token savings by attributing per-turn token deltas to tool calls in the transcript, deterministic, no synthetic benchmark
- [0005-deterministic-tool-routing-guard.md](decisions/0005-deterministic-tool-routing-guard.md) — PreToolUse hook that denies heavy/broad Bash calls toward ctx_execute/graphify by command shape, with a per-session retry safety valve so an unavailable alternate tool can't deadlock a session

## experiments
<!-- newest first; one line: model + headline result -->
- _none yet_

## logs
<!-- newest first -->
- [2026-07-08-confirm-guard-and-warmup-health-check.md](logs/2026-07-08-confirm-guard-and-warmup-health-check.md) — found the tool-routing guard was never actually deployed (decision said "installed", global state didn't match) and fixed via install.sh --dev; fixed context-mode MCP (broken system Node from a Homebrew llhttp mismatch); added a ctx/graphify health check to warm-up
- [2026-07-08-ci-and-deterministic-routing.md](logs/2026-07-08-ci-and-deterministic-routing.md) — added GitHub Actions CI (audit gap #1); built a PreToolUse hook that deterministically routes heavy Bash calls to ctx_execute/graphify instead of relying on agent judgment (decisions/0005)
- [2026-07-07-fault-tolerance-audit.md](logs/2026-07-07-fault-tolerance-audit.md) — stepped-back audit of compass as long-lived software; core deterministic-filter pattern holds, 9 gaps found at the edges (Claude Code format coupling, install topology, cache versioning); backlog now in current-state.md
- [2026-07-07-precommit-backstop-and-tool-efficiency.md](logs/2026-07-07-precommit-backstop-and-tool-efficiency.md) — installed the `.git/hooks/pre-commit` backstop, built `tool_efficiency.py` (decisions/0004), surfaced a Bash-truncation-vs-ctx measurement bias
- [2026-07-07-context-usage-nudge.md](logs/2026-07-07-context-usage-nudge.md) — built a token-based Stop hook (session_status.py + session-context-nudge.py) replacing ADE's turn-count nudge, swapped jq for python3 in install.sh
- [2026-07-06-dedup-calibration-and-wrapup-ux.md](logs/2026-07-06-dedup-calibration-and-wrapup-ux.md) — calibrated learnings-dedup against real content, closed the residual paraphrase gap (decisions/0003), streamlined wrap-up's review flow
- [2026-07-06-close-audited-gaps.md](logs/2026-07-06-close-audited-gaps.md) — deterministic checker/fixer + install guard + staleness hook + learnings-dedup filter, closing 5 audited gaps
- [2026-07-06-graphify-integration.md](logs/2026-07-06-graphify-integration.md) — wired graphify in as a deterministic-only navigation layer, auto-refreshed via the SessionStart hook

## learnings
- [compass-agent-portability-boundary.md](learnings/compass-agent-portability-boundary.md) — `.compass/`/`AGENTS.md`/`SPEC.md` are agent-agnostic and travel with the repo; skills/hooks/session-context tools are Claude-Code-specific and need manual translation
- [hooks-require-session-restart.md](learnings/hooks-require-session-restart.md) — Claude Code loads hook config once at session start; changes need a new session, can't hot-swap
- [claude-project-dir-slug-replaces-non-alphanumeric.md](learnings/claude-project-dir-slug-replaces-non-alphanumeric.md) — the `~/.claude/projects/<slug>` slug replaces every non-alphanumeric char with `-`, not just `/`
- [test-assertions-must-encode-intent-not-observed-output.md](learnings/test-assertions-must-encode-intent-not-observed-output.md) — decide the correct expected value from spec/intent before running the code; never shape an assertion (or a mock feeding one) to match observed output
- [bash-pipefail-empty-substitution.md](learnings/bash-pipefail-empty-substitution.md) — guard `var=$(grep|awk)` with `|| true` under `set -euo pipefail`, or an expected "not found" kills the script
- [test-helper-naming-must-avoid-test-glob.md](learnings/test-helper-naming-must-avoid-test-glob.md) — a shared test helper named `test_*.sh` gets silently "run" (and passes) by a glob-based test runner
- [plan-approval-does-not-blanket-authorize-actions.md](learnings/plan-approval-does-not-blanket-authorize-actions.md) — approving a plan doesn't pre-authorize every action in it; the permission classifier can still deny individually
- [graphify-cli-quirks.md](learnings/graphify-cli-quirks.md) — Ollama backend needs `graphifyy[ollama]` extras; `--backend` flag syntax differs by subcommand (not currently used — deterministic-only)
- [verify-flagged-installs-directly.md](learnings/verify-flagged-installs-directly.md) — when an install is flagged as a possible typosquat, verify via the registry's own API, not just the project's docs
- [pretooluse-hooks-fail-closed.md](learnings/pretooluse-hooks-fail-closed.md) — PreToolUse hooks block the tool call if the hook script crashes/times out, unlike SessionStart/Stop; a denying hook must be fast and defensive on every path
- [decision-installed-claim-vs-actual-deployment.md](learnings/decision-installed-claim-vs-actual-deployment.md) — a decision/log documenting a hook as "installed" doesn't mean install.sh was actually re-run; verify the symlink/settings.json directly

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
