<!-- compass-spec-version: 1 -->
# Agent instructions

This repo uses the compass protocol for agent memory. Before substantial work:
1. Read `.compass/INDEX.md` — the retrieval map.
2. Read `.compass/context/current-state.md` — where the work stands and what's next (if present).

At task end, run the wrap-up workflow (git-diff-first, human-reviewed) to capture knowledge back into
`.compass/`. Never write `.compass/` silently.

[graphify](https://github.com/safishamsi/graphify) is an optional, opt-in structural navigation layer
(not installed by default — see README.md). If `graphify-out/graph.json` happens to exist, prefer
`graphify query "<question>"` over grepping for "what calls X" / "where does Y live" style questions.
compass stays the source of curated intent; graphify is read-only reference material, never written or
modified by compass workflows.

If asked how many tokens the session has used, run
`python3 ~/.claude/tools/session-context/session_status.py` (installed globally, works from any repo)
instead of estimating — it reads the session's transcript for exact API-billed usage. A `Stop` hook
(`session-context-nudge.py`) already nudges toward `/wrap-up` once usage crosses 75%, once per session.
