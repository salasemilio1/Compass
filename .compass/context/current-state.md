# current state

Date: 2026-07-10
Commit: df7f6eec5a05ad736774c883ba95612798efe02f

## Where things stand
Compass is `main`, steady-state, maintenance-mode. This session closed out the prior session's full
backlog: graphify moved from default-installed+enforced to an opt-in add-on (decisions/0008, after a
week of forced-routing enforcement still produced under 5 real uses), learnings-dedup stage 3 was
verified end-to-end against the real `fastembed` model, the four stray backup symlinks under
`~/.claude/` were deleted, and context-mode was confirmed working as a cross-agent MCP server in
Cursor (in addition to Claude Code).

## Known issues / things to remember
- graphify is now opt-in only (`COMPASS_WITH_GRAPHIFY=1 ./install.sh`, or install it directly). The
  `PreToolUse` routing guard no longer has a graphify-specific rule — broad searches always route to
  `ctx_execute` now.
- `install.sh`'s context-mode install command was fixed this session — it previously referenced a
  nonexistent npm package (`@anthropic-ai/mcp-server-context-mode`, 404). Now installs the real
  `context-mode` package and registers it via `claude mcp add context-mode -- context-mode`.
- `~/.cursor/mcp.json` (global, on this machine) now registers `context-mode` for Cursor — confirmed
  showing up as an MCP server there.
- `install.sh` hard-requires `python3` (hook registration) and benefits from `uv`/`pipx`/`pip`
  (compass CLI) and `npm` (context-mode) — all degrade to a warning, never a hard failure. `node` is
  no longer required for context-mode (fixed to use `npm install -g` instead of `npx`).
- `graphify-out/` and `.compass/learnings/.embedding-cache.json` are gitignored/regenerable.
- `.git/hooks/pre-commit` backstop (runs `compass-check.sh` check-only) is installed in this repo.

## Next steps
1. Commit this session's changes (graphify opt-in, dedup verification, context-mode fix) — currently
   staged, not yet committed.
2. No other backlog items remain from prior sessions — this session closed all of them.
3. Compass stays maintenance-mode: fix real breakage as it's found, don't proactively expand scope.
   If graphify's usage picture changes on a future, larger repo, that's new evidence — don't
   re-add it to compass's default path without it.
