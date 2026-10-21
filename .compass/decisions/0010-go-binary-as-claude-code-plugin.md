# 0010 — compass becomes one Go binary, shipped as a Claude Code plugin

Date: 2026-10-21
Supersedes: 0006
Amends: 0007 (the CLI moves from Python/Click to Go; its scope grows from `init` to every deterministic step)

## Context
v2's deterministic tools never ran outside this repo. `wrap-up` called `scripts/compass-check.sh` and
`tools/learning-dedup/` by repo-relative path, and `install.sh` didn't ship them, so in ebay-flipper no
cap, supersession mark or dedup ever applied ([retro](../context/2026-10-15-ebay-flipper-retro.md)).
Install was also spread across six places: `~/.claude/skills/`, `~/.claude/hooks/`,
`~/.claude/tools/`, `settings.json` edits, a uv tool, and uv's Python and caches. There was no
uninstall.

Requirements: one-command install by pasting the repo link to an agent, everything in one directory,
uninstall as easy as install, minimal footprint, Mac and Linux.

## Decision
1. **One Go binary** (`compass`) that is, at the same time:
   - the CLI: `init`, `check [--fix]`, `index`, `doctor`, `stats`;
   - the hook handler: `compass hook <event>`;
   - the MCP server: `compass mcp`.

   All three are thin adapters over one core (`internal/`), the only code that touches `.compass/`.
2. **Distributed as a Claude Code plugin.** The repo is its own marketplace (`"source": "./"`). The
   plugin holds the binaries in `bin/` plus a small sh launcher that picks the OS/CPU build,
   `hooks/hooks.json`, `.mcp.json` and the markdown skills. Install:
   `claude plugin marketplace add <owner>/compass` then `claude plugin install compass@compass`.
3. **Prebuilt binaries are committed** for darwin/linux × arm64/amd64, built by a CI release job.
   Windows is out of scope.
4. **Interface model:**
   - The contract is the `.compass/` format plus `SPEC.md`.
   - The CLI is canonical and every command has `--json`.
   - The MCP surface is capped at about 4 tools (`search`, `get`, `context(path)`, `record`), since
     each tool's description costs tokens in every session.
   - Hooks translate agent events into neutral ones.
   - The adapters are hand-written over shared typed request/result structs, with a parity test. No
     generated registry until there are more than about 10 operations.
5. **Portability:** the baseline is the format, CLI, MCP and `AGENTS.md`. Hooks and skills are a
   Claude Code accelerator ([positioning](../context/positioning.md)).

## Evidence (spike, 2026-10-21, hello-world plugin in the scratchpad)
- Cross-compiled 4 static binaries from one Mac in seconds, about 7.5 MB each, including the official
  MCP Go SDK (`modelcontextprotocol/go-sdk` v1.8.0).
- Non-interactive marketplace add plus install from the Bash tool took about 1.2 s with no prompts.
- In a fresh session, the SessionStart hook's `additionalContext` reached the model and the MCP tool
  `mcp__plugin_compass-spike_compass__hello` worked.
- PreToolUse and PostToolUse `additionalContext` reach the model for Read and Edit. Hooks fire inside
  subagents and include `agent_id`/`agent_type`.
- Uninstall removes the registration and the data directory at once. The cache copy is marked
  `.orphaned_at` and purged about 14 days later.

## Alternatives considered
- **Stay on Python (uv tool).** Every machine at the company already has Python and uv. Rejected: the
  interpreter and venvs live outside the plugin directory, so "delete it and nothing is left" can't
  hold, and Python's 30–50 ms startup is paid on every PreToolUse hook call.
- **Rust.** Similar binaries at a higher authoring cost. **Compiled Bun/Deno:** 50–100 MB binaries.
  **Bash:** the source of the recorded pipefail bugs, and no Windows path.
- **Download binaries on first run** into `${CLAUDE_PLUGIN_DATA}`. This keeps the repo small, but it
  needs network on first use and has a race with the first hook. Deferred; switching later doesn't
  change anything users see.
- **Keep hooks as standalone scripts.** Rejected: version skew between hooks and CLI, and files
  scattered outside the unit.

## Consequences
- The plugin's `bin/` is **appended** to the Bash tool's `PATH`, so any other `compass` wins. The
  Python `compass-cli` must be uninstalled on migration, and `compass doctor` flags any shadowing
  binary. Hooks always use `${CLAUDE_PLUGIN_ROOT}/bin/compass`, since `bin/` isn't on hooks' `PATH`.
- Every release adds about 30 MB of binaries to git history (option A). Revisit if the repo size
  hurts.
- PreToolUse hooks block the tool call on crash or timeout, so every hook path must exit 0 on
  internal error, within a budget under 50 ms with no network.
- Uninstall leaves the cache for up to 14 days. That's the plugin system's behaviour, not compass's.
- Embedding dedup can't live in the core (onnxruntime needs cgo). It's resolved by
  [0011](0011-dedup-by-index-judge-not-embeddings.md).
- `install.sh`, `scripts/`, `tools/` and `cli/` are retired as their subcommands land.
