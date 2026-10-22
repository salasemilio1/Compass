# current state

Date: 2026-10-22
Commit: 1d12c47a51783e7bcb931a396eedf0bcbfda5470

## Objective
Ship compass v3 as a narrow tool that owns only the why, the why-not and where-we-are
(positioning.md "Scope"; decisions 0015 scope cut, 0016 current-state is last known intent).
The work is on branch `claude/compass-scope-refine-4e1fc9`, on top of `claude/go-rewrite-74db42`;
nothing is pushed, released or merged. The Codex eval of the core operations
(experiments/2026-10-22-codex-core-operations-eval.md) passed 4 of 5 bars; its fixes are
decisions 0017 and 0018.

## Next steps
1. **Retrieval at scale:** the `retrieval-experiment` session measures INDEX, all-claims,
   keyword find and SQLite FTS5/BM25 on an ebay-flipper corpus; build only what it shows is needed.
   Decision 0018 (one choice per record) cut folding but agents still don't record choices that
   look obvious to them (experiments/2026-10-22-one-choice-rerun.md); leave that to code comments.
2. **Emilio:** review both branches, push, open a PR, watch CI's first run (verify-plugin, smoke,
   `check --require-fresh`); after merge tag `v0.3.0` and test `go install ...@latest`.
3. **Emilio:** `uv tool uninstall compass-cli` (it still shadows the Go binary; doctor's warning
   stays until then); remove the v2 hooks, skills and session tools from `~/.claude` (this repo no
   longer holds their source) and the compass-workflow section of `~/.claude/CLAUDE.md`.
