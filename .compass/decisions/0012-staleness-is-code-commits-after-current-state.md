---
claim: current-state is stale when code commits exist after the later of its in-history anchor and its last commit; stale warns locally and fails only CI (--require-fresh)
date: 2026-10-21
premises:
  - goal:disposable-sessions
refs:
  - context/positioning.md
  - internal/compass/state.go
---
# current-state is stale when code commits exist after the later of its in-history anchor and its last commit; stale warns locally and fails only CI (--require-fresh)

## Context
positioning.md says the Stop hook blocks "if the branch has commits newer than current-state's
`Commit:` anchor" and CI fails a PR that "leaves current-state's anchor stale". Taken literally,
current-state is always stale: committing it creates a commit newer than its own anchor.

## Decision
- current-state is **stale** when commits that change anything outside `.compass/` exist after
  the later of its anchor and the last commit that changed current-state.md. Committing code and
  current-state together (anchored at the parent) is fresh; a commit touching only `.compass/`
  never makes it stale.
- The anchor counts only if it is in HEAD's history. A squash-merged or rebased branch leaves its
  anchor as an unreachable object; freshness then falls back to the last current-state commit and
  check warns (`anchor-unresolved`).
- Stale is a **warning** in `compass check` and pre-commit, and an **error** only with
  `--require-fresh`, which CI passes on pull requests. Blocking every code commit locally until
  current-state is rewritten would make it rewritten after every commit, which is churn, not state.
- `check --fix` stamps HEAD only into an anchor that is missing, malformed or names no commit (a
  live Haiku session typed an invented hash); never into a real stale one.
- Stop blocks once per turn (`stop_hook_active`). PreCompact blocks only a manual `/compact`, once
  per session: blocking automatic compaction would strand a full context, and PreCompact cannot
  inject text into the summary. After compaction, the SessionStart brief (source `compact`) asks
  for the rewrite instead.

## Consequences
- The live Stop test: a session that only said "done" was blocked once and rewrote and committed
  current-state itself.
- CI needs full history (`fetch-depth: 0`) to resolve anchors.
