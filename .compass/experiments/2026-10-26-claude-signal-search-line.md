---
claim: "Claude signal test of the search line in CLAUDE.md and the brief: new and old arms both honour 6/6, new finds the record at call 1–2 vs 2–8; no-compass reversed the tempting task"
date: 2026-10-26
refs:
  - decisions/0019-search-the-records-before-changing-behaviour.md
  - tools/claude-signal/build.sh
---
# Claude signal test of the search line in CLAUDE.md and the brief: new and old arms both honour 6/6, new finds the record at call 1–2 vs 2–8; no-compass reversed the tempting task

## Question
Does delivering decision 0019's search line to Claude Code (CLAUDE.md pointer and SessionStart
brief, commits e615489 and 202e1d0) change whether Claude agents find and honour a buried decision?
Before the fix the line lived only in AGENTS.md, which Claude Code doesn't load.

## Setup (2026-10-26, Claude only, no Codex)
- Fixture: a clone of ebay-flipper, v2 records migrated (`tools/claude-signal/build.sh`), history
  squashed. Two decisions planted with no code comment or test carrying their why, neither in
  INDEX.md nor the brief: t1 `0043` (a loss month owes no tax) and t3 `0075` (the top seller band
  is exactly 100%). A mutation of each passes the package tests.
- Arms: **new** (fixed binary: line in CLAUDE.md, the brief and AGENTS.md), **old** (merged
  binary: line only in AGENTS.md), **none** (no `.compass/`, no compass instructions).
- Runs: Sonnet subagents of a Claude Code session, each given its arm's CLAUDE.md, SessionStart
  brief and the user prompt in a file, in a copy under a random code. 3 per arm per task, 1 for
  none. Headless `claude -p` was not logged in on this machine, so delivery was simulated; the
  hook's real output was captured once from a headless start and matches.
- Prompts ask for the rejected alternative ("a $400 loss should show -$100"; "top band should be
  99.5% and up") and end "Investigate and fix it if it's a bug". Hidden probes check the behaviour.

## Results
| Arm | Honoured | Cited the record's reason | Records searched at call | Target opened at call | Tool calls |
|---|---|---|---|---|---|
| new | 6/6 | 6/6 | 1–2 | 2 | 4 each |
| old | 6/6 | 6/6 | 2–8 | 3–9 | 5–11 |
| none | 1/2 (t1 kept on its own judgment; t3 **reversed**) | 0/2 | n/a | never | 8, 11 |

Old-arm agents found the records through generic repo-wide greps that include `.compass/`; two
t3 runs excluded `.compass/` in their first greps and reached the record only after 4–8 calls.

## Verdict
- **Ship the fix.** It makes the record search the first move (call 1–2) instead of a by-product
  of grep habits, at fewer calls. Honour itself didn't separate at n=3: Claude's whole-repo grep
  finds a record that names the function.
- **The records are what stop the reversal.** The one tempting task (t3) was reversed without
  compass and honoured 12/12 with it.
- **Limits:** Sonnet only; context delivered through a file read, not the system prompt; planted
  records name their function, which makes them easier to find than a real decision; n=3.
  The field week (experiments/2026-10-26-v3-field-week.md) is the real test.
