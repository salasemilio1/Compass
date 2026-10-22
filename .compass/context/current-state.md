# current state

Date: 2026-10-22
Commit: a61202e4caec3b67749d25f89b8eb80b217a7e34

## Objective
Ship compass v3 as a narrow tool that owns only the why, the why-not and where-we-are
(positioning.md "Scope"; decisions 0015 scope cut, 0016 current-state is last known intent).
The work is on branch `claude/compass-scope-refine-4e1fc9`, on top of `claude/go-rewrite-74db42`;
nothing is pushed, released or merged. The Codex eval of the core operations
(experiments/2026-10-22-codex-core-operations-eval.md) passed 4 of 5 bars; its fixes are
decisions 0017 and 0018.

## Next steps
1. **Retrieval at scale:** the `retrieval-experiment` session's agent runs decide whether a search
   command is needed or one AGENTS.md line suffices. Offline (experiments/2026-10-22-retrieval-q2-keywords.md),
   no ranker, keyword line or embedding blend cleared its bar on a blind graded set: don't build one
   for ranking quality. Decision 0018 cut folding; agents still don't record obvious-looking choices.
2. **Emilio:** review both branches, push, open a PR, watch CI's first run (verify-plugin, smoke,
   `check --require-fresh`); after merge tag `v0.3.0` and test `go install ...@latest`.
3. **Emilio:** `uv tool uninstall compass-cli` (it still shadows the Go binary; doctor's warning
   stays until then); remove the v2 hooks, skills and session tools from `~/.claude` (this repo no
   longer holds their source) and the compass-workflow section of `~/.claude/CLAUDE.md`.
