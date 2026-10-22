---
claim: "portability (§1) still holds; retrieval B–E deferred, hygiene replaced, session economics moved out"
---
# Compass v3: direction and open options

Date: 2026-10-15
Status: **superseded except §1** (2026-10-21) by [positioning.md](positioning.md) and
[delivery-plan.md](delivery-plan.md). §1 (portability) still holds. §2 retrieval options B–E are
deferred until a retrieval eval shows need; §3 is replaced by positioning.md's staleness rule; §4
(session economics) is session policy and moves to a separate tool. Kept as the record of what was
considered. Evidence: [2026-10-15-ebay-flipper-retro.md](2026-10-15-ebay-flipper-retro.md).

`.compass/` is read and written by agents, not people. Its design should be judged by one thing:
how cheaply an agent gets the right context, and how reliably the corpus stays correct as it grows.
The first long-running project showed where v2 breaks: an index too large to read in one call,
guards that never ran outside this repo, and most write-only docs never retrieved again.

## 1. Portability: compass is one CLI, then gets out of the way
Hard requirement. Installing compass on a machine must be one command, and after that nobody thinks
about it.

What went wrong: `wrap-up` calls `scripts/compass-check.sh` and `uv run tools/learning-dedup/...` by
repo-relative path. Those exist only inside this repo, and `install.sh` doesn't ship them, so in
every other project the "mandatory" check silently never ran. The same trap that 0007 fixed for
`init` still holds for everything else.

Requirements:
- **One install:** e.g. `uv tool install` from git, or one static binary. It brings the skills, the
  hooks and every deterministic tool. No clone to keep around, no `$DOTFILES`, no per-repo vendoring.
- **Every deterministic step is a subcommand.** `compass init`, `compass check [--fix]`,
  `compass index` (generate INDEX.md), `compass dedup`, `compass stats` (today's
  `tools/usage-retro/retro.py`), `compass doctor`. Skills and hooks call `compass <subcommand>`, never
  a path.
- **Per-repo wiring is one command:** `compass init` also wires `compass check` into the repo's
  pre-commit or lefthook config and prints the one CI line (`uvx … compass check`), so the guard
  runs in the project itself.
- **Self-checking:** `compass doctor` verifies the installed skills, hooks and CLI agree on the spec
  version; warm-up's health line uses it.
- **Updates:** re-running the install updates everything, idempotently, as today.

## 2. Retrieval architecture: options
> Superseded: option A (capped generated index) is the core; B–E deferred.

Markdown in git has real strengths for this job: it diffs, it is reviewable in a PR, any agent can
read it, and it needs no server. What it lacks is a cheap way to ask "which few docs matter for this
task" without reading a catalogue. Options, cheapest first:

**A. Frontmatter plus a generated, capped index.** Every doc carries `type`, `status`
(live / implemented / superseded / historical), `date`, `summary` and `supersedes`. `compass index`
generates INDEX.md from it with a hard token cap: start-here, the live plans, then pointers to
per-category listings. Folders become optional; structure lives in metadata. Agents search with
`rg` filtered by `status: live`. Smallest change; probably enough at ~2 MB of text.

**B. A derived SQLite index (FTS5), markdown stays the source of truth.** `compass index` also builds
a gitignored `.compass/.index.db`: one row per doc with its frontmatter, and an FTS5 table over body
text. `compass search "<question>" --status live --type decision -k 5` returns ranked paths and
summaries. Rebuilt from files in milliseconds, so it never drifts and needs no backup. SQLite is in
the Python stdlib, so it costs no dependency.

**C. Embeddings on top of B.** Store chunk vectors (`sqlite-vec`, or plain numpy for this size) for
semantic search and for dedup, which today runs a separate embedder. Worth it only if it beats FTS5
on a retrieval eval (below); it adds a model download and makes results harder to predict.

**D. Expose B/C to agents as tools.** An MCP server or Claude Code skill with `compass_search`,
`compass_get`, `compass_record`, so an agent asks for context instead of reading an index. Warm-up
becomes "current-state plus the top 5 hits for the task". Most agent-native; most moving parts.

**E. Let the agent shape the structure from a template.** Instead of fixed folders, `init` gives a
template and the agent fits the structure to the project. With A's frontmatter, the folders matter
less. Keep a small fixed core that the tools depend on (index, current-state, decisions, plans);
everything else is free-form with frontmatter.

**How to choose: measure, don't guess.** Build a retrieval eval from real use: questions agents
actually asked in ebay-flipper sessions, each with the doc(s) that answered it. Score A, B and C on
recall@5 and tokens read per question. Adopt the cheapest one that clears the bar. The transcripts
already hold the questions and the docs that were then read.

Delivery triggers (file/symbol, prompt, caller) and the four-arm trigger eval, including a
decaying-log arm, are specified in [positioning.md](positioning.md); its staleness rule (nothing is
maintained in place) applies to every option here.

## 3. Corpus hygiene (any option)
> Superseded by positioning.md's staleness rule and `compass check`.

- **Cap the index by tokens, not rows,** enforced by `compass check` (target ~3k tokens).
- **`status` on every doc,** so warm-up and search skip dead material (~60% of the ebay-flipper
  corpus).
- **Drop or shrink logs.** The PR description and commit messages already carry them. Keep the
  `Commits:` anchor on current-state instead.
- **Learnings become rules or merge into the doc they affect.** A learning that never changes
  behaviour is a diary entry. Prefer: a CLAUDE.md rule, a lint check, a line in the relevant plan, or
  one `learnings.md` per area.
- **Split plans:** contract on top and short; critique and break dispositions in an appendix the
  reader can skip; as-built notes replace the parts they change.
- **Cap `current-state.md`** (e.g. 80 lines); stale next-steps move to a backlog doc.

## 4. Session economics: make short sessions the default, not a discipline
> Superseded: session policy, moved out of compass to a separate tool.

**Why sessions reached 1M (Emilio, 2026-10-15).** Wrap-up was not part of the PR process. A session
would take a problem through the work loop, open a PR, then start the next problem in the same
context, about 3 PRs' worth, and run wrap-up only when the context hit the limit. The long tail
was a process shape, not forgetfulness. Relying on the operator to end sessions early does not work;
the design has to make the short session the easy path and a session past ~500k hard to reach.

The unit is **one problem → one PR → one session**:
- **Wrap-up ships inside the PR** (ebay-flipper CLAUDE.md step 6, 2026-10-15). Once the PR is open
  and its docs are on the branch, the session's job is done. Nothing is left that needs that context.
- **The next problem starts a new session,** not a new chapter. Claude Code's sub-sessions (seen
  2026-10-15) make this cheap: the current session starts a fresh session in its own worktree, with
  a brief, and stays short itself. The same mechanism runs independent problems in parallel, each
  in a focused PR with its own short context.
- **The handoff is the brief, not the transcript.** A fresh session gets `current-state.md`, the
  plan, and a written brief: problem, files, what done looks like. If a brief can't be written, the
  work isn't scoped well enough for a PR yet.

Guard rails, in the hooks:
- **Nudge on absolute tokens, repeating:** soft at ~250k ("this PR should be close to done"), firm at
  ~400k ("wrap up into the PR and hand off"). Not once at a percentage of an assumed 200k window.
- **Hard ceiling at ~500k:** past it, the hook blocks new feature work until wrap-up is done and the
  next step is handed to a fresh session. It should still allow wrap-up and the handoff itself.
- **Track it:** `compass stats` reports the context reached per merged PR and the share of cost above
  300k, so a drift back toward long sessions shows up in the numbers.
- Warm-up should state its own cost (tokens read) in the health line, so a growing corpus is visible.
