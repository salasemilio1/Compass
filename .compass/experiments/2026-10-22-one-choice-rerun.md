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

---

# Results (2026-10-22, after the runs)

9 runs, all exited 0, `go test ./...` passed in every final tree; every run ran `compass resume`
and rewrote current-state.

| Run | Decision records written | Where the tax-base choice went |
|---|---|---|
| R3-1 | none | current-state (a new "Status" section) and a code comment; its code needed no float, so 0001 was never in play |
| R3-2 | new record, superseding nothing: "Percentage discounts permit a temporary float calculation before rounding back to cents" | code comment; not in the record |
| R3-3 | superseded 0001 (the retire warning fired): "Amounts remain int64 cents; floats are limited to tax and discount intermediates" | record body and code comment |
| R1 ×3 | none | n/a |
| R4 ×3 | none | rounding kept 3/3, decision cited |

- **Bar 1, no fold: FAIL (0/3).** No run gave the tax-base choice a claim of its own. Folding fell
  from 3/3 to 1/3 (R3-2 recorded its float choice as a new record instead of superseding 0001, as the
  rule asks), but the tax-base choice itself was never treated as a decision: the runs recorded
  the float exception, which touches an existing record, and left "after tax" to code comments.
- **Bar 2, trivia stays out: PASS** (0 records in R1).
- **Bar 3, no regression: PASS** (R4 3/3).
- **Bar 4, no empty or forked records: PASS** (0 and 0 across 9 trees).

## Verdict
The rule fixes how agents record (fewer folds, nothing empty or forked) but not what they think
is a decision. Every run across both re-runs (6/6) chose "after tax" and wrote its reason in a code
comment, so the why is not lost; it is just not a decision record. A wording change can't make an
agent see an obvious-to-it choice as a choice. What would: a reversal of it later (a record gets
written when someone pushes back), or a human naming the choice. One run also wrote status into
current-state (a "Status" section), which 0016 says belongs in git. n = 3.
