---
claim: "v3 field week on ebay-flipper (2026-10-26 to 10-30): orientation, current-state upkeep, decision reads and overhead from real transcripts, bars set before the week"
date: 2026-10-26
refs:
  - tools/field-retro/retro.py
  - experiments/2026-10-26-claude-signal-search-line.md
---
# v3 field week on ebay-flipper (2026-10-26 to 10-30): orientation, current-state upkeep, decision reads and overhead from real transcripts, bars set before the week

Status of this file: **protocol and bars written before the week** (committed first, so git shows
they predate the data). Results are appended on 2026-10-30 in a later commit.

## Why
v3 has no field data: every result so far is a small eval on a copy or a synthetic repo. This week
Emilio works on ebay-flipper as usual with the v3 plugin installed, and the transcripts are the
data. Three open questions, each with a bar and a kill signal:
1. **Field baseline:** do sessions orient from compass and keep current-state true?
2. **Decision reads:** do agents search the records before changing something a live decision
   covers, and does anything get reversed silently?
3. **Overhead:** is compass's cost on real sessions small enough for what it buys?

## Setup
- Repo: `~/Dev/Misc/personal-dashboard` (and its worktrees), migrated to v3 on 2026-10-26.
- Window: 2026-10-26 to 2026-10-30, Claude Code only (Codex runs, if any, are reported apart).
- Tool: `tools/field-retro/retro.py --repo ~/Dev/Misc/personal-dashboard --since 2026-10-26
  --until 2026-10-31 --flagged flagged.jsonl`. Deterministic; reads transcripts and git only.
- Baseline (same script, v2, 2026-10-12 to 2026-10-26, 15 sessions): brief 0/15, current-state
  rewritten in 0/11 sessions with commits, compass share of tool I/O median 16.8%, 80 flagged
  edits, all after the session had searched the records (v2's warm-up and work loop read them).

## Measures and bars
| # | Measure | Bar | Kill / rethink signal |
|---|---|---|---|
| 1a | Continue-style sessions that oriented (brief or `compass resume`) before the first edit | ≥ 80% | < 50% |
| 1b | Sessions with a commit that rewrote current-state | ≥ 90% | < 60% |
| 2a | Flagged edits (code-like name or dir/file path a live decision names) made after a records search | ≥ 80% | < 50% |
| 2b | Silent reversals among flagged edits (judge, below) | 0 | any one: write it up; two: build `compass find` per decision 0019 |
| 2c | Human catches ("we decided", "why did you change"…) | each read by hand | one caused by a missed record counts as 2b |
| 3a | Compass share of tool I/O, median per session | ≤ 10% | > 16.8% (v2's level) |
| 3b | Benefit events (flagged edits the judge marks *kept, informed* or *superseded, informed*, plus continue sessions that resumed without re-pasting the task) | ≥ 1 per 10 sessions | 0 in the week while 3a fails |

## Judge pass (Friday)
Each flagged edit (`flagged.jsonl`: the edit's old and new text and the decisions it names) goes to
a Sonnet subagent, graded blind to session and date: **kept** (the edit doesn't change what the
decision decided), **superseded, informed** (a later record supersedes it, or the user asked for the
change), or **silent reversal** (it changes the decided behaviour with neither). Emilio spot-checks
every reversal and five random kept items. Claude only, no Codex.

## Targeted replays (at most 5)
For benefit events in 3b, replay the session's first prompt on a copy of the repo at that commit
with `.compass/` and the compass instructions removed (Claude subagents, the
`tools/claude-signal/` method), and count how often it reverses or asks. This bounds what compass
was worth on the events that matter, without a second arm for the whole week.

## Limits
One user, one repo, one week: perhaps 10–20 sessions. Path and name flagging misses decisions that
name neither; the human-catch grep is the backstop. The judge is an LLM; reversals are hand-checked.

## Results (2026-10-30)

Appended after the week; the protocol and bars above are unchanged. Window 2026-10-26 to 10-30 14:47
UTC: 8 real sessions (two headless test runs of 10-29/30 excluded). Run:
`tools/field-retro/retro.py --repo ~/Dev/Personal\ Projects/personal-dashboard --former-path
~/Dev/Misc/personal-dashboard` (the repo moved on 10-28; the script was fixed the same day to follow it
and to count briefs and current-state rewrites correctly, see context/2026-10-29-ebay-flipper-v3-days-1-4.md).

**The week was half dark.** A dangling `~/.claude/settings.json` symlink turned off every hook and
the plugin for 5 of the 8 sessions (10-28 14:28 to 10-29 ~21:00). Fixed by decision 0020 (compass
says so) and the e2e install test; the measures below mix 3 lit and 5 dark sessions.

| # | Measure | Result | Bar | Verdict |
|---|---|---|---|---|
| 1a | Continue-style sessions that oriented first | 0/0: no session opened with a continue prompt | ≥ 80% | no data |
| — | Brief injected at start | 1/8 (v3); 2 more got v2's brief before the plugin was installed | tracked | outage |
| — | Oriented before first edit (brief or `compass resume`), any session | 5/8 (4 of the 5 dark sessions ran `compass resume` from the AGENTS.md block) | tracked | — |
| 1b | Sessions with a commit that rewrote current-state | 5/6 (83%) | ≥ 90% | below bar, above kill (60%) |
| 2a | Flagged edits searched first | 12/12 | ≥ 80% | pass, weak (below) |
| 2b | Silent reversals | 0/12 (blind Sonnet judge: 12 kept) | 0 | pass |
| 2c | Human catches | 0 | read by hand | none |
| 3a | Compass share of tool I/O, median | 15.6% (lit 14.8%, dark 16.2%) | ≤ 10% | fails bar, under kill (16.8%) |
| 3b | Benefit events | 12 flagged edits kept after a search: an upper bound; 0 where a decided behaviour was at stake | ≥ 1 per 10 | not shown |

**2a/2b are weak evidence.** All 12 flagged edits were new files (scratch analysis, probes, tests, an
additive market feature); the flagger caught no edit to existing decided code. The judge's grades:
kept ×12, each with a one-line reason (no decision clause contradicted). Spot-check for Emilio, drawn
at random (seed 30): E01 `pd-push-accuracy/internal/deals/zz_probe_integration_test.go` (17176c1c),
E04 `internal/store/apple_market_test.go` (3ca29fd6), E05 `frontend/src/routes/insights/market/+page.svelte`
(034abcc2), E09 a read-only EDA scratch script (17176c1c), E10 `internal/deals/watch_parse_test.go`
(ecb202aa).

**Replays: none run.** The protocol replays benefit events to bound what compass was worth. No event
had a decision at stake (every flagged edit was additive), so a replay without compass would show
nothing; spending up to 5 on it would measure the flagger, not compass.

**3a, where the I/O went.** Of compass I/O: task plans in `.compass/logs/plans/` 28%, other
`.compass/` reads 24%, experiments 15%, current-state 14%, decisions 10%, compass CLI output 9%.
ebay-flipper keeps its task plans under `.compass/`, which decision 0016 says belong with the task;
counted as task reads instead, the share is roughly 11% (estimate, not re-run).

**Verdicts against the protocol's signals.** No silent reversal, so `compass find` (decision 0019) is
not triggered. 3a is between bar and kill: no per-session trimming yet; move ebay-flipper's plans out
of `.compass/` first and re-measure. 1a and 3b have no usable data this week; the dark sessions
dominate everything else.
