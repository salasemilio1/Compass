# compass

compass is a memory system for Claude Code. It stores project context in your repo so each session starts informed instead of cold.

The problem it solves: Claude sessions are stateless. Every time you open a new session you re-explain what you're building, where you left off, and what decisions you've already made. compass stores all of that in a `.compass/` folder in your repo and loads it at the start of each session.

## What you need

- git
- [Claude Code](https://claude.ai/code) CLI

Optional: context-mode MCP (sandboxes large command output so it doesn't eat your context window — useful for test runs and diffs inside sessions).

## Install

```bash
git clone https://github.com/salasemilio1/Compass
cd Compass
bash install.sh
```

The script is safe to re-run. If you already have a tool set up it will say so and move on.

## Commands

Use these as slash commands inside any Claude Code session.

| Command | What it does |
|---|---|
| `/init-compass` | Add compass to a project (run once per project) |
| `/warm-up` | Load project context at the start of a session |
| `/wrap-up` | Save what you built at the end of a session |

`/init-compass` wraps the `compass` CLI, which does the actual scaffolding
(`compass init --mode committed\|branch-local`) deterministically — directories, `INDEX.md`,
`.gitignore`, `AGENTS.md`. That means it also works standalone, from a plain terminal, with no
Claude Code dependency — useful if you're wiring compass into another agent's workflow.

## How a session works

1. Run `/init-compass` once when you start a new project (or `compass init` directly).
2. Start each session with `/warm-up` — it reads the index and loads only what's relevant.
3. Do your work.
4. End with `/wrap-up` — it reads your git diff and drafts a log entry for you to review before saving.

Your context lives in `.compass/`. Commit it to keep history across machines, or add it to `.gitignore` to keep it local only.

## context-mode MCP (optional)

context-mode runs large command output (test suites, diffs, build logs) in a sandboxed context so the results don't consume your main session window. Install it with:

```bash
npm install -g context-mode
claude mcp add context-mode -- context-mode
```

Run `claude mcp list` to confirm it's active. Once installed, Claude Code will use it automatically when compass wrap-up runs git diffs.

context-mode is a standard MCP server, not Claude-Code-specific — it also works in any other MCP-compatible client (Cursor, VS Code Copilot, Gemini CLI, etc.). For Cursor, add the same stdio command to `~/.cursor/mcp.json` (global) or `<project>/.cursor/mcp.json` (project-scoped):

```json
{
  "mcpServers": {
    "context-mode": { "command": "context-mode", "args": [] }
  }
}
```

## graphify (optional navigation layer)

[graphify](https://github.com/safishamsi/graphify) turns a codebase into a queryable structural graph — "what calls what, where does X live" — so an agent can navigate without repeatedly grepping and reading files. It's a separate, optional layer:

| Layer | Answers |
|---|---|
| **graphify** | "What calls X? Where does Y live? How is Z connected?" — auto-derived, regenerated from code |
| **context-mode** | Keeps large tool output (diffs, test runs, build logs) out of your session window |
| **compass** | "Why was this built this way? What did we decide? What's the status?" — curated, human-reviewed |

**Not installed by default.** A week of real usage showed `graphify query` was called fewer than 5 times despite a routing guard built specifically to force it (decisions/0005) — compass's own scale rarely creates a case where the graph beats grep. Opt in with `COMPASS_WITH_GRAPHIFY=1 ./install.sh`, or install it directly (`uv tool install graphifyy && graphify install`). compass deliberately uses only graphify's **deterministic** mode — `graphify update <path>` — which parses code via tree-sitter and markdown structure/links via regex. No LLM, no API key, no network call, no backend to configure. graphify also supports a semantic LLM-extraction layer for richer cross-document relationships, but compass doesn't wire that in: it adds a dependency and latency/cost tradeoff that isn't worth it for a tool meant to be easy and orthogonal. If you want that layer, run graphify directly outside of compass's wiring.

Once opted in and a graph exists (`graphify update .` in the target repo), the `SessionStart` hook keeps it fresh automatically — it re-runs `graphify update .` in the background on every session start, silently, since it's free and instant. `warm-up` prefers `graphify query` over grepping for structural lookups when a graph exists, and stays silent about it otherwise.

## What gets installed

The install script sets up:

- Three slash commands linked into `~/.claude/skills/`
- A `SessionStart` hook that nudges you to run `/warm-up` when you open a repo that has `.compass/`
- A `Stop` hook that nudges toward `/wrap-up` once context usage crosses 75% of the window
- A `PreToolUse` hook that deterministically routes heavy `Bash` calls (broad greps/finds, full
  diffs, unbounded logs, test/build runners) toward `ctx_execute` instead of leaving it to agent
  judgment — see `.compass/decisions/0005-deterministic-tool-routing-guard.md`
- The `compass` CLI (`compass init`), installed via `uv tool`/`pipx`/`pip` from this checkout
- No shell config changes; the only package installs are the optional/adjacent tools above
  (compass CLI, context-mode, and graphify if opted in)
