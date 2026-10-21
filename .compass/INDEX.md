# compass index — compass

> An in-repo agent-memory protocol for Claude Code: stores curated project context (`.compass/`) in the
> repo so sessions start informed instead of cold, plus skills to load/save it (`warm-up`, `wrap-up`,
> `init-compass`) and a `SessionStart` hook that nudges agents to use it.

**Stack:** bash (install script, hook, checks), Python (learnings dedup, session-context, compass CLI), Markdown (skills, spec, template)   ·   **Mode:** committed, steady-state (main)   ·   **Updated:** 2026-10-21 (v3 direction set: Go plugin, positioning/delivery, dedup by index judge)

## Start here
- New to the project? Read: [context/overview.md](context/overview.md), [SPEC.md](../SPEC.md)
- Resuming? Read: [context/current-state.md](context/current-state.md)
- Common tasks: adding/changing a skill → edit `skills/<name>/SKILL.md`, then update this INDEX line;
  changing `compass init` → edit `cli/compass_cli/`, tests in `cli/tests/`
- A `graphify-out/graph.json` structural graph is available for this repo (via `graphify query`) —
  see [context/overview.md](context/overview.md) for the three-layer model.

## context
<!-- one line per entry; this is a retrieval hint, not a summary -->
- [current-state.md](context/current-state.md) — compass init CLI verified working end-to-end; stray backup symlinks in ~/.claude flagged for deletion next session; dedup stage 3 flagged as never verified against a real embedder
- [overview.md](context/overview.md) — what compass is, the three-layer model (compass/context-mode/graphify), repo layout
- [positioning.md](context/positioning.md) — v3 core: compass is the colleague layer (non-derivable intent); current-state front door + path-triggered decision delivery gated by a trigger eval; no human gate; no manual wrap-up
- [v3-direction.md](context/v3-direction.md) — open: one-CLI portability (guards never shipped), capped generated index, retrieval options (frontmatter, SQLite FTS5, embeddings, MCP), chosen by a retrieval eval
- [2026-10-15-ebay-flipper-retro.md](context/2026-10-15-ebay-flipper-retro.md) — first long-run field data: 90% of cost in main sessions, 78% above 300k context; 103 KB INDEX; logs/learnings mostly never re-read; model-by-task grades

## environment
<!-- one file per machine/service, keyed by host; overwritten in place, not accumulated -->
- _none yet_ — compass's own repo doesn't SSH anywhere; this category exists for repos scaffolded by compass that do

## architecture
- _none yet_

## decisions
- [SUPERSEDED by 0008] [0001-graphify-deterministic-only.md](decisions/0001-graphify-deterministic-only.md) — graphify wiring uses only its deterministic mode (no LLM/backend), for zero-cost always-fresh navigation
- [SUPERSEDED by 0006] [0002-close-audited-gaps-with-deterministic-scripts.md](decisions/0002-close-audited-gaps-with-deterministic-scripts.md) — closed 5 audited bloat/staleness gaps with compass-check.sh + a 3-stage learnings-dedup filter; rejected Caveman/Headroom/semantic-tool-selection as the wrong layer
- [SUPERSEDED by 0011] [0003-learnings-dedup-always-runs-stage3.md](decisions/0003-learnings-dedup-always-runs-stage3.md) — dropped the confidently-novel lexical short-circuit after real-content calibration found true paraphrases it silently missed
- [0004-transcript-based-tool-efficiency-measurement.md](decisions/0004-transcript-based-tool-efficiency-measurement.md) — measure context-mode/graphify's real token savings by attributing per-turn token deltas to tool calls in the transcript, deterministic, no synthetic benchmark
- [AMENDED by 0008] [0005-deterministic-tool-routing-guard.md](decisions/0005-deterministic-tool-routing-guard.md) — PreToolUse hook that denies heavy/broad Bash calls toward ctx_execute/graphify by command shape, with a per-session retry safety valve so an unavailable alternate tool can't deadlock a session
- [SUPERSEDED by 0010] [0006-copy-not-symlink-global-install.md](decisions/0006-copy-not-symlink-global-install.md) — install.sh copies (not symlinks) skills/hooks so an ephemeral dotfiles clone can be deleted after install; drops the now-pointless $DOTFILES canonical-source guard and --dev flag (supersedes 0002)
- [AMENDED by 0010] [0007-compass-init-as-cli.md](decisions/0007-compass-init-as-cli.md) — init-compass's mechanical scaffolding moves to a real Click CLI (cli/), fixing the template-path bug by construction; warm-up/wrap-up stay markdown skills (judgment-heavy, weak fit for a CLI)
- [0008-graphify-becomes-optional-add-on.md](decisions/0008-graphify-becomes-optional-add-on.md) — graphify moves from default-installed+enforced to opt-in only (`COMPASS_WITH_GRAPHIFY=1`); a week under decisions/0005's forced-routing guard produced under 5 real uses despite the tool working well when live-tested, so demand — not enforcement — was the gap; drops the guard's graphify rule, keeps deterministic-only mode if opted in
- [0009-environment-category-and-repo-layout-check.md](decisions/0009-environment-category-and-repo-layout-check.md) — adds an `environment/` category (one file per machine/service, keyed by host, overwritten in place) for SSH access/ports/version quirks that had no home; adds a warn-only `compass-check.sh` check that a repo's top-level dirs are mentioned in `context/overview.md`; spec version 1 → 2
- [0010-go-binary-as-claude-code-plugin.md](decisions/0010-go-binary-as-claude-code-plugin.md) — compass becomes one Go binary (CLI + hook handler + MCP server over one core) shipped as a Claude Code plugin with committed mac/linux binaries; spike-verified install, hooks, MCP, subagent delivery (supersedes 0006, amends 0007)
- [0011-dedup-by-index-judge-not-embeddings.md](decisions/0011-dedup-by-index-judge-not-embeddings.md) — drop MiniLM embeddings (0.30 recall, 230 MB); dedup = cheap LLM judge over INDEX one-line hooks (P 0.96 / R 0.99, $0.017); merges append, never delete; no bulk consolidation (supersedes 0003)

## experiments
<!-- newest first; one line: model + headline result -->
- _none yet_

## logs
<!-- newest first -->
- [2026-07-15-environment-category-and-layout-check.md](logs/2026-07-15-environment-category-and-layout-check.md) — added `environment/` category + `compass-check.sh` repo-layout drift check (decisions/0009); fixed this repo's own stale `context/overview.md`; spec v2
- [2026-07-10-graphify-optional-and-cursor-ctx-mode.md](logs/2026-07-10-graphify-optional-and-cursor-ctx-mode.md) — graphify moved to opt-in (decisions/0008) after a week of enforced routing produced under 5 uses; learnings-dedup stage 3 verified against the real fastembed model; context-mode confirmed working in Cursor; fixed install.sh's broken context-mode package reference
- [2026-07-10-presentation-prep-and-dedup-audit.md](logs/2026-07-10-presentation-prep-and-dedup-audit.md) — verified compass CLI end-to-end (no bugs), built presentation diagrams, found dedup stage 3 has never been verified against a real embedder, flagged stray backup symlinks in ~/.claude for cleanup
- [2026-07-08-compass-init-cli.md](logs/2026-07-08-compass-init-cli.md) — extracted init-compass's mechanical scaffolding into a real `compass` CLI (cli/), bundled template via importlib.resources, wired into install.sh/CI/skill (decisions/0007)
- [2026-07-08-tool-routing-guard-background-fix.md](logs/2026-07-08-tool-routing-guard-background-fix.md) — fixed tool-routing-guard.py denying backgrounded dev-server commands (npm run dev) by exempting run_in_background calls from all routing rules
- [2026-07-08-hardening-and-install-rework.md](logs/2026-07-08-hardening-and-install-rework.md) — closed the fault-tolerance audit backlog (loud-not-silent fixes across compass-check.sh/register_hook.py/session_status.py/tool-routing-guard.py), merged graphify-integration to main, reworked install.sh to copy instead of symlink (decisions/0006); caught and reverted a false "typo fix" to graphify's real graphifyy package name
- [2026-07-08-confirm-guard-and-warmup-health-check.md](logs/2026-07-08-confirm-guard-and-warmup-health-check.md) — found the tool-routing guard was never actually deployed (decision said "installed", global state didn't match) and fixed via install.sh --dev; fixed context-mode MCP (broken system Node from a Homebrew llhttp mismatch); added a ctx/graphify health check to warm-up
- [2026-07-08-ci-and-deterministic-routing.md](logs/2026-07-08-ci-and-deterministic-routing.md) — added GitHub Actions CI (audit gap #1); built a PreToolUse hook that deterministically routes heavy Bash calls to ctx_execute/graphify instead of relying on agent judgment (decisions/0005)
- [2026-07-07-fault-tolerance-audit.md](logs/2026-07-07-fault-tolerance-audit.md) — stepped-back audit of compass as long-lived software; core deterministic-filter pattern holds, 9 gaps found at the edges (Claude Code format coupling, install topology, cache versioning); backlog now in current-state.md
- [2026-07-07-precommit-backstop-and-tool-efficiency.md](logs/2026-07-07-precommit-backstop-and-tool-efficiency.md) — installed the `.git/hooks/pre-commit` backstop, built `tool_efficiency.py` (decisions/0004), surfaced a Bash-truncation-vs-ctx measurement bias
- [2026-07-07-context-usage-nudge.md](logs/2026-07-07-context-usage-nudge.md) — built a token-based Stop hook (session_status.py + session-context-nudge.py) replacing ADE's turn-count nudge, swapped jq for python3 in install.sh
- [2026-07-06-dedup-calibration-and-wrapup-ux.md](logs/2026-07-06-dedup-calibration-and-wrapup-ux.md) — calibrated learnings-dedup against real content, closed the residual paraphrase gap (decisions/0003), streamlined wrap-up's review flow
- [2026-07-06-close-audited-gaps.md](logs/2026-07-06-close-audited-gaps.md) — deterministic checker/fixer + install guard + staleness hook + learnings-dedup filter, closing 5 audited gaps
- [2026-07-06-graphify-integration.md](logs/2026-07-06-graphify-integration.md) — wired graphify in as a deterministic-only navigation layer, auto-refreshed via the SessionStart hook

## learnings
- [context-mode-package-name-was-wrong.md](learnings/context-mode-package-name-was-wrong.md) — install.sh/README documented a nonexistent npm package (`@anthropic-ai/mcp-server-context-mode`, 404); the real package is just `context-mode`
- [dedup-stage3-never-verified-with-real-embedder.md](learnings/dedup-stage3-never-verified-with-real-embedder.md) — verified 2026-07-10: real fastembed/MiniLM run correctly scores a loose paraphrase at 0.59 (OK) and a tight paraphrase at 0.85 (POSSIBLE_DUPLICATE); model caches under $TMPDIR, not ~/.cache
- [pretooluse-guard-must-exempt-backgrounded-commands.md](learnings/pretooluse-guard-must-exempt-backgrounded-commands.md) — a routing hook denying heavy commands by shape must also exempt run_in_background calls; narrowing the regex doesn't generalize, execution-mode is the right axis
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
- [check-existing-learnings-before-treating-as-typo.md](learnings/check-existing-learnings-before-treating-as-typo.md) — a doubled letter that "looks like a typo" (graphifyy) was actually the real package name; two learnings already said so — grep learnings/ before "fixing" it

---

## Folder guide

| Folder | Holds | Naming |
|---|---|---|
| `context/` | Domain facts, goals, constraints, glossary. `current-state.md` tracks where you left off. | `slug.md` |
| `environment/` | Machine/service specifics: SSH access, non-default ports/paths, pinned versions, quirks. One file per host, overwritten in place. | `host-slug.md` |
| `architecture/` | Components, data flow, module boundaries. Start with `overview.md`. | `slug.md` |
| `decisions/` | ADRs: what was decided, alternatives, consequences. | `NNNN-slug.md` |
| `experiments/` | One file per ML run: config, results, observations, failures. | `YYYY-MM-DD-slug.md` |
| `logs/` | One file per completed task: what changed and why. | `YYYY-MM-DD-slug.md` |
| `learnings/` | Durable gotchas extracted from logs and experiments. | `slug.md` |
| `scratch/` | Ephemeral working notes — always gitignored, never authoritative. | any |
