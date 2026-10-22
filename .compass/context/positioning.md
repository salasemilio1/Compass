---
claim: "v3 goal (disposable sessions, cross-agent resume), staleness rule (nothing maintained in place, expiry by event), and the core: check/index/record/resume + AGENTS.md block + hooks"
---
# Compass v3: positioning and core

Date: 2026-10-21
Status: live, a proposal under review. It fed decisions 0010 (Go + plugin) and 0011 (dedup), both since
superseded; 0015 cut learnings. The gated scaling layer is in [delivery-plan.md](delivery-plan.md).
Evidence: [2026-10-15-ebay-flipper-retro.md](2026-10-15-ebay-flipper-retro.md),
`experiments/2026-10-21-dedup-architectures.md` (branch `claude/dedup-experiment-436ce8`), and the
plugin hook spike recorded in delivery-plan.md.

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
| `CLAUDE.md` / `AGENTS.md` | Rules every session follows | Writes only its managed block; a learning that is a rule goes here, by hand |
| Issue tracker | Tasks and backlog | None. current-state holds only "now and next" |
| **Colleagues** | **Where we are, why, and what matters for what you're about to touch** | **This is compass** |

**Litmus test for every write:** could a fresh agent derive this from the code plus `git log`
cheaply? If yes, compass doesn't store it. It points to it.

**What compass holds:** current state (last known intent), decisions (with their premises), and
experiment results; plans live with the task (a PR, an issue, a scratch file). Learnings route out: to a rule, to a comment, or merged into the decision they affect.

## Staleness: nothing is maintained in place
Any document someone has to keep current goes stale; READMEs and specs do, and so did this repo's
hand-kept INDEX line for current-state. Compass's answer is not better upkeep. It removes upkeep.
Every compass doc is one of three kinds:

| Kind | Examples | Why it can't silently rot |
|---|---|---|
| **Generated** | INDEX.md, symbol tables, the search index, every doc's `status` | Rebuilt from source on every commit and on every `record`; the hook checks the index hash, so a decision recorded but not yet committed is still delivered |
| **Overwritten whole** | `current-state.md` | Rewritten at each boundary, never patched; its `Commit:` anchor is checked |
| **Immutable dated record** | decisions, goals, experiment results | States what was decided or found *on a date*. History stays true; only its applicability changes. A record's text is never edited: `status` is derived from the `supersedes:` links of later records, not written into the old one |

Overwriting is not a guarantee of truth: an agent can rewrite current-state from a stale copy, and the
`Commit:` anchor proves recency, not correctness. The checks catch the mechanical part (anchor, paths
that exist); the rest is why current-state stays small.

A present-tense doc maintained by hand (an architecture overview, a hand-written spec) is not
allowed. If it can't be generated, it is a dated record or it doesn't exist.

**Expiry is by event, not by age.** A decaying log (keep the last N entries, let the rest fall away)
is a sound design when two assumptions hold: context goes stale once written, and anything still
relevant keeps getting rewritten. The second holds for **hot context**, the work in progress, and
compass's front door is exactly that model with a window of one. It fails for **cold-but-critical
context**: a constraint that matters precisely because no one has touched that code lately, so no
one re-logs it, so it decays just before the agent that needs it arrives. Age doesn't track truth: a
three-day-old note can be wrong and a month-one decision can still hold. A record leaves the
delivered set when an event says it no longer applies:
- a later decision `supersedes:` it;
- `compass check` finds its anchor gone (a constrained symbol renamed or deleted);
- `compass check` finds heavy churn on its constrained paths since the record's commit and flags it
  for review in the next change that touches them (a flag, not an expiry; the dismissal rate is
  tracked so flag fatigue shows up);
- a goal its premises cite is superseded.

Premises can also rot with no event in the repo (a vendor changes terms, a requirement is dropped in
a meeting). Nothing mechanical catches that; the agent or the human records a superseding decision
when they learn it, which is the closed loop's "that's outdated" path.

Whether cold-but-critical context is common enough to justify this is an empirical question the
trigger eval in [delivery-plan.md](delivery-plan.md) answers (arm 4).

## The core: what ships first
The goal is a **disposable session**: an agent, Claude or not, starts informed without a discovery
phase, and work resumes on a different agent when one hits its usage limit. That alone is enough
value. Everything in this doc serves it; the scaling layer in
[delivery-plan.md](delivery-plan.md) is built only if an eval says it's needed.

**Front door: `current-state.md`.** It's capped (about 80 lines) and holds last known intent: the
objective and the next steps, never a plan; git and the user's current prompt win over it. It's overwritten at each boundary, with its
`Commit:` anchor checked. What loads at session start is **recent state plus live long-term memory**:
current-state (the hot window) and the capped list of live `scope: global` claims, regardless of
their age. Recency picks the state; status picks the decisions. On any project, even an afternoon
one, the value is visible in the first session. It's also the evidence-backed core: in ebay-flipper
it was the most re-read file, by 22 later sessions.

**The resume brief: `compass resume`.** Usage limits hit mid-task, before any commit or Stop hook.
`compass resume` prints what a fresh agent needs and nothing it can derive itself: current-state,
live global claims, and `git status` plus a capped `git diff --stat` of uncommitted work (the code is
the truth; current-state says why). Any agent can run it; the Claude SessionStart hook runs it
automatically.

**The core, in full:**
1. **The format:** `current-state.md`, decisions, goals and experiments, each with frontmatter
   (`claim`, `premises`, `supersedes`, `refs`, `scope`).
2. **`compass check`:** caps, the `Commit:` anchor, next steps written, every doc has a `claim`,
   status derived from `supersedes:`, refs that are present resolve. Runs in pre-commit and CI.
3. **`compass index`:** generates INDEX.md under a token cap.
4. **`compass record` and `compass resume`:** write a dated record; print the resume brief.
5. **An `AGENTS.md` managed block:** read the resume brief first, rewrite current-state at
   boundaries and before long steps, record decisions when they happen, comment the why. Compass
   writes only this block and a pointer into files the user owns, never learnings into `CLAUDE.md`.
6. **Claude Code accelerators (plugin):** SessionStart runs `compass resume`; PreCompact and Stop
   check the anchor; CI runs `compass check`.

Ship that, measure resume cost and cross-agent resume, then decide on delivery.

## Decisions are the current best answer, not law
Projects change: ebay-flipper began as a personal dashboard. Compass prevents **uninformed**
reversals and encourages **informed** ones.
- A decision records `decision:` (what holds now, stated as fact, not as an order) and `premises:`
  (the assumptions it rests on). An agent challenging a decision checks the premises, not the
  conclusion. Premises may reference a project goal by tag (`goal:personal-dashboard`). Goals are
  dated records like decisions (not a hand-maintained list in `context/overview.md`, which the
  staleness rule forbids); retiring a goal means recording one that supersedes it.
- **Superseding is cheap.** The agent proposes the change like any design change. If adopted, a new
  decision carries `supersedes:`; the old one's derived status becomes superseded and it stops
  being delivered. Its text is untouched.
- **Pivots surface their casualties.** When a goal is superseded,
  `compass check` lists every live decision whose premises reference that tag. The match is exact,
  not a fuzzy guess at prose.
- The `AGENTS.md` block states it: *decisions are the current best answer; if you see a better
  approach, propose it and address the premises.*

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
| Regenerate INDEX.md, derive status from `supersedes:`, stamp anchors | `compass index` / `check --fix` in the pre-commit hook | deterministic, runs on every commit |
| Record a decision or experiment | Written **when it happens**, through `compass record` (CLI, so any agent can call it; an MCP wrapper is optional). The `AGENTS.md` rule says so, and the hook nudges after a design change lands | judgment, in the flow of work |
| Update `current-state.md` | Triggered at natural boundaries: after a commit, after a PR is opened (`gh pr create` seen by PostToolUse), before compaction (PreCompact), and **before any long step** (the `AGENTS.md` rule), because usage limits hit mid-task, not at commits | judgment, at the boundary |
| Make sure it actually happened | **Stop hook:** if the branch has commits newer than current-state's `Commit:` anchor, ending the turn is blocked once with "update current-state first" (guarded by `stop_hook_active`, so it can't loop). **CI:** `compass check` fails a PR that changes code but leaves current-state's anchor stale | enforced in code |

A prompt asks for the update, and code verifies it was done. `wrap-up` survives only as an optional
manual command for an off-cycle catch-up.

Code enforces what review used to:
- **Entries have a size ceiling.** An entry past it fails `compass check` until the agent restates
  it (keeping every fact) or splits it, so appends can't grow without bound.
- **Provenance comes from git, not the agent.** The commit that added a doc is derived from history,
  so it can't be invented. Optional `refs:` are validated when present and never required.
- **Learnings route out, so there is nothing to dedup.** A learning becomes a rule, a comment, or
  part of the decision it affects; there is no learnings category (decision 0015 cut it, so 0011's
  dedup is moot; the dedup experiment had shrunk the corpus only 5%).

## Measures
compass claims **continuity**, not discovery: a fresh or different agent picks up where the last
one stopped. It does not claim fewer reads on a task the prompt already specifies (the core eval
found none, n=2). The measures are taken by evals recorded in `experiments/`, not by the binary:
there is no `stats` command and no status-line counter.
- **Resume cost (headline for the core):** tokens of the brief a fresh agent gets before its first
  useful action (the core eval: 93% smaller than v2's front door).
- **Cross-agent resume:** a task interrupted mid-step in one agent and resumed in another from
  `compass resume` alone; scored on whether the second agent continues without redoing discovery.
  A cut-off session leaves current-state as it was; the brief's `git status` and diff carry the
  unfinished work, so the measure counts what those carried.
- **Unspecified continuation:** on a "continue" prompt that names no task, whether the agent finds
  the live next step from the stub or the full brief (pending).
- **Discovery overhead:** Read/Grep/Glob calls before the first edit; tracked as a guard against
  compass adding cost, not as a claimed saving.
- Delivery's measures (under- and over-feeding, coverage, flag fatigue) live in
  [delivery-plan.md](delivery-plan.md).

## Scope: what a compass does (2026-10-22)
Compass is a compass, not a map or a guide. It tells an agent **where the project is, why it is
shaped this way, and which way is next**, then leaves it to navigate the code itself. It orients;
it does not walk the route. When the agent reaches ground where a past choice matters, it supplies
the why and the why-not, and nothing else.

Every question about a codebase already has an owner. Compass owns only the ones nobody else does:

| Question | Owner | Tense |
|---|---|---|
| Who | Git: author, co-authors, PR reviewers | past |
| When | Git (compass records take their dates from it) | past |
| Where | The code tree; git for which paths changed. "What calls this" is a view of the code (graphify, LSP), not a store | present |
| What it does | Code | present |
| What it should do | Tests | present |
| What changed | Git | past |
| How | Code; how to use it, the README | present |
| Why this line | Comments | present |
| Why this change | Commit message, PR | past |
| **Why the system is shaped this way** | **Compass decisions**: reasons that span many commits, which no single commit owns | standing |
| **Why not** | **Compass decisions**: rejected alternatives; code shows only the path taken | standing |
| **Where we are, what's next** | **Compass current-state** | future |

Git can answer every question for one change, in the past tense, scattered through history; it
can't answer what holds across commits or where the work is going. That is compass's whole remit.

**What follows from the scope** (the core eval's gaps, by owner):
- **A specified task ignores compass.** Correct: the prompt already says where to go. The
  SessionStart stub (≤500 chars) stays out of the way unless the session resumes or has
  unfinished work.
- **Discovery didn't get cheaper.** Finding code is a where/what question, owned by the code, `rg`,
  an LSP or graphify. Compass's part is the why-not at the right file, which is
  [delivery-plan.md](delivery-plan.md) and stays gated on its eval.
- **Handoff cost.** Mostly the receiving agent's own exploration; compass's part is a short brief
  that says what not to re-read.
- **A session cut off mid-step.** Progress lives in git (small commits whose messages say what and
  why, plus the uncommitted diff). current-state is **last known intent**: the objective and the
  next step, never a step-by-step plan, which would bloat it and go stale the moment the work
  changes direction. A plan is task tracking and lives with the task (a PR description, an issue, a
  scratch file). Where current-state and git disagree, git and the user's current prompt win, and
  the disagreement itself tells the next agent the work drifted.

## What compass doesn't do
Discovery of code (see the scope table), task tracking, code summaries, shrinking tool output (context-mode does that), code-structure maps
(graphify or an LSP does that), restating git history, and **session policy**: token nudges, hard
context ceilings and cost stats belong to a separate tool, not memory.

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
Compass is **portable memory with Claude-Code-first acceleration**:
- **The baseline, which works in any agent:** the `.compass/` format, the `compass` binary
  (`check`, `index`, `record`, `resume`), and the `AGENTS.md` block telling the agent to run
  `compass resume` first.
- **Distribution matches that:** the same binary ships through GitHub releases and `go install`, not
  only inside the Claude Code plugin, so Codex, Cursor and CI get it without Claude.
- **Hooks are an accelerator.** Verified only for Claude Code. Elsewhere the agent runs
  `compass resume` because `AGENTS.md` says to. Codex and Cursor hooks get added as their formats are
  verified.

## Open questions
- How well `compass resume` alone carries a mid-task handoff between agents; the cross-agent resume
  measure answers it.

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
6. **Delivery is Claude-only (med): accepted** as an explicit scope decision (now in delivery-plan.md). The CLI plus AGENTS.md is
   the baseline and hooks are the accelerator.
7. **Silent Bash skips can't be measured (med): accepted.** Skips are logged and measured.
8. **Over-engineered before the eval (low-med): accepted** for build order: the front door ships
   first, and the delivery fields are required only once delivery ships. **Not accepted** for the
   language choice. Go is justified by install and footprint (decision 0010), independent of
   delivery.
9. **Premise matching is fuzzy (low): accepted.** Goal tags make pivot detection an exact match.

## Appendix: whole-plan review dispositions (2026-10-21, fresh-context review)
All accepted:
1. **Cross-agent resume vs plugin-only shipping:** the binary also ships via GitHub releases and
   `go install`.
2. **Usage limits hit mid-task:** current-state refreshed before long steps and before compaction;
   `compass resume` includes uncommitted git state.
3. **`CLAUDE.md` is the user's:** compass writes only a managed block and a pointer.
4. **v3-direction.md contradicted this doc:** marked superseded except §1 (portability).
5. **Session economics is policy, not memory:** moved out to a separate tool.
6. **Retrieval options B–E deferred:** a capped generated index plus `rg` until a retrieval eval
   shows need.
7. **Delivery was ~70% of this doc:** moved to [delivery-plan.md](delivery-plan.md), gated.
8. **Learnings and dedup may be moot:** 0011 made conditional on learnings surviving as a category
   (they didn't: 0015 cut the category).
