# current state

Date: 2026-10-22
Commit: 0f350a57cb0f7054783843753ce40fdfe8667773

## Objective
Ship compass v3 as a narrow tool that owns only the why, the why-not and where-we-are
(positioning.md "Scope"; decisions 0015 scope cut, 0016 current-state is last known intent).
The work is stacked on PR #2 (branch `claude/go-rewrite-74db42`); it is not
merged or released. The Codex eval of the core operations
(experiments/2026-10-22-codex-core-operations-eval.md) passed 4 of 5 bars; its fixes are
decisions 0017 and 0018; 0019 adds the search instruction.

## Next steps
1. **Watch decision reads in real use:** the block now says to search the records with `rg` before
   changing behaviour (decision 0019; it took buried decisions from 3/9 to 9/9 in the
   retrieval-at-scale eval, branch `claude/retrieval-experiment-0cc627`, not merged). Build a
   keyword `compass find` over live records only if a run reverses a decision because the target
   was buried in `rg` output or a superseded record misled it.
2. **Emilio:** review PR #2 and its CI (verify-plugin, smoke on ubuntu and macos,
   `check --require-fresh`); after merge tag `v0.3.0` and test `go install ...@latest`.
3. **Emilio:** `uv tool uninstall compass-cli` (it still shadows the Go binary; doctor's warning
   stays until then); remove the v2 hooks, skills and session tools from `~/.claude` (this repo no
   longer holds their source) and the compass-workflow section of `~/.claude/CLAUDE.md`.
