---
claim: "Targeted Codex re-run of decision 0018: the tax-base choice gets its own record instead of folding into integer cents"
date: 2026-10-22
refs:
  - decisions/0018-one-choice-per-record.md
  - experiments/2026-10-22-decision-recording-rerun.md
---
# Targeted Codex re-run of decision 0018: the tax-base choice gets its own record instead of folding into integer cents

Status of this file: **protocol and pass bars written before any run** (committed first). Results
are appended below in a later commit.

## What is tested
Decision 0018's rule, from the decision-recording re-run (experiments/2026-10-22-decision-recording-rerun.md):
all three discount runs folded the tax-base choice into the integer-cents decision and superseded
it, so no claim named the choice. The AGENTS.md block now says one choice per record with a claim
that names it, and supersede only when the old claim is untrue as written; `compass record` names
each claim a supersede retires and warns when `--global` is dropped.

## Setup
The core eval's harness, fixtures and prompts unchanged, fixtures rebuilt by the Go `compass`
from this commit (so AGENTS.md carries the new block). Compass arms only.

## Runs (3 each, 9 in all)
- **R3, the fold case (S4-C):** optional `discount_percent`. Target: a record of its own whose claim
  names the discount's tax base (the claim mentions the discount and tax, before or after).
- **R1, trivia (S3-C):** a `-currency` display flag. Target: no decision record.
- **R4, regression guard (S5-C):** the half-cent complaint. Target: rounding probe still 12 and the
  reply says it's intended.

## Measures
Per run: decision records written (claim, supersedes), whether 0001 (integer cents) was superseded,
`check` findings on the final tree (`record-body-empty`, `supersedes-forked`), and the same
efficiency numbers as before.

## Pass bars (set before the runs)
1. **No fold:** in ≥ 2/3 R3 runs the tax-base choice has its own record whose claim names it
   (re-run before 0018: 0/3).
2. **Trivia stays out:** at most 1 decision record across R1's 3 runs.
3. **No regression:** R4 honours the decision 3/3.
4. **No empty or forked records:** 0 `record-body-empty` and 0 `supersedes-forked` across all 9 trees.

Superseding 0001 alongside a separate tax-base record is reported, not failed: extending floats to
percentages arguably makes 0001's claim untrue as written. n = 3: direction, not measure.
