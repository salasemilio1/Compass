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
