---
claim: "gated: file/symbol, prompt and caller triggers for decision delivery; four-arm trigger eval with design/held-out sets"
---
# Compass v3: decision delivery (gated plan)

Date: 2026-10-21
Status: live, **gated**. Nothing here is built until the core in [positioning.md](positioning.md)
ships and the trigger eval below clears its pre-registered bar. If it fails, this plan is cut and
compass ships as the core alone.

## Why a scaling layer
**Scaling layer: decision delivery.** A single state file is short-term memory. As a project ages it
either grows into an unreadable index (v2's 103 KB INDEX.md) or drops the "why" exactly when a later
agent needs it, and it never reaches a subagent that opens `internal/db/` three hours into a task.
Decisions are long-term memory, delivered by the code they concern. A project with no decisions pays
nothing; a 650-PR codebase gets the shoulder tap.

**The scaling layer has to earn its place. Build order follows from that:**
1. Ship the core ([positioning.md](positioning.md)).
2. Run the trigger eval.
3. Build delivery only if the eval clears its bar.

The delivery frontmatter (below) becomes required only when delivery ships.

**The trigger eval:**
- **Cases:** real incidents (reverted PRs, superseded decisions, a subagent undoing an architecture).
  Each is replayed in four arms:
  1. current-state alone;
  2. current-state plus heavy why-comments, the same decision text written into the constrained
     files;
  3. current-state plus delivery;
  4. a decaying log: the one-line summaries of the last N recorded entries, by recency, with the
     full text queryable (N set so its token cost matches arm 3's median).

  If arm 2 matches arm 3, the answer is the comments rule and no delivery layer. If arm 4 matches
  arm 3, the answer is a decaying log and no triggers. "Matches" is pre-registered: an equivalence
  margin and a fixed number of replays per arm, since replays are nondeterministic and incidents few.
- **Two incident sets:** a **design set** and a **held-out set**, split before anything runs. The
  coverage table below is built on the design set and picks which triggers get built; the pass/fail
  bar is judged only on the held-out set, so the same incidents never both shape and grade the design.
- **Per-incident trigger coverage (design set):** for each incident, which trigger would have fired
  before the violating action: file path, symbol (and whether the read was partial), caller (one
  hop, from a call graph computed offline for the eval), prompt match, or none. Also the **gap**,
  counted in recorded entries (arm 4's unit): how many entries were written between the last one
  that stated the violated constraint and the incident. A gap usually inside the window means
  context stays hot and decay suffices; long gaps are the case for delivery. Where no transcript
  exists, the gap is marked unknown rather than guessed.
- **Prompt precision and recall are pre-registered too:** the prompt trigger ships only if it clears
  a stated bar on both, since a noisy prompt trigger trains the agent to ignore compass lines.
- **No leakage:** a decision's `constrains:` globs are derived only from history *before* the
  incident, and incidents are held out of any tuning.
- **Pass/fail bar, written down before it runs:** delivery must prevent at least a pre-registered
  share of the violations that current-state alone misses (set when the incident set is built,
  before results).
- **If it fails:** delivery is cut, and compass ships as a current-state maintainer.

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
  fires on what was actually read: the symbol whose span overlaps the range, or the file when the
  decision has no symbol anchor.

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

Frontmatter that makes this work. `compass check` and `compass record` enforce it:
- `claim:` one line, at most about 200 characters, never "see doc". A decision's claim is the
  decision; an experiment's claim is its result.
- `premises:` the assumptions it rests on.
- `constrains:` anchors: a path glob (`internal/db/**`) or a symbol (`api.py::create_order`,
  `internal/db/store.go::Store.Open`). Never line numbers, which move on every edit. Path globs
  default from git: the files changed in the PR that added the doc, minus shared files (lockfiles,
  `go.mod`, entry points, generated code). Symbols are set by the agent at record time when the
  decision is about one function or type; when present they narrow the path match.
- `scope: global`, for cross-cutting decisions with no natural anchor. These are injected at session
  start as a short, capped list. Most negative decisions ("we don't use Redis") are better served by
  the prompt trigger below; `global` is for the few that apply to nearly every task.
- `triggers:` a few task-level terms written by the agent at record time, for the prompt trigger:
  the words a future task would use, not the words of the decision ("caching, cache, memoize" on
  "we don't use Redis"). Lexical matching on the claim alone misses exactly these cases.
- `reversal_cost: low | high`
- `refs:` *optional*, typed links to systems that own the underlying facts: `commit:abc123`,
  `pr:123`, `issue:45`, `mlflow:run/7f3e…`. Include one only when a real reference exists. A doc
  whose evidence is just the code carries no `refs:`, and nothing is ever invented to fill the field.
  `compass check` validates refs that are present (a commit must exist) but never requires one.
- `status:` live, superseded or historical, **derived** by `compass index` from later records'
  `supersedes:` links (historical is the one value set at record time). Only live docs are
  delivered.

Provenance doesn't depend on the agent: the commit that added a doc is read from `git log`, never
written by hand.

**What triggers delivery.** Three triggers, all deterministic and cheap. Which ones get built is
decided by the per-incident coverage table from the trigger eval, not up front:

| Trigger | Fires on | Catches | Mechanism |
|---|---|---|---|
| **Code (file or symbol)** | Read or Edit of a constrained path. For a symbol anchor: a **partial** Read (`offset`/`limit`) or an Edit whose `old_string` overlaps the symbol's current span gets the full tier; a **full-file** Read (the default) gets only a one-line pointer for symbol-anchored docs; Write, MultiEdit and `sed -i` fall back to the file | Decisions about specific code. Most reads are full-file, so the symbol anchor's real precision is at the Edit tier | Path glob match, then a span lookup in a cached symbol table |
| **Prompt** | UserPromptSubmit for the main agent. For a subagent, PreToolUse on the Agent/Task tool, appending to the task prompt (**unverified: on the spike list**; UserPromptSubmit doesn't fire for subagents) | Negative and cross-cutting decisions, before any file is opened; the planning moment, which file triggers reach too late | FTS5 over claims and `triggers:`; fires on a **matched-term count** (e.g. ≥2 distinct terms), not a raw BM25 score, which isn't comparable across queries or corpus sizes. Prompts with fewer than ~3 content terms ("continue", "fix it") and pasted blocks over a size limit are skipped |
| **Caller** *(build only if the eval shows misses)* | Edit of a direct caller or callee of a symbol-anchored decision | Breaking a function's contract from another file | One hop in a call graph (graphify's if installed, else none). For the eval the graph is computed offline, so coverage can be scored regardless |

The symbol table is built by `compass index` and cached per file keyed by content hash, so a file
edited mid-session is re-parsed on its next lookup. It starts with universal-ctags when installed
(top-level functions, types and methods with their line spans) and falls back to the file when it
isn't; in-binary parsers (cgo tree-sitter, a bigger binary) are added only if the fallback rate says
so. A symbol that no longer resolves
fails `compass check`: a rename makes a stale anchor visible instead of silent. If a symbol can't be
resolved at hook time, the trigger falls back to the file and logs the fallback.

**How much is injected, escalating with proximity to a change:**

| Agent action | Injected | Size |
|---|---|---|
| Prompt matches a claim | claim + doc path | ~40 tokens |
| Reads a constrained path or symbol | claim + doc path | ~40 tokens |
| Edits it | claim + premises + rejected alternatives | ~120 tokens |
| First edit in a context to a path whose doc has `reversal_cost: high` | **Off by default (opt-in per project).** PreToolUse denies once: "read decisions/0012's premises first; superseding is fine if they no longer hold" | full doc read |

Grep and Glob results don't trigger injection. A search returns many paths, and the agent reading a
file is the real signal.

**Caps:**
- At most 3 entries and about 300 tokens per injection, with the most specific glob first.
- A **per-session budget per trigger** (about 2k tokens for code, a smaller one for prompt), so
  early prompt matches can't starve the code triggers that fire next to an edit. Once a budget is
  spent, its injections shrink to a one-line pointer ("compass: 3 more decisions apply to
  internal/db/; call context(path)").

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

**Still to verify in a spike, before the trigger that depends on it is built:**
- UserPromptSubmit `additionalContext` reaches the model (prompt trigger, main agent).
- PreToolUse on the Agent/Task tool can append to a subagent's task prompt (prompt trigger,
  subagents).

**Constraints:**
- PreToolUse hooks block the tool call if they crash or time out, so every path exits cleanly.
- The budget is under 50 ms, with no network and no model on the hot path. Matching is path globs,
  symbol spans and an FTS query against a cached index.

## Measures
- **Under-feeding:** violations prevented in the trigger eval (the headline metric).
- **Over-feeding:** injected tokens per session, with a median target under ~2k, and the share a
  judge rates irrelevant to the change, with a target under 20%.
- **Coverage, measured before delivery is built:** the share of edits in real history that touch a
  path any decision constrains, the per-trigger table from the trigger eval, the Bash parse-skip
  and symbol-fallback rates, and the full-file share of Reads.
- **Flag fatigue:** the share of churn flags dismissed without a superseding record.
- **Over-firing guard:** a glob matching more than ~25% of tracked files is flagged by `compass check`.
- **Rule clarity check at record time:** a cheap model gets only the injected line and must be able
  to state what was decided and what it rests on, or the line is rewritten.

## Open questions
- Whether code anchors capture "vicinity" well enough. In ebay-flipper, 137 of 181 learnings
  referenced no `internal/` path, which suggests the prompt trigger may carry more load than the
  code trigger. The per-incident coverage table answers this before delivery is built.
- Whether a lexical prompt match is precise enough even with `triggers:` terms, or prompts and claims
  differ often enough to need embeddings (which decision 0011 dropped for dedup). The pre-registered
  prompt precision/recall bar answers it.
- How often reads are full-file. If nearly always, the symbol anchor only sharpens the Edit tier.
- Whether MCP `context(path)` / `search` tools are worth adding for agents without hooks, or the
  `AGENTS.md` instruction plus `rg` is enough.

## Appendix: delivery-redesign critique dispositions (2026-10-21, fresh-context review)
Applied to the staleness section, the four-arm eval and the three triggers. All accepted:
1. **Full-file reads defeat the symbol anchor (high):** a full Read gets only a pointer for
   symbol-anchored docs; symbol precision lives at the Edit tier; the full-read share is measured.
2. **Lexical prompt match misses "add caching" vs "no Redis" (high):** a `triggers:` field of
   task-level terms, firing on matched-term count, short and pasted prompts skipped, precision and
   recall pre-registered.
3. **Subagent prompt delivery unverified (high):** moved to PreToolUse on the Agent tool and onto
   the spike list.
4. **The coverage table and the pass/fail bar used the same incidents (high):** design and held-out
   sets.
5. **"Matches" undefined (high):** equivalence margin and replay count pre-registered.
6. **Gap measured in time vs a window in entries (med):** gap counted in entries; unknown without a
   transcript.
7. **Status edits contradict immutability (med):** status is derived from `supersedes:`; current-state
   checks that files in play exist, and the doc says plainly that overwriting proves recency, not
   truth.
8. **Premises rot with no event; goals lived in a banned doc (med):** goals are dated records now;
   the out-of-repo premise case is named and left to the closed loop.
9. **One budget for all triggers (med):** per-trigger budgets.
10. **Index stale between record and commit (med):** rebuilt on `record`, hash-checked by the hook.
11. **Parser cost (low):** universal-ctags first, file fallback, in-binary parsers only if needed.
12. **Caller trigger needs a graph (low):** computed offline for the eval.
13. **Flag fatigue (low):** dismissal rate measured.
