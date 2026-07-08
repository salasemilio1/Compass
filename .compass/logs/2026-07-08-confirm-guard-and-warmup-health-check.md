# confirm tool-routing guard deployment; add warm-up system health check

Commits: e905c23..HEAD

## Why
Follow-up to decisions/0005: the tool-routing guard and context-usage nudge were both flagged in
`current-state.md` as "built but unconfirmed live." Checking on that surfaced two separate gaps: the
guard wasn't actually deployed, and `context-mode` MCP was down for an unrelated system reason. Fixing
both, plus a request to make system health visible on every warm-up instead of needing an ad hoc check.

## What changed
- **Diagnosed `context-mode` MCP failure**: a Homebrew `llhttp` upgrade (9.3.0 → 9.4.2) left
  `/opt/homebrew/opt/llhttp` symlinked to the new version while the Node binary was still linked against
  9.3.0, so `node` (and therefore `context-mode`, a Node MCP server) couldn't launch at all. Fixed by
  relinking `/opt/homebrew/opt/llhttp` to the 9.3.0 keg (both versions were still installed). This is a
  local machine fix, not a compass code change — `context-mode doctor` and `claude mcp list` confirm
  it's connected now.
- **Diagnosed the tool-routing guard was never deployed**: `hooks/tool-routing-guard.py` existed in the
  repo and decisions/0005 described it as installed, but `install.sh --dev` was never re-run after it
  was added, so it had no symlink in `~/.claude/hooks/` and no `PreToolUse` entry in
  `~/.claude/settings.json`. Ran `install.sh --dev` (with explicit user approval, since it edits global
  config) to symlink the hook and register it. Confirmed the `PreToolUse` block now exists in
  `settings.json`. Per `learnings/hooks-require-session-restart.md`, it still won't fire until a fresh
  session — that confirmation is still outstanding.
- **Added a system health check to `warm-up`**: `skills/warm-up/SKILL.md` now runs two cheap
  deterministic checks (`claude mcp list` for ctx, `command -v graphify` for the CLI) as step 2, and
  reports them as a one-line status (e.g. `System: ctx ✓ connected · graphify ✓ installed`) leading the
  final warm-up summary. Steps renumbered accordingly.

## Assumptions changed
- "Built and documented as installed" was previously treated as equivalent to "deployed." It isn't —
  see `learnings/decision-installed-claim-vs-actual-deployment.md`.

## Learnings
- [decision-installed-claim-vs-actual-deployment.md](../learnings/decision-installed-claim-vs-actual-deployment.md)

## Next steps
- Start a fresh session to confirm the `PreToolUse` guard actually fires and denies as designed — this
  is now the first real opportunity to check, since the guard is properly deployed for the first time.
- After that, re-run `tool_efficiency.py --all` per decisions/0004/0005's follow-up.
- Continue the audit backlog: gap #3 (defensive transcript parsing) is next.
