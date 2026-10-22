---
claim: The AGENTS.md block tells agents to search the records with rg before changing behaviour; no search command is built
date: 2026-10-22
premises:
  - goal:disposable-sessions
refs:
  - commit:51277e4
  - experiments/2026-10-22-retrieval-q2-keywords.md
---
# The AGENTS.md block tells agents to search the records with rg before changing behaviour; no search command is built

## Context
The retrieval-at-scale eval (branch `claude/retrieval-experiment-0cc627`, results in commit
51277e4, `experiments/2026-10-22-retrieval-at-scale.md` there) ran 72 Codex runs on ebay-flipper's
127 live records with three old decisions guarding behaviour no comment or test explains. As
shipped, compass honoured them 3/9: agents ran `compass resume`, then searched the code by
identifier and never reached `decisions/`. Every claim in INDEX gave 4/9; a 10-record memory 5/9.
One AGENTS.md line telling agents to search the records first gave 9/9, as did keyword and
SQLite/BM25 `compass find` commands; no mechanism beat the line on honour or tokens. The blind
offline set (experiments/2026-10-22-retrieval-q2-keywords.md) found no ranker, keyword field or
embedding blend worth building either.

## Decision
The managed AGENTS.md block carries the tested line word for word: "**Before changing
behaviour,** search the records: `rg -il <words> .compass/` lists the files that mention the
words." No search command, index change or dependency is added.

## Alternatives
- A keyword `compass find` over live records (9/9, ~150 lines, no dependency): not now. It ties the
  line; its only edge is a short ranked list without superseded records, which changed no outcome.
  Build it if a run reverses a decision because the target was buried in `rg` output or a
  superseded record misled it.
- SQLite FTS5/BM25 find (9/9, +6.8 MB): rejected, nothing over keyword search offline or online.
- Uncapped INDEX (4/9) and `compass why <file>` (6/9): rejected.

## Consequences
- Repos with the old block see `agents-block-drift` until `compass init` is re-run.
- The evidence is one model, one repo and three planted targets that were easier to find than a
  random real decision; harder vocabulary or a much larger corpus may favour a ranked live-only search.
