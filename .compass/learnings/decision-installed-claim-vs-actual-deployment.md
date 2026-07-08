# a decision/log saying "installed" doesn't mean it's deployed

`decisions/0005-deterministic-tool-routing-guard.md` documented the `PreToolUse` tool-routing guard as
built and installed. In reality, `install.sh --dev` was never re-run after the hook file was added, so
`~/.claude/hooks/tool-routing-guard.py` didn't exist as a symlink and `~/.claude/settings.json` had no
`PreToolUse` entry — the guard had never fired, in any session, since it was written.

**How to apply:** when resuming work on a hook/skill/MCP integration that a prior decision or log
describes as "installed," verify the actual global state directly (`ls` the symlink target, `grep`/parse
`settings.json`, `claude mcp list`) before trusting the prose — prior sessions can build and document a
change without completing the deploy step, and nothing else catches that gap. See also
[[pretooluse-hooks-fail-closed]] for a related PreToolUse hook gotcha found in the same work.
