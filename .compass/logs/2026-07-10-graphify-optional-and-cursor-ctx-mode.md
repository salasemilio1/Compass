# graphify becomes opt-in; dedup stage 3 verified for real; context-mode confirmed in Cursor

Commits: df7f6ee..HEAD (uncommitted at time of writing)

## Why

Graphify (the structural-graph navigation layer) is ~3.2MB, roughly half of compass's on-disk
footprint, and — despite decisions/0005's `PreToolUse` guard built specifically to force
`graphify query` usage — a week of real sessions still produced fewer than 5 uses. The user asked
for a determination: is this "not used correctly" or "used but not useful," with removal or fix as
the outcome. Separately, two backlog items remained from the prior session: verify learnings-dedup
stage 3 against a real `fastembed` run (never confirmed, only mock-calibrated), and get
context-mode working in Cursor (the user's other coding agent), since it had been Claude-Code-only
so far.

## What changed

**graphify → opt-in add-on (decisions/0008, supersedes 0001, amends 0005):**
- Live-tested `graphify query`/`explain` this session — confirmed genuinely useful for
  cross-language call-graph questions (e.g. bash `install.sh` calling into Python `init.py` in one
  shot), weaker for markdown/doc questions. The gap was demand, not capability or enforcement:
  decisions/0005's guard had a full week of unconditional enforcement on broad-search commands and
  still didn't move usage.
- `install.sh` now skips `install_graphify` unless `COMPASS_WITH_GRAPHIFY=1` is set.
- `hooks/tool-routing-guard.py` dropped its graphify-routing rule entirely; broad
  greps/finds/ripgrep now always redirect to `ctx_execute`, regardless of whether a graph exists.
  Simplified `classify()`'s signature (no longer takes `graph_exists`). Updated
  `tests/test_tool_routing_guard.py` to match.
- README/AGENTS.md/`skills/warm-up/SKILL.md` updated to describe graphify as opt-in, not standard.
- Deleted the four stray `*.backup-20260704100310` symlinks under `~/.claude/` (pre-dated the
  copy-not-symlink switch in decisions/0006).
- Ran `install.sh` for real (not just the test harness) to bring `~/.claude/hooks/`,
  `~/.claude/skills/` up to date with the guard/skill changes.

**Learnings-dedup stage 3 verified end-to-end:** ran `check_similarity.py` for real (via `uv run`,
pulling the actual `fastembed` dependency) against two hand-written paraphrases of an existing
learning. A loose paraphrase scored 0.59 cosine — correctly stayed `OK`. A tight paraphrase scored
0.85 — correctly printed `POSSIBLE_DUPLICATE ... embedding:0.85`. Also resolved the "model weights
not found" mystery from the prior audit: `fastembed`'s default cache is the OS temp dir
(`$TMPDIR/fastembed_cache`), not `~/.cache` — the model had never failed to download, the wrong
directory was checked.

**context-mode confirmed working in Cursor:** `context-mode` is a generic MCP server (npm package
by mksglu, built on `@modelcontextprotocol/sdk`), not Claude-Code-specific — explicitly documented
to support Claude Code, Gemini CLI, VS Code Copilot, OpenCode, and Codex CLI. Added
`~/.cursor/mcp.json` (global) pointing at the already-installed `context-mode` binary; user
confirmed it shows up as an MCP server in Cursor.

**Bug fixed along the way:** `install.sh`/README documented
`claude mcp add context-mode npx @anthropic-ai/mcp-server-context-mode` as the install command —
that package returns a 404 on npm, it never existed. Traced the real binary
(`/opt/homebrew/bin/context-mode` -> `context-mode@1.0.169` via `npm ls -g`) and fixed
`install_context_mode()` to `npm install -g context-mode` + `claude mcp add context-mode --
context-mode`, matching what was actually working. Same doc-vs-reality gap as the earlier
"installed claim vs actual deployment" learning.

## Learnings produced

- `learnings/context-mode-package-name-was-wrong.md` — the documented install command referenced a
  nonexistent npm package.
- `learnings/dedup-stage3-never-verified-with-real-embedder.md` — rewritten to record the real
  verification result and the `$TMPDIR` cache-location fix.

## Assumptions changed

- graphify is no longer assumed to be present in a compass-adopted repo by default; any code/docs
  that reference it should treat `graphify-out/graph.json` as possibly absent (most already did).
- `context-mode` is confirmed cross-agent, not Claude-Code-only — future compass work aimed at
  other agents (Cursor, etc.) can lean on it without re-verifying compatibility.
