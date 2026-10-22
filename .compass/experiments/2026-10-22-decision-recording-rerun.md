---
claim: "Targeted Codex re-run of decision 0017: trivia no longer recorded, real decisions still recorded with a body, no rounding regression"
date: 2026-10-22
refs:
  - decisions/0017-narrow-decision-recording.md
  - experiments/2026-10-22-codex-core-operations-eval.md
---
# Targeted Codex re-run of decision 0017: trivia no longer recorded, real decisions still recorded with a body, no rounding regression

Status of this file: **protocol and pass bars written before any run** (committed first). Results
are appended below in a later commit.

## What is tested
Decision 0017's two fixes, found by experiments/2026-10-22-codex-core-operations-eval.md:
1. The AGENTS.md block now asks for a decision record only when the agent picks between real
   alternatives a later agent could reasonably undo, with `--body-file` carrying the why and the
   rejected alternatives. In the core eval, runs recorded trivia: 4 of 6 runs on the two scenarios
   below recorded a choice nobody would reverse.
2. `compass record` warns on stderr when it writes the bare skeleton, and `check` warns
   `record-body-empty`. In the core eval, 1 of 3 discount records was an empty skeleton.

## Setup
The core eval's harness, fixtures and prompts unchanged (same `invoicer` fixture, isolated
`CODEX_HOME`, empty `ZDOTDIR`, `gpt-6.1-sol` at high effort, a fresh copy per run), with the
fixtures rebuilt by the Go `compass` from this commit, so their AGENTS.md carries the new block.
Only compass arms run; the no-compass baselines don't change with the block.

## Runs (3 each, 12 in all)
- **R1, trivia (the core eval's S3-C):** add a `-currency` display flag on a stale current-state.
  Target: no decision record. Core eval: 3/3 recorded one.
- **R2, trivia (S1-S):** "Continue." with the stub; the next step is a fully specified `Summary()`.
  Target: no decision record. Core eval: 1/3.
- **R3, a real decision (S4-C):** an optional `discount_percent`. Target: a decision record whose
  body states whether the discount applies before or after tax and why. Core eval: 2/3, plus one
  empty skeleton.
- **R4, regression guard (S5-C):** the customer's half-cent complaint with the rounding decision
  `scope: local`. Target: the rounding probe still gives 12 and the reply says it's intended.
  Core eval: 3/3.

Every final tree also gets `compass check`, counting `record-body-empty`.

## Pass bars (set before the runs)
1. **Over-recording fixed:** at most 1 decision record across the 6 runs of R1 and R2 (core eval:
   4 of 6).
2. **Real decisions still recorded:** R3 states the tax base and why in ≥ 2/3 runs.
3. **No empty records:** 0 `record-body-empty` findings across all 12 final trees.
4. **No regression:** R4 honours the decision 3/3.

n = 3 per run type, one model, one synthetic repo: a directional check of the fix, not a measure.

---

# Results (2026-10-22, after the runs)

12 runs, all exited 0, `go test ./...` passed in every final tree; every run ran `compass resume`
first and rewrote current-state.

| Run | Decision records | Outcome |
|---|---|---|
| R1 currency flag ×3 | 0, 0, 0 (core eval: 3/3) | flag done; nothing recorded |
| R2 Summary (stub) ×3 | 0, 0, 0 (core eval: 1/3) | step done; nothing recorded |
| R3 discount ×3 | 1, 1, 1, all with full bodies | tax base and why stated in 2/3 |
| R4 rounding complaint ×3 | 0 | honoured 3/3, citing the decision |

- **Bar 1, over-recording fixed: PASS.** 0 decision records in the 6 trivia runs (core eval: 4 of 6).
- **Bar 2, real decisions still recorded: PASS (2/3, at the bar).** All three R3 runs recorded
  unprompted with Context/Decision/Alternatives filled in. Two state the discount applies to the
  tax-inclusive total and reject "discount before tax" because it would change the per-line tax
  Ledgerly reconciles; R3-1's record covers only the float exception and never names the tax base.
  All three framed the record as extending decision 0001 (integer cents) and superseded it, so
  the tax-base choice sits under a cents claim rather than its own.
- **Bar 3, no empty records: PASS.** 0 `record-body-empty` across the 12 final trees (core eval
  had 1 of 3 discount records empty).
- **Bar 4, no regression: PASS.** R4 kept half-to-even 3/3 and cited the decision.

Other: R3's uncached tokens rose to 27–33k (core eval 16–22k), the price of writing real bodies.

## Verdict
Decision 0017 does what it was for: trivia records went from 4 of 6 to 0 of 6, empty bodies to 0,
with no loss on the rounding guard. Still open: agents attach a new choice to the nearest existing
decision (superseding it) instead of recording the choice under its own claim, which can hide it;
n = 3, so this is direction, not measure.
