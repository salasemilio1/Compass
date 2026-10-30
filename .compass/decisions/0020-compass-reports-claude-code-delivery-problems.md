---
claim: compass reports Claude Code delivery problems (doctor; resume and pre-commit warn inside Claude Code) and never edits ~/.claude to fix them
date: 2026-10-29
premises:
  - goal:disposable-sessions
refs:
  - docs/specs/2026-10-29-delivery-health.md
  - context/2026-10-29-ebay-flipper-v3-days-1-4.md
---
# compass reports Claude Code delivery problems (doctor; resume and pre-commit warn inside Claude Code) and never edits ~/.claude to fix them

## Context
On 2026-10-28 a folder move left `~/.claude/settings.json` a dangling symlink on the maintainer's
machine. Claude Code then loaded no hooks and no plugins; compass was off for 5 of 8 ebay-flipper
sessions and nothing said so (context/2026-10-29-ebay-flipper-v3-days-1-4.md). The SessionStart
hook can't report its own absence.

## Decision
A read-only delivery check (`internal/compass/delivery.go`) runs in `compass doctor` (problems with
fixes, exit 1) and, when `CLAUDECODE=1`, in `compass resume` and `compass hook pre-commit` (stderr
warnings; outputs and exit codes unchanged). The AGENTS.md block tells agents to run doctor when no
`compass:` message arrived at session start. `tests/e2e.sh` installs the plugin into a clean config
dir with the real `claude` CLI and runs in CI (CLI pinned).

## Alternatives
- `compass doctor --fix` editing `~/.claude/settings.json`: rejected by Emilio (2026-10-29). Another
  setup tool owns that file on his machine, and the README promises nothing is written outside
  repositories.
- Checking from the SessionStart hook: rejected, a hook that runs can't see that hooks don't, and it
  would cost every session.
- Warning outside Claude Code too: rejected, Codex and terminal users who never installed the
  plugin would get noise forever; there it's a doctor note.
