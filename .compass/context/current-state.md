# current state

Date: 2026-10-22
Commit: b7fd072dbca57d6d0412a7e33ba75093ed4a0e77

## Objective
Ship compass v3 as a narrow tool that owns only the why, the why-not and where-we-are
(positioning.md "Scope"; decisions 0015 scope cut, 0016 current-state is last known intent).
The work is on branch `claude/compass-scope-refine-4e1fc9`, on top of `claude/go-rewrite-74db42`;
nothing is pushed, released or merged. The Codex eval of the core operations
(experiments/2026-10-22-codex-core-operations-eval.md) passed 4 of 5 bars; its two fixes
are decision 0017.

## Next steps
1. **Watch decision recording in real use:** decision 0017 cut trivia records to 0 of 6 in the
   targeted re-run (experiments/2026-10-22-decision-recording-rerun.md), but agents fold a new
   choice into the nearest existing decision and supersede it; decide whether the block should
   say "one choice per record" once that shows up outside the fixture.
2. **Emilio:** review both branches, push, open a PR, watch CI's first run (verify-plugin, smoke,
   `check --require-fresh`); after merge tag `v0.3.0` and test `go install ...@latest`.
3. **Emilio:** `uv tool uninstall compass-cli` (it still shadows the Go binary; doctor's warning
   stays until then); remove the v2 hooks, skills and session tools from `~/.claude` (this repo no
   longer holds their source) and the compass-workflow section of `~/.claude/CLAUDE.md`.
