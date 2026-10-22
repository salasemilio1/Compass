# current state

Date: 2026-10-22
Commit: 2a9f310313f3adc1ac884d08b9195a1569092e41

## Objective
Ship compass v3 as a narrow tool that owns only the why, the why-not and where-we-are
(positioning.md "Scope"; decisions 0015 scope cut, 0016 current-state is last known intent).
The work is on branch `claude/compass-scope-refine-4e1fc9`, on top of `claude/go-rewrite-74db42`;
nothing is pushed, released or merged. The Codex eval of the core operations
(experiments/2026-10-22-codex-core-operations-eval.md) passed 4 of 5 bars.

## Next steps
1. **Decide on the eval's two fixes:** narrow the AGENTS.md block's "record a decision" rule to
   choices with a rejected alternative someone might reverse (runs recorded trivia), and have
   `check` flag a record whose body is the empty skeleton. Both are additions, so they need a yes.
2. **Emilio:** review both branches, push, open a PR, watch CI's first run (verify-plugin, smoke,
   `check --require-fresh`); after merge tag `v0.3.0` and test `go install ...@latest`.
3. **Emilio:** `uv tool uninstall compass-cli` (it still shadows the Go binary; doctor's warning
   stays until then); remove the v2 hooks, skills and session tools from `~/.claude` (this repo no
   longer holds their source) and the compass-workflow section of `~/.claude/CLAUDE.md`.
