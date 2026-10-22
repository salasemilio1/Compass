---
claim: One choice per record, its claim naming it; supersede only when the old claim is untrue, and record says what it retires
date: 2026-10-22
premises:
  - decision:0017-narrow-decision-recording
refs:
  - experiments/2026-10-22-decision-recording-rerun.md
---
# One choice per record, its claim naming it; supersede only when the old claim is untrue, and record says what it retires

## Context
After 0017, the targeted re-run (experiments/2026-10-22-decision-recording-rerun.md) showed agents
folding a new choice into the nearest existing decision: all three discount runs superseded
"amounts are integer cents" and put the tax-base choice in its body, so no claim (the line INDEX
and resume show) named it. A probe also showed superseding a global record without `--global`
silently stopped resume delivering it.

## Decision
- One choice per record, and its claim names that choice.
- Supersede only when the old record's claim is no longer true as written; the new claim states
  the whole current answer. A choice that builds on a decision still in force is a new record that
  cites it (`--premise decision:<id>`).
- `compass record` names each claim a supersede retires, says when dropping `--global` stops
  delivery, and points at the live record when the target was already superseded.
- `check` warns `supersedes-forked` when a record's successors end in more than one live record
  (two merged branches), following chains.

## Alternatives
- Refuse a supersede without a flag stating why: rejected for now, more friction; revisit if the
  warnings don't stop folding.
- Detect a record covering two choices: rejected, not mechanically decidable.

## Consequences
- More records stay live, so retrieval at scale matters more; a separate experiment measures it.
