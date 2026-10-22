# compass

compass keeps a project's working memory in the repo, in `.compass/`, so any agent session is
disposable: a fresh agent (Claude Code, Codex, Cursor, anything that reads `AGENTS.md`) runs
`compass resume` and knows where the work stands, what was decided, and what is uncommitted,
without a discovery phase. When one agent hits its usage limit, another picks up from there.

compass v3 is one static Go binary. The v2 tools below (`install.sh`, the skills, the Python
`compass-cli`) still work and are retired as the Go commands replace them.

## Install (v3)

**Claude Code plugin** (binaries are committed; no Go, Python or network needed after install):

```bash
claude plugin marketplace add salasemilio1/Compass
```

```bash
claude plugin install compass@compass
```

The plugin's `SessionStart` hook injects the resume brief (a short stub with the objective and next
step when current-state is current and the tree is clean on a fresh start), `Stop` asks for an updated
current-state once when code was committed after it, and `PreCompact` asks once before a manual
`/compact`. Inside Claude Code the agent's Bash tool has the plugin's `compass` on PATH, and the
git pre-commit hook finds the plugin's copy on its own. For your own terminal, install a release
binary as below (the plugin's cache path changes with every plugin version, so don't link to it).

**Without Claude Code**, from a release (macOS and Linux, arm64 and amd64):

```bash
gh release download --repo salasemilio1/Compass --pattern "compass_*_$(uname -s | tr A-Z a-z)-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/').tar.gz"
```

Unpack it and put `compass` on your PATH. A file downloaded by a browser on macOS is quarantined;
clear it with `xattr -d com.apple.quarantine compass`. Or, with Go installed:

```bash
go install github.com/salasemilio1/Compass/cmd/compass@latest
```

**Uninstall:** `claude plugin uninstall compass@compass` (and `claude plugin marketplace remove
compass`), or delete the binary. Nothing else is written outside your repositories.

## Use (v3)

```bash
compass init        # once per repo: .compass/, the AGENTS.md block, a git pre-commit hook;
                    # fill in the current-state template before your first commit (check rejects it)
compass resume      # what a fresh agent reads first
compass record decision --claim "Prices come from the model's own fit" --premise "goal:quick-flip"
compass check       # caps, anchor, files in play, claims, supersedes, refs (pre-commit and CI)
compass index       # regenerate .compass/INDEX.md (pre-commit does this)
compass doctor      # install problems, such as an old compass shadowing this one
```

`compass help` lists every flag; `check`, `resume`, `record` and `doctor` take `--json`. Exit status: 0 ok, 1 findings or failure, 2 usage error.

**The format.** `.compass/context/current-state.md` is the front door: at most 80 lines
(objective, just done, files in play, next steps), rewritten whole at each boundary, with a
`Commit:` line set from `git rev-parse HEAD`. Decisions, goals and experiments are dated records
that are never edited; each has frontmatter:

```yaml
---
claim: one line, what holds (required, at most 200 characters)
date: 2026-10-21
scope: global          # optional: resume delivers live global records to every session
supersedes:            # optional: the records this one replaces; their status becomes superseded
  - "0006"
premises:              # optional: what the decision rests on, e.g. goal:<slug>
  - goal:portable-memory
refs:                  # optional: paths and commit:<hash> are checked; other schemes are left alone
  - internal/compass/check.go
---
```

`INDEX.md` is generated, never hand-edited. Status is derived from later records' `supersedes:`.
A Claude-memory-style `description:` is read as the claim.

**CI:** run `compass check --require-fresh` on pull requests with full history
(`actions/checkout` with `fetch-depth: 0`), so a PR that changes code fails if current-state was
not rewritten after it.

## Migrating from v2 and the Python `compass-cli`

1. Remove the Python CLI. It sits earlier on PATH and shadows the Go binary (the plugin's `bin/`
   is appended to PATH, so any other `compass` wins):

   ```bash
   uv tool uninstall compass-cli
   ```

   (or `pipx uninstall compass-cli`). `compass doctor` reports any `compass` shadowing this one,
   and the SessionStart brief warns the agent and names the plugin's binary until you do.
2. In each repo, run `compass init`. It keeps your files and adds what is missing: the v3 block in
   `AGENTS.md`, a one-line pointer in `CLAUDE.md`, the pre-commit hook (only if you have none;
   otherwise add `compass hook pre-commit` to yours). Delete the old v2 instructions it reports
   (the `compass-spec-version` text in `AGENTS.md`, "Read `.compass/INDEX.md` first" in
   `CLAUDE.md`).
3. Run `compass check` and work through it: add a `claim:` to each record (or keep
   `description:`), restate current-state under 80 lines, split records over 300 lines. Folders
   v3 doesn't use (`logs/`, `learnings/`) are left alone and counted in the index.
4. Remove v2's global hooks and skills from `~/.claude` if `install.sh` put them there; the plugin
   replaces the SessionStart nudge.

## v2 (legacy)

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
