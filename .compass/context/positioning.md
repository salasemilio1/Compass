# Compass v3: positioning and context delivery

Date: 2026-10-21
Status: live, a proposal under review. It feeds decisions 0010 (Go + plugin) and 0011 (dedup).
Evidence: [2026-10-15-ebay-flipper-retro.md](2026-10-15-ebay-flipper-retro.md),
`experiments/2026-10-21-dedup-architectures.md` (branch `claude/dedup-experiment-436ce8`), and the
plugin hook spike described below.

## The problem compass exists for
An agent's context window should not be the source of truth for a project. Without compass, the two
failure modes are:
1. **Expensive resumes.** Long-lived sessions re-read ~800k tokens every morning, about a quarter of a
   5-hour usage window, just to get back to where they were.
2. **Uninformed reversals.** A fresh agent, often a subagent, proposes undoing an architectural choice
   because it can't see the constraint that produced it.

## Where compass sits
Before agents, project knowledge lived in four layers. Agents keep three of them and lose the fourth:

| Layer | Owns | Compass's relationship |
|---|---|---|
| Code | What the system does | None |
| Comments | Why *this line* | Enforces it: a rule in compass's managed `AGENTS.md` block, "comment the why", since agents under-comment |
| Git (commits, PRs, blame) | What changed and when | Points to commits and never restates them, so there is no `logs/` category |
| `CLAUDE.md` / `AGENTS.md` | Rules every session follows | Promotes learnings into a managed, capped block |
| Issue tracker | Tasks and backlog | None. current-state holds only "now and next" |
| **Colleagues** | **Where we are, why, and what matters for what you're about to touch** | **This is compass** |

**Litmus test for every write:** could a fresh agent derive this from the code plus `git log`
cheaply? If yes, compass doesn't store it. It points to it.

**What compass holds:** current state, decisions (with their premises), live plans, and experiment
results. Learnings route out: to a rule, to a comment, or merged into the decision they affect.

## Two layers: a front door and a scaling layer
**Front door: `current-state.md`.** It's capped (about 80 lines) and covers the objective, what was
just done, the files in play and the next steps. It loads at session start and is kept current by the
loop: updated at wrap-up, with its `Commit:` anchor checked. On any project, even an afternoon one,
the value is visible in the first session. It's also the evidence-backed core: in ebay-flipper it was
the most re-read file, by 22 later sessions.

**Scaling layer: decision delivery.** A single state file is short-term memory. As a project ages it
either grows into an unreadable index (v2's 103 KB INDEX.md) or drops the "why" exactly when a later
agent needs it, and it never reaches a subagent that opens `internal/db/` three hours into a task.
Decisions are long-term memory, delivered by the code they concern. A project with no decisions pays
nothing; a 650-PR codebase gets the shoulder tap.

**The scaling layer has to earn its place. Build order follows from that:**
1. Ship the front door.
2. Run the trigger eval.
3. Build delivery only if the eval clears its bar.

The delivery frontmatter (below) becomes required only when delivery ships.

**The trigger eval:**
- **Cases:** real incidents (reverted PRs, superseded decisions, a subagent undoing an architecture).
  Each is replayed in three arms:
  1. current-state alone;
  2. current-state plus heavy why-comments, the same decision text written into the constrained
     files;
  3. current-state plus delivery.

  If arm 2 matches arm 3, the answer is the comments rule and no delivery layer.
- **No leakage:** a decision's `constrains:` globs are derived only from history *before* the
  incident, and incidents are held out of any tuning.
- **Pass/fail bar, written down before it runs:** delivery must prevent at least a pre-registered
  share of the violations that current-state alone misses (set when the incident set is built,
  before results).
- **If it fails:** delivery is cut, and compass ships as a current-state maintainer.

## Decisions are the current best answer, not law
Projects change: ebay-flipper began as a personal dashboard. Compass prevents **uninformed**
reversals and encourages **informed** ones.
- A decision records `decision:` (what holds now, stated as fact, not as an order) and `premises:`
  (the assumptions it rests on). An agent challenging a decision checks the premises, not the
  conclusion. Premises may reference a project goal by tag (`goal:personal-dashboard`), which
  `context/overview.md` defines.
- **Superseding is cheap.** The agent proposes the change like any design change. If adopted, a new
  decision carries `supersedes:`, and the old one is marked superseded and stops being delivered.
- **Pivots surface their casualties.** When a goal tag is retired from `context/overview.md`,
  `compass check` lists every live decision whose premises reference that tag. The match is exact,
  not a fuzzy guess at prose.
- The `AGENTS.md` block states it: *decisions are the current best answer; if you see a better
  approach, propose it and address the premises.*

## Why not just comments?
Comments are the strongest alternative: co-located, zero infrastructure, and they work in any agent.
For a *local* why (this function, this line), comments win, and compass's `AGENTS.md` rule pushes
agents to write them. Comments break down for:
- **Decisions that span files.** One comment in one file is missed by agents editing the others. A
  copy in every file drifts the first time the decision changes.
- **Things that aren't in the code.** "We don't use Redis because …" and rejected alternatives have
  no line to attach to.
- **Lifecycle.** A comment has no status. When a decision is superseded, its 12 copies stay
  confidently wrong until someone greps for them, and a pivot can't find them by goal.
- **Cost on every read.** A comment is paid for on every read of the file, by every agent, even one
  fixing a typo. Delivery costs a decision once per context.
- **Partial reads.** An agent reading lines 300–400 or grepping never sees the header comment. A hook
  fires on the path, whatever range was read.

The rule that follows: local why goes in a comment; cross-file, negative or long-lived decisions go in
compass. The trigger eval's comments arm tests whether this distinction earns its cost.

## How delivery works
**What is injected: the claim, not a description.** A skills-style description says what a document
is *about*, which invites the agent to decide it doesn't apply. Compass injects the actionable line
itself, so there is no relevance to misjudge:

```
compass · decisions/0012 (2026-08): The store is a single SQLite file.
  rests on: one-binary deploy; single user.
```

The stance on how to treat these lines (current best answer, not law; challenge the premises; propose
superseding) lives once in the `AGENTS.md` block, not in every injection.

Frontmatter that makes this work. `compass check` and the MCP `record` tool enforce it:
- `claim:` one line, at most about 200 characters, never "see doc". A decision's claim is the
  decision; an experiment's claim is its result.
- `premises:` the assumptions it rests on.
- `constrains:` path globs. They default from git: the files changed in the PR that added the doc,
  minus shared files (lockfiles, `go.mod`, entry points, generated code).
- `scope: global`, for cross-cutting decisions with no natural path. These are injected at session
  start as a short, capped list instead of being path-triggered.
- `reversal_cost: low | high`
- `refs:` *optional*, typed links to systems that own the underlying facts: `commit:abc123`,
  `pr:123`, `issue:45`, `mlflow:run/7f3e…`. Include one only when a real reference exists. A doc
  whose evidence is just the code carries no `refs:`, and nothing is ever invented to fill the field.
  `compass check` validates refs that are present (a commit must exist) but never requires one.
- `status:` live, superseded or historical. Only live docs are delivered.

Provenance doesn't depend on the agent: the commit that added a doc is read from `git log`, never
written by hand.

**When it's injected, escalating with proximity to a change:**

| Agent action | Injected | Size |
|---|---|---|
| Reads a constrained path | claim + doc path | ~40 tokens |
| Edits it | claim + premises + rejected alternatives | ~120 tokens |
| First edit in a context to a path whose doc has `reversal_cost: high` | **Off by default (opt-in per project).** PreToolUse denies once: "read decisions/0012's premises first; superseding is fine if they no longer hold" | full doc read |

Grep and Glob results don't trigger injection. A search returns many paths, and the agent reading a
file is the real signal.

**Caps:**
- At most 3 entries and about 300 tokens per injection, with the most specific glob first.
- A **per-session budget** of about 2k tokens. Once it's spent, injections shrink to a one-line
  pointer ("compass: 3 more decisions apply to internal/db/; call context(path)").

**Showing each doc once per context.** A ledger in `${CLAUDE_PLUGIN_DATA}` is keyed by session, agent
(main or the subagent's id), doc and tier.
- A subagent starts with an empty ledger.
- Compaction clears the session's ledger, since that context is gone. Re-injections count against
  the session budget.
- A higher tier may re-show a doc.
- Writes are atomic appends, because parallel tool calls share the file.

**Bash:** best-effort parsing for read commands (`cat`, `sed -n`) and write commands (`sed -i`, `>`,
heredocs). Anything that doesn't parse is skipped without blocking, but **every skip is logged**, so
the skip rate is measured rather than invisible.

**Verified in the spike (2026-10-21):**
- PreToolUse and PostToolUse `additionalContext` both reach the model for Read and Edit.
- Hooks fire inside subagents, and the context reaches the subagent.
- Hook input carries `session_id`, plus `agent_id`/`agent_type` for subagents.
- The plugin's `bin/` is not on hooks' `PATH`, so hooks use `${CLAUDE_PLUGIN_ROOT}/bin/compass`.

**Constraints:**
- PreToolUse hooks block the tool call if they crash or time out, so every path exits cleanly.
- The budget is under 50 ms, with no network and no model on the hot path. Matching is path globs
  against a cached index.

## The closed loop: no human gate
Compass runs without a human in the loop. The human sees and steers it but never has to approve:
- **Visible:** compass's changes land in commits or PRs; current-state notes what ran.
- **Steerable:** "forget X" or "that's outdated", said to the agent, becomes a write through compass.
- **Reversible:** every change is a commit; originals stay in git history.

### No manual wrap-up
In v2, the loop depended on someone remembering to run `wrap-up`, and in ebay-flipper it ran only
when context hit the limit. In v3 there is no wrap-up command to remember. Its work splits by kind:

| Wrap-up's job today | v3 | Kind |
|---|---|---|
| Regenerate INDEX.md, mark superseded docs, stamp anchors | `compass index` / `check --fix` in the pre-commit hook | deterministic, runs on every commit |
| Record a decision or experiment | Written **when it happens**, through MCP `record`. The `AGENTS.md` rule says so, and the hook nudges after a design change lands | judgment, in the flow of work |
| Update `current-state.md` | Triggered at natural boundaries: after a commit, after a PR is opened (`gh pr create` seen by PostToolUse), and before compaction | judgment, at the boundary |
| Make sure it actually happened | **Stop hook:** if the branch has commits newer than current-state's `Commit:` anchor, ending the turn is blocked once with "update current-state first" (guarded by `stop_hook_active`, so it can't loop). **CI:** `compass check` fails a PR that changes code but leaves current-state's anchor stale | enforced in code |

A prompt asks for the update, and code verifies it was done. `wrap-up` survives only as an optional
manual command for an off-cycle catch-up.

Code enforces what review used to:
- **Merges never delete.** A "same lesson" verdict appends the new detail to the existing entry. A
  wrong merge costs length, never information.
- **Entries have a size ceiling.** An entry past it fails `compass check` until the agent restates
  it (keeping every fact) or splits it, so appends can't grow without bound.
- **Provenance comes from git, not the agent.** The commit that added a doc is derived from history,
  so it can't be invented. Optional `refs:` are validated when present and never required.
- **No bulk consolidation passes.** The dedup experiment found unsupervised Haiku consolidation
  merged distinct lessons 18 out of 18 times.
- **Routing outcomes are checked.** When learnings route out to rules, comments or decisions, a
  manifest maps every original to an outcome, and `compass check` rejects an incomplete one.
- **A rule clarity check runs at record time.** A cheap model gets only the injected line and must be
  able to state what was decided and what it rests on, or the line is rewritten.
- **Over-firing guards:** a glob matching more than ~25% of tracked files is flagged.

## Measures
- **Under-feeding:** violations prevented in the trigger eval (the headline metric).
- **Over-feeding:** injected tokens per session, with a median target under ~2k, and the share a
  judge rates irrelevant to the change, with a target under 20%.
- **Resume cost:** tokens read at session start, reported in warm-up's health line.
- **Coverage, measured before delivery is built:** the share of edits in real history that touch a
  path any decision constrains, and the Bash parse-skip rate.

## What compass doesn't do
Task tracking, code summaries, shrinking tool output (context-mode does that), code-structure maps
(graphify or an LSP does that), and restating git history.

## Integrating with ML tooling (future, not designed yet)
Compass stays the interpretation layer and never becomes a tracker:
- **Git** owns which code and config produced a run.
- **MLflow** (or W&B) owns the run: params, metrics, artifacts.
- **Compass** owns what the runs mean and what was decided because of them.

An experiment doc links to runs through `refs:` (`mlflow:run/…`) instead of copying metrics, and an
agent that needs the numbers composes compass with that tool's own MCP server.

Reproducibility guards, such as refusing to train from a dirty working tree, belong in the training
entry point. That covers every run, human or agent. Compass's role is to make sure the convention
exists (an `AGENTS.md` rule from `compass init`) and, at most, to warn when an experiment's evidence
lives in a tracker it doesn't reference.

## Portability: a scope decision
Compass is **portable memory with Claude-Code-first delivery**:
- **The baseline, which works in any agent:** the `.compass/` format, the CLI, the MCP
  `context(path)` and `search` tools, and an `AGENTS.md` instruction to call `context(path)` before
  editing.
- **Hooks are an accelerator.** Push delivery is verified only for Claude Code. Elsewhere delivery is
  pull, with the weakness this doc names: the agent can decide not to ask. Codex and Cursor hooks
  get added as their formats are verified.

## Open questions
- Whether `constrains:` path globs capture "vicinity" well enough. In ebay-flipper, 137 of 181
  learnings referenced no `internal/` path. The coverage measure above answers this before delivery
  is built; `scope: global` covers cross-cutting decisions.

## Appendix: critique dispositions (2026-10-21, fresh-context Sonnet review)
1. **No detector for a wrong routing or an invented premise (high): partly accepted.** Added a
   per-entry size ceiling and provenance derived from git, not hand-written by the agent. **Rejected** the proposed human/CI
   approval gate for high-reversal-cost decisions and status changes. No human gate is a product
   constraint. Supersession already goes through the agent's normal design conversation and PR.
2. **The eval is circular (high): accepted.** Globs come only from history before each incident;
   incidents are held out; the pass/fail bar is written down before the run.
3. **Path globs over- and under-match (high): accepted.** Shared files are excluded from
   git-derived defaults, `scope: global` was added, and coverage is measured before building.
4. **Grep fan-out and compaction re-injection blow the budget (med): accepted.** Grep and Glob no
   longer trigger; there's a per-session budget with a pointer fallback.
5. **Deny-once wastes turns and the ledger is unsafe under concurrency (med): accepted.** Deny is
   off by default and opt-in; ledger writes are atomic appends.
6. **Delivery is Claude-only (med): accepted** as an explicit scope decision. MCP plus AGENTS.md is
   the baseline and hooks are the accelerator.
7. **Silent Bash skips can't be measured (med): accepted.** Skips are logged and measured.
8. **Over-engineered before the eval (low-med): accepted** for build order: the front door ships
   first, and the delivery fields are required only once delivery ships. **Not accepted** for the
   language choice. Go is justified by install and footprint (decision 0010), independent of
   delivery.
9. **Premise matching is fuzzy (low): accepted.** Goal tags make pivot detection an exact match.
