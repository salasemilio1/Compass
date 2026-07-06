<!-- compass-spec-version: 1 -->
# Agent instructions

This repo uses the compass protocol for agent memory. Before substantial work:
1. Read `.compass/INDEX.md` — the retrieval map.
2. Read `.compass/context/current-state.md` — where the work stands and what's next (if present).

At task end, run the wrap-up workflow (git-diff-first, human-reviewed) to capture knowledge back into
`.compass/`. Never write `.compass/` silently.

This repo is also piloting [graphify](https://github.com/safishamsi/graphify) for structural navigation.
If `graphify-out/graph.json` exists, prefer `graphify query "<question>"` over grepping for "what calls
X" / "where does Y live" style questions. compass stays the source of curated intent; graphify is
read-only reference material, never written or modified by compass workflows.
