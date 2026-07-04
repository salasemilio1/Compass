# compass

compass is a memory system for Claude Code. It stores project context in your repo so each session starts informed instead of cold.

The problem it solves: Claude sessions are stateless. Every time you open a new session you re-explain what you're building, where you left off, and what decisions you've already made. compass stores all of that in a `.compass/` folder in your repo and loads it at the start of each session.

## What you need

- git
- [Claude Code](https://claude.ai/code) CLI

Optional: context-mode MCP (sandboxes large command output so it doesn't eat your context window — useful for test runs and diffs inside sessions).

## Install

```bash
git clone https://github.com/salasemilio1/dotfiles
cd dotfiles
bash compass/install.sh
```

The script is safe to re-run. If you already have a tool set up it will say so and move on.

## Commands

Use these as slash commands inside any Claude Code session.

| Command | What it does |
|---|---|
| `/init-compass` | Add compass to a project (run once per project) |
| `/warm-up` | Load project context at the start of a session |
| `/wrap-up` | Save what you built at the end of a session |
| `/explain <thing>` | Quick explanation of code or a compass entry |
| `/new-experiment` | Log an ML run with config, results, and notes |

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

## graphify (optional navigation layer)

[graphify](https://github.com/safishamsi/graphify) turns a codebase into a queryable structural graph — "what calls what, where does X live" — so an agent can navigate without repeatedly grepping and reading files. It's a separate, optional layer:

| Layer | Answers |
|---|---|
| **graphify** | "What calls X? Where does Y live? How is Z connected?" — auto-derived, regenerated from code |
| **context-mode** | Keeps large tool output (diffs, test runs, build logs) out of your session window |
| **compass** | "Why was this built this way? What did we decide? What's the status?" — curated, human-reviewed |

`install.sh` installs graphify's CLI/skill for you (`uv tool install graphifyy && graphify install`, or `pipx`/`pip` as a fallback). Code files (`.py`, `.json`, `.sh`, etc.) are parsed locally via tree-sitter — no LLM calls. Only prose files (`.md`, `.txt`, `.rst`) go through an LLM for concept/relationship extraction.

To keep that LLM call local instead of hitting a cloud API, point graphify at Ollama:

```bash
export OLLAMA_BASE_URL=http://localhost:11434/v1
export OLLAMA_MODEL=<your pulled model>   # e.g. qwen2.5-coder:7b
```

With no cloud API key set in your environment, graphify auto-detects Ollama. If a cloud key is present, pass `--backend ollama` explicitly. `warm-up` will use an existing `graphify-out/graph.json` for structural lookups when present, and stays silent about it otherwise.

## What gets installed

The install script sets up:

- Three slash commands linked into `~/.claude/skills/`
- A `SessionStart` hook that nudges you to run `/warm-up` when you open a repo that has `.compass/`
- Nothing else — no shell config changes, no package installs
