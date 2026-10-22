---
claim: Record a decision only for real alternatives a later agent could undo, with its why; an empty-bodied record warns
date: 2026-10-22
premises:
  - goal:disposable-sessions
refs:
  - experiments/2026-10-22-codex-core-operations-eval.md
---
# Record a decision only for real alternatives a later agent could undo, with its why; an empty-bodied record warns

## Context
In the Codex core-operations eval, "record a decision when it happens" made agents record choices
nobody would reverse (a missing trailing newline, a display suffix: 4 of 6 runs on those tasks),
and one of three real decisions was the unfilled skeleton: a claim with no why. Records are
immutable, so both kinds of noise accumulate and bury the decisions that prevented a reversal.

## Decision
- The AGENTS.md block asks for a decision record only when the agent picks between real
  alternatives and a later agent could reasonably undo it (the rejected option looks simpler, or
  the reason lies outside the code), written with `--body-file` carrying the why and the
  rejected alternatives. A choice the code or a comment explains is not a decision.
- A record whose body is nothing but headings (and HTML comments) is empty: `compass record`
  warns on stderr when it writes one, and `check` warns `record-body-empty`. A record without a
  claim gets only `claim-missing`.

## Alternatives
- Make `record` refuse an empty body: rejected, a warning is enough and keeps the two-step
  "record, then fill in before committing" flow working.
- An error in `check`: rejected, warnings never block a commit, and a filled-in body can't be
  judged mechanically beyond "is there one".

## Consequences
- Fewer, fuller records; whether agents still record the decisions that matter is checked by the
  targeted re-run in experiments/.
