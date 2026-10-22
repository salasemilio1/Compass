# Compass v2

**Status:** proposal · **Date:** 2026-10-20 · **Replaces:** SPEC.md v1 once built

Compass keeps project context in the repo so agent sessions are cheap to start and safe to throw away.
v1 proved the idea. It also stopped scaling on a long-running project. This doc proposes v2.

## Why v2

The goal hasn't changed. A fresh session should get back up to speed in seconds, in any agent tool, without
someone spending ten minutes re-explaining the project. A second goal is new. Agents should get the last
agent's notes about a piece of code at the moment they're about to change it, the way a human teammate would
remember.

v1 falls short of both on ebay-flipper, its heaviest user (246 commits, 208 branches, 55 worktrees, 523
compass files):

| Symptom | Measured | Cause |
|---|---|---|
| Warm-up is expensive | `INDEX.md` is 117 KB (about 29k tokens). With `current-state.md` it's about 35k tokens before the task starts. | The index is hand-written and grows one long row per entry, forever. |
| Archival and dedup never ran | 109 log rows against a cap of 15. 181 learnings, never deduplicated. | `install.sh` doesn't install `scripts/compass-check.sh` or `tools/learning-dedup`. The skills call them by a path that only exists inside the Compass repo. |
| Merge conflicts and drift | 100 of 246 commits touch `INDEX.md`. Every worktree edits it and `current-state.md`. | Derived data is stored as source. One shared state file for every branch. |
| Plans crowd out memory | `context/` holds 104 files and 1.6 MB, mostly plans up to 53 KB. | No separate home for working documents. |
| Memory only grows | Nothing expires, merges or gets re-checked. | No lifecycle. |
| Wrap-up slows delivery | A human approves every write. | Work now arrives as a ticket and leaves as a PR. The PR is already the review. |

## Goals

1. Session start costs at most 2k tokens, no matter how old the project is.
2. Before an agent edits a file, it sees the notes that matter for that file.
3. A session can die at any moment (usage limit, crash, switching from Claude Code to Cursor) and the next one
   resumes from where it stopped.
4. An agent can install and configure Compass from the repo link alone. The only human step is restarting
   the session once.
5. Memory needs no review beyond normal PR review.

Non-goals for v2: cross-repo search, a hosted service, replacing code review, storing secrets.

## Principles

- **Markdown is the source of truth.** Everything else (the index, the search database, session bookkeeping)
  is derived and can be deleted and rebuilt.
- **The binary is deterministic. The agent writes the prose.** The binary finds, ranks, validates, fills
  metadata and hands out templates. Anything that needs judgment or summarizing is the agent's job, done
  through the same tools.
- **Behavior lives in one binary.** Each agent tool integration is just configuration that calls it. Adding a
  new tool means writing a config, not porting logic.
- **Push beats pull.** Agents forget to search. The important context gets injected by hooks at the right
  moment. Search is there for everything else.
- **Budgets are enforced in code.** Every output that reaches an agent has a token cap. CI fails when a new
  or changed file goes over its cap.
- **The PR is the gate.** Memory changes ship in the same PR as the code they describe and get reviewed for
  intent like everything else.

## Architecture

```
            .compass/  (Markdown + front-matter, in git)       source of truth
                 │  indexed incrementally (mtime, then content hash)
                 ▼
       .compass/.index.db  (SQLite + FTS5, WAL, gitignored)    derived cache + session bookkeeping
                 │
    ┌────────────┼───────────────┬──────────────────┐
    ▼            ▼               ▼                  ▼
  MCP server    Hooks           CLI                AGENTS.md section
  (agents       (push context   (CI, scripts,      (the guaranteed fallback
   pull)         at the right    humans)            for tools without hooks)
                 moment)
```

All four surfaces call the same engine package. The engine parses entries, keeps the index current, ranks
results and fits them into a token budget.

### Who does what

| Job | Binary | Agent |
|---|---|---|
| Find and rank entries, fit them to a budget | yes | |
| Fill `paths`, dates and status; validate everything | yes | |
| Hand out a template for a log, decision or handoff | yes | |
| Write the log, decision, learning or handoff text | | yes |
| Decide which learnings say the same thing | | yes, with candidates from the binary |
| Draft `brief.md` and `areas.yml` at init | | yes, from a template |

## The `.compass/` folder

```
.compass/
  brief.md                       what the project is and how it's built (capped)
  areas.yml                      area name → path globs
  areas/<area>.md                optional one-page overview per area (capped)
  state/<branch>.md              the handoff for one branch
  decisions/YYYY-MM-DD-slug.md   why a choice was made
  learnings/slug.md              gotchas worth never repeating
  issues/slug.md                 known problems that are still open
  logs/YYYY-MM-DD-slug.md        what a unit of work changed
  experiments/YYYY-MM-DD-slug.md ML runs
  plans/slug.md                  working plans, never auto-loaded
  docs/slug.md                   long reference material, never auto-loaded
  scratch/                       gitignored, never indexed
  .index.db                      derived, gitignored
```

`INDEX.md` goes away as a hand-written file. `compass index --md` prints one for humans who want it, and
`check` fails if someone commits an edited one.

An entry's id is its file path stem, like `decisions/2026-10-19-push-retries-idempotent`. There are no
sequence numbers, so 55 worktrees can't hand out the same id. `check` catches the rare slug collision at
merge.

The indexer skips `scratch/`, dot-files and anything gitignored. ebay-flipper's `scratch/` alone is 163 MB.

### Entry format

Every entry starts with front-matter.

```yaml
---
type: decision            # decision | learning | issue | log | experiment | plan | doc
title: Retries on push routes must be idempotent
summary: Push retries can fire twice after a timeout, so every push handler checks the delivery id first.
date: 2026-10-19
areas: [api]
paths: [internal/api/push/**]
status: active            # active | superseded | resolved | done | archived
supersedes: [decisions/2026-09-02-push-retry-backoff]
---
```

`title` states the claim. `summary` is one sentence that search results and injections show without opening
the file. `paths` lists at most five files or globs central to the entry. When a change is broader than that,
`paths` stays empty and the entry relies on `areas`.

### State files

`state/<branch>.md` is the handoff: what's being done, what was tried, what's next, what's blocking. It's
committed with the branch so another machine or tool can pick it up. Branch names map to files with `/`
replaced by `--`. A detached HEAD uses the worktree folder name.

State files are short-lived. `compass wrap` deletes the branch's state file in the PR's final commit, so it
never lands on main. `compact` sweeps any left behind by branches that merged or were deleted. A renamed
branch keeps its state if `compass handoff` runs once after the rename, since it looks up the old name by the
worktree.

Project-wide problems don't go in state files. They're `issues/` entries, so they survive branches and
show up in `related()`.

### Budgets

All caps are in tokens, estimated as bytes / 4.

| Output | Cap |
|---|---|
| Session start injection (brief, this branch's state, open items) | 2,000 |
| `brief.md` | 800 |
| `state/<branch>.md` | 600 |
| `areas/<area>.md` | 500 |
| Pre-edit injection | 400, at most 3 entries |
| Search results | 10 hits of id, title and summary |
| learning body | 350 |
| issue body | 400 |
| decision body | 800 |
| log body | 1,000 |
| plans, experiments, docs | no cap, never auto-loaded |

Open items in the start pack are the titles of active issues and active plans, newest first, cut to fit.

Body caps apply to new and changed entries only. Migrated entries are grandfathered until someone edits them.
The caps sit near the 75th percentile of ebay-flipper's existing entries, so normal writing passes. `check
--fix` never truncates text. Over-budget files fail the check and the agent rewrites them.

## Interfaces

### MCP tools

| Tool | Returns |
|---|---|
| `brief()` | The session start pack |
| `related(paths)` | Entries for those files and their areas, best first |
| `search(query, type?, k?)` | ids, titles and summaries |
| `get(id)` | One full entry |
| `template(type)` | The skeleton and rules for a new entry or handoff |
| `note(type, title, summary, body, paths?)` | Validates and writes a new entry |
| `handoff(doing, tried, next, blocked)` | Validates and overwrites this branch's state file |

The server finds the repo from MCP roots when the client sends them, then from `--root`, then from the git
root of its working directory. A user-level config in some tools starts the server outside the project, so
`compass install` always writes `--root` where it can.

The server also ships MCP server instructions. Some clients add those to the system prompt and some don't, so
they're a bonus. The real fallback is a short Compass section in AGENTS.md that tells agents to call
`related` before editing and `note` when something surprises them.

### Hooks

| Event | Runs | Effect |
|---|---|---|
| Session start | `compass hook start` | Injects `brief()` |
| Before an edit | `compass hook pre-edit` | Injects `related(file)`, skipping entries this session has already seen |
| Stop, before compaction | `compass hook checkpoint` | Updates the mechanical part of the state file: branch, last commit, changed files, active plan. It writes only if something changed. |

The pre-edit matcher covers every edit tool: `Edit`, `Write`, `MultiEdit`, `NotebookEdit` in Claude Code
and `apply_patch` in Codex. Each hook process is separate, so the "already seen" list lives in `.index.db`,
keyed by the session id the client passes in.

Hooks must never get in the way. Each one has an internal timeout of 200 ms and always exits 0. On any error
it prints nothing. The pre-edit path checks file mtimes and only hashes files that changed, so it stays well
under 50 ms on a repo the size of ebay-flipper. SQLite runs in WAL mode so parallel sessions can read while
one writes.

Hooks are subcommands of the binary. Nothing gets installed as a loose script.

### CLI

`init`, `install`, `index`, `brief`, `search`, `related`, `get`, `template`, `note`, `handoff`, `wrap`,
`check`, `compact`, `migrate`, `pr-summary`, `mcp`. Output is compact by default, with `--json` for scripts.

## Retrieval

`related(paths)` is deterministic. It drops superseded, resolved and archived entries. It ranks the rest by
how specifically they match:

1. an exact file in `paths`
2. a glob in `paths` that covers the file
3. a directory glob
4. an area that covers the file

Recency breaks ties. At most two entries of one type make the cut, so learnings and issues don't get pushed
out by a run of new decisions in a busy area. The area overview goes first when one exists and fits the
budget.

This matters in hot code. In ebay-flipper, 52 entries mention `internal/deals`. Sorting by type and date
would fill every slot with recent decisions and never show a learning.

`search` is BM25 over title, summary, body and areas through SQLite FTS5. Vector search is phase 2. It only
ships if the eval set shows it beats BM25. At this scale the vectors can live as blobs and be compared by
brute force in Go, so no SQLite extension is needed.

A missing or corrupt `.index.db` triggers a full rebuild, which takes about a second for 500 files.

## Workflows

### Human

| Step | You | Compass |
|---|---|---|
| 1 | Write the ticket: problem, examples, goal, constraints | |
| 2 | Hand it to an agent | |
| 3 | Review the PR for intent and what changed | `pr-summary` adds a "Memory changes" section listing new, changed and superseded entries |
| 4 | Merge | Memory merges with the code. The branch's state file is already gone. |
| Monthly | Review the compaction PR | |

Nobody runs warm-up or wrap-up by hand.

### Agent

| Phase | What happens |
|---|---|
| Start | The start hook injects the brief, this branch's state and open items. |
| Orient | Reads the ticket. Calls `related` or `search` for the task area. Writes `plans/<slug>.md` for non-trivial work. |
| Edit | The pre-edit hook injects the top entries for each file the first time they're relevant. |
| During | Calls `note()` when something surprises it. Calls `handoff()` at natural stopping points. The checkpoint hook keeps the mechanical part of state current. |
| Session dies | The next session, in any tool, resumes from `state/<branch>.md`. |
| Before the PR | Runs `compass wrap`. The binary lists the commit range and changed files and returns templates. The agent writes the log and any decisions, learnings or issues, then calls `wrap --finish`. That fills `paths`, marks the plan done, deletes the state file, runs `check` and commits. |
| CI | `compass check` passes or the PR is blocked. |

## Keeping quality up without a human gate

`compass check` runs in pre-commit and CI. It fails on:

- missing or invalid front-matter
- a new or changed file over its budget
- a slug collision
- a `supersedes` pointing to an entry that doesn't exist
- a committed state file on main
- an edited `INDEX.md`

It warns on:

- `paths` that match no file in the repo
- text that reads like instructions to an agent ("ignore previous", "you must run"). Memory gets injected
  into prompts, so a bad entry is a prompt-injection path.

`compass compact` runs monthly as an agent task and opens a PR. The binary proposes near-duplicate learnings
by BM25 overlap, entries whose paths are all gone, and stale state files. The agent merges, archives or
leaves each one and suggests learnings to promote into AGENTS.md, a test or a lint rule. A learning that's
really a rule belongs in code.

## Install and packaging

`INSTALL.md` is written for agents. It has exact commands and no choices. A person pastes the repo link and
says "install this." The agent then:

1. Installs the binary (`brew install`, or a `curl | sh` script that picks the right release).
2. Runs `compass install`. It detects which agent tools are present and writes each one's MCP config, hooks
   and the AGENTS.md section. If the Claude Code plugin is already installed, it skips Claude Code. Safe to
   re-run.
3. Runs `compass init` in the project. It creates `.compass/` and returns templates. The agent drafts
   `brief.md` and `areas.yml` from the code. The binary gitignores `.index.db` and `scratch/`.
4. Tells the person to restart the session so the hooks load.

For Claude Code, Compass also ships as a plugin that bundles the MCP server, hooks and a short skill in one
install.

| Tool | Pull (MCP) | Push | Fallback |
|---|---|---|---|
| Claude Code | yes | before the edit | |
| Cursor | yes | session start, and after the first edit in an area (its pre-tool hook can't add context) | AGENTS.md |
| Codex and others | yes | varies | AGENTS.md |
| CI | | | `check` and `pr-summary` through the CLI |

## Tech stack

| Layer | Choice |
|---|---|
| Language | Go, one static binary |
| Index | SQLite through `modernc.org/sqlite` (pure Go) with FTS5, WAL mode |
| Parsing | `goldmark` and `yaml.v3` |
| MCP | Official Go SDK, stdio transport |
| Vectors (phase 2) | EmbeddingGemma through Ollama, brute-force cosine in Go |
| Release | GoReleaser to GitHub releases and a Homebrew tap |
| CI | `compass check` as a pre-commit hook and a GitHub Action |

## Migrating from v1

`compass migrate` converts a v1 `.compass/` in one PR. The mechanical part is deterministic. It moves files,
adds front-matter and grandfathers bodies, so the diff is mostly renames a reviewer can skim. The agent pass
after it only writes new files: `brief.md`, `areas.yml` and the issues pulled out of `current-state.md`.

| v1 | v2 |
|---|---|
| `INDEX.md` | deleted. Its rows fill `summary` where an entry has none. |
| existing `name` / `description` front-matter (113 learnings, 51 decisions in ebay-flipper) | `description` becomes `summary`, `metadata.type` becomes `type` |
| `context/current-state.md` | known issues become `issues/` entries. The rest becomes `state/main.md`, and the full original is kept in `docs/`. |
| `context/overview.md`, `architecture/overview.md` | input for the agent's `brief.md` draft, then moved to `docs/` |
| `context/*-plan.md` | `plans/` |
| other `context/`, `architecture/`, `design-system/` files | `docs/` |
| `decisions/NNNN-slug.md` | `decisions/YYYY-MM-DD-slug.md`, dated from git history |
| `learnings/`, `logs/`, `experiments/` | same folders with front-matter added. Experiments get it for the first time. |
| `logs/archive.md` | split back into entries, marked `archived` |
| `learnings/.embedding-cache.json` | deleted |
| `paths` | backfilled from the commit that added the entry, only when it touched five files or fewer |
| `scratch/` | left alone |

## Measuring it

Three numbers, tracked before and after migrating ebay-flipper:

- tokens injected at session start
- the share of pre-edit injections followed by a `get` on one of the injected ids, or the id cited in the
  session's notes
- score on a set of about 20 questions a new agent should be able to answer about the project, each paired
  with the entry that answers it

## Risks

- **Memory quality drops without a human gate.** Mitigated by `check` in CI, budgets and monthly compaction.
  If it still bloats, tighten the budgets.
- **Push depends on each tool's hooks.** Claude Code gets everything. Cursor gets context after the first
  edit, not before. Everything else falls back to AGENTS.md, which suggests rather than enforces.
- **Areas need upkeep.** A new package outside every glob gets no area notes. `check` warns when a changed
  file matches no area.
- **Migrated bodies stay long.** Grandfathering keeps migration reviewable but leaves old entries over
  budget. Compaction trims them over time.

## Milestones

1. Engine and CLI: `index`, `brief`, `related`, `search`, `check`, `migrate`. Migrate a copy of ebay-flipper
   and measure the three numbers.
2. Hooks, MCP server and `install` for Claude Code, including the plugin.
3. `wrap`, `pr-summary`, the CI action, and config for Cursor and Codex.
4. `compact`, then vectors if the evals call for them.
