# current state

Date: 2026-10-22
Commit: 175a96b9b904c2ea8c45403af68c358ebe0edece

## Objective
Ship the compass v3 core (positioning.md, "The core"; decision 0014, which supersedes 0010) as one
Go binary. The core and the fixes from its eval are on branch `claude/go-rewrite-74db42`; it is not
pushed, released or merged.

## Just done
- 3d5f7a6: the brief's "already contains current-state" line is on its own line and says INDEX.md
  is for lookup; this repo's AGENTS.md no longer requires Claude-only tooling.
- d072cac: an over-long current-state is cut by section (Just done, Known issues, Files in play
  first), never the next steps; `check` warns `next-steps-missing`.
- bfbc883: SessionStart sends the full brief only on compact/resume, uncommitted work, stale or
  unknown freshness, or a shadowing Python compass; otherwise a stub of at most 500 characters.
- 175a96b: no `anchor-unresolved` warning for a rebased anchor whose freshness is still known;
  `--fix` never restamps an anchor that merely isn't in a shallow clone.
- v2 retirement: `install.sh`, `scripts/`, `cli/` and their tests/CI steps deleted; `skills/`,
  `hooks/`, `tools/` kept with their tests. README/SPEC fixed, decision 0014 recorded,
  positioning.md Measures and overview.md rewritten for the Go layout.
- This checkout's shared pre-commit hook runs the Go compass (v2 hook saved as `pre-commit.v2.bak`).

## Files in play
- `internal/compass/` — the core (the only code that touches .compass/)
- `internal/hook/hook.go` — Claude Code hook adapters, SessionStart gating
- `plugin/` — what the marketplace installs (rebuild with `make plugin` before committing Go changes)
- `.github/workflows/ci.yml` — go job plus the Python tests for the v2 leftovers (never run)
- `.compass/decisions/0014-go-binary-as-built.md` — what shipped, issues 6 and 12

## Next steps
1. **Run the SessionStart eval once `claude -p` has a plan login:** three unspecified "continue"
   tasks, stub vs full brief, on an ebay-flipper copy; record it under `experiments/`.
2. **Emilio:** review, push the branch, open a PR and watch CI's first run (verify-plugin on
   Linux, smoke on ubuntu and macos, `check --require-fresh`).
3. **Emilio:** after merge, tag `v0.3.0`; test `go install ...@latest` and the release archives.
4. **Emilio:** `uv tool uninstall compass-cli` (it shadows the Go binary); remove the v2 hook
   entries from `~/.claude/settings.json` (`compass-session-start.sh`, `session-context-nudge.py`,
   `tool-routing-guard.py`) before deleting their files; remove the v2 skills from `~/.claude/skills`
   and the compass-workflow section of `~/.claude/CLAUDE.md`. Until then both SessionStart hooks fire.
5. **Emilio:** delete the `claude/dedup-experiment-436ce8` branch; delivery stays gated.

## Known issues
- CI and release workflows have never run.
- The globally installed v2 `wrap-up` (runs `scripts/compass-check.sh --fix`) and `init-compass`
  (`install.sh`, `cli/`) are still in use and fail in this repo after merge, until step 4.
- Deferred: a v2-to-v3 migration tool, detecting dropped next steps, an AGENTS.md neutrality lint.
