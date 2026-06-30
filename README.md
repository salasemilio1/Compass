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

## How a session works

1. Run `/init-compass` once when you start a new project.
2. Start each session with `/warm-up` — it reads the index and loads only what's relevant.
3. Do your work.
4. End with `/wrap-up` — it reads your git diff and drafts a log entry for you to review before saving.

Your context lives in `.compass/`. Commit it to keep history across machines, or add it to `.gitignore` to keep it local only.

## context-mode MCP (optional)

context-mode runs large command output (test suites, diffs, build logs) in a sandboxed context so the results don't consume your main session window. Install it with:

```bash
claude mcp add context-mode npx @anthropic-ai/mcp-server-context-mode
```

Run `claude mcp list` to confirm it's active. Once installed, Claude Code will use it automatically when compass wrap-up runs git diffs.

## What gets installed

The install script sets up:

- Three slash commands linked into `~/.claude/skills/`
- A `SessionStart` hook that nudges you to run `/warm-up` when you open a repo that has `.compass/`
- Nothing else — no shell config changes, no package installs
