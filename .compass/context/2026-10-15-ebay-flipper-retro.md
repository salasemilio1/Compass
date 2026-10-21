---
claim: "first long-run field data: 90% of cost in main sessions, 78% above 300k context; 103 KB INDEX; logs/learnings mostly never re-read; model-by-task grades"
---
# Field retro: compass and the work loop on ebay-flipper (2026-07 to 2026-10)

Date: 2026-10-15
Source: `~/Dev/Misc/personal-dashboard` (ebay-flipper), `main` at `28e8c3d`, 233 commits, 3 months.
Method: `tools/usage-retro/retro.py` over 46 main-session and 471 subagent transcripts (2026-07-31
onward; the repo began 07-13), git history, and two subagent passes: a blind grading of 40
critique/break reports and a test-quality read. Direction this feeds: [v3-direction.md](v3-direction.md).

**Caveats.** Costs are list-price estimates, relative only. "Never re-read" counts tool inputs that
name a file, so reads through `grep` over a folder are missed: it is an upper bound. Model
comparisons are 10 reports per cell, graded once.

## 1. Where the tokens went
- Main sessions were 90% of estimated cost ($3,164); subagents 10% ($341).
- 81% of main-session cost is cache reads: re-reading a long context every turn.
- Main cost by context size at the turn:

  | Context | Turns | Share of cost | $ per turn |
  |---|---|---|---|
  | < 100k | 776 | 2% | 0.08 |
  | 100–300k | 5,110 | 20% | 0.13 |
  | 300–600k | 6,163 | 47% | 0.24 |
  | > 600k | 2,357 | 31% | 0.41 |

- The 12 costliest sessions peaked at 750k–965k tokens. "Sessions are disposable" held in principle,
  not in practice.
- `hooks/session-context-nudge.py` assumes a 200k window (`COMPASS_CONTEXT_WINDOW`) and fires once at
  75%, so on a 1M-window model it fired around 150k, once, and nothing followed it to 965k.

**Root cause (Emilio):** wrap-up was not part of the PR process, so a session carried about 3 PRs'
worth of work and wrapped up only at the context limit.

**Learning:** session length is the cost lever, and it is set by process shape: one problem, one
PR with its docs, one session. The next problem goes to a fresh session; Claude Code's sub-sessions
make that a single step. Tuning which model a subagent uses touches 10%.

## 2. Models by task (blind-graded, unblinded after)
| Task | Model | Real findings | Severe | Median $ | $ per severe |
|---|---|---|---|---|---|
| Plan critique | Opus | 10.0 | 2.6 | 0.93 | ~0.36 |
| Plan critique | Sonnet | 8.1 | 2.2 | 0.48 | ~0.22 |
| Break pass | Opus | 7.3 | 2.5 | 1.71 | ~0.68 |
| Break pass | Sonnet | 7.1 | 1.7 | 1.11 | ~0.65 |

Routing this supports:
- **Haiku:** mechanical work with a fixed output shape (61 runs: labelling batches at ~$0.20,
  `file:line` lookups, one-file edits from an exact spec). No evidence on judgment work.
- **Sonnet:** default for critiques and break passes; cheapest per severe finding on critiques.
- **Opus:** the orchestrating session, and the final break pass on money, migration and concurrency
  code, where its extra severe findings matter more than the price.
- **Untested:** a Sonnet main session implementing an already-vetted plan. Opus sessions averaged
  $81 and Sonnet $30, but on different work.

Strong reports had, per finding, a `file:line`, a concrete scenario and a fix, often reproduced with
a throwaway test, plus a list of what held and what was refuted. Weak ones were surveys or relays of
another report's summary. That output shape belongs in the critique and break prompt templates.

## 3. The .compass/ corpus
| Category | Files | Size | Never re-read by another session |
|---|---|---|---|
| context (plans) | 83 | 1.25 MB, median 14 KB, max 46 KB | 38% |
| decisions | 92 | 0.28 MB | 44% |
| learnings | 172 | 0.26 MB | 56% |
| logs | 93 | 0.40 MB | 62% |
| experiments | 15 | 0.04 MB | 0% |

- **INDEX.md is 103 KB (~25k tokens, 377 rows, median row 259 chars).** That is over the Read tool's
  per-call cap, so warm-up reads half of it. SPEC §4 caps only logs and experiments at 15; learnings,
  decisions and context are uncapped, and those are what grew.
- **The guards never ran.** `scripts/compass-check.sh` and `tools/learning-dedup/` exist only in this
  repo; `install.sh` does not ship them and `wrap-up` calls them by repo-relative path. In
  ebay-flipper neither path exists, so no cap, supersession mark or dedup ever applied.
- **What earned its keep:** `current-state.md` (read by 22 later sessions), the plans (the most-read
  category), experiments (all re-read), and warm-up's cost (~74k tokens at the first task, system
  prompt included).
- **What mostly did not:** logs (they restate commits and PR bodies) and one-file-per-gotcha
  learnings. Learnings paid off when they became a rule (CLAUDE.md, a lint check such as forbidigo)
  or a line in the plan for the code they affect.
- **Plans mix three jobs:** the contract, the critique dispositions and the as-built notes. Readers
  pay for all three.
- **Unused fixed folders:** `architecture/` stayed empty and `environment/` was never created; host
  facts went to Claude memory and `current-state.md`.
- **Docs-only PRs** were 53 of 233 commits and each cost CI minutes. The project now commits wrap-up
  to the feature's own PR.

## 4. The work loop (plan, critique, implement, break)
- Pre-code critique works: 63 of 71 plans carry one; graded reports averaged 7–10 concrete defects
  and 2–2.6 severe ones each.
- What escaped after merge was mostly UI: a router crash on first load (#79) and a page crash on
  config mutes (#99). Break passes read Go and run Go tests. A browser smoke step for UI-touching
  PRs would have caught both.
- Tests follow the spec-first philosophy well (exact-boundary pairs, fake clocks, N-goroutine
  invariants, fuzz on parsers). Gaps: wall-clock sleeps and timing assertions (the known flakes) and
  fuzzing in only two packages. Low value: tests that restate config mappings, and hand-computed
  aggregate struct equality.

## 5. Carry into the next project
1. One problem → one PR (wrap-up included) → one session; the next problem starts a sub-session.
   Hooks nudge at ~250k and ~400k and block new work past ~500k.
2. The model routing above, written into the global CLAUDE.md.
3. A fixed output shape for critique and break prompts (`file:line`, scenario, fix, held/refuted).
4. A browser smoke step in the loop for any UI change.
5. "Inject a clock; never assert on wall time" as a testing rule.
6. Compass v3 per [v3-direction.md](v3-direction.md): a capped, generated index; a `status` on every
   doc; guards shipped in the CLI.
7. Re-run `tools/usage-retro/retro.py` at each project milestone and compare against this table.
