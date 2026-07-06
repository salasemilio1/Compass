# close audited compass gaps with deterministic scripts

Date: 2026-07-06
Commits: f01c3c9..(uncommitted -- see note below)

## Why
Followed up an audit of compass's own design (assuming continued real use) that found five places
prompt-only discipline would let it drift toward bloat/staleness: unbounded INDEX.md growth, no
dedup/supersession for learnings/decisions, a global-install blast-radius risk, unverifiable
current-state.md staleness, and silent AGENTS.md/spec drift. See decisions/0002 for the reasoning.

## What changed
- `scripts/compass-check.sh` (new) -- deterministic checker+fixer for INDEX.md archival (15-entry cap),
  ADR supersession marking, and current-state.md's Commit: anchor; AGENTS.md spec-version drift is
  warn-only.
- `tools/learning-dedup/check_similarity.py` (new) -- three-stage dedup filter (hash -> lexical Jaccard
  -> fastembed/MiniLM cosine as last resort) for new learnings/ entries.
- `install.sh` -- canonical-source guard (`$DOTFILES/compass` or explicit `--dev`); `COMPASS_HOME` env
  override replacing hardcoded `$HOME/.claude` throughout, for testability.
- `hooks/compass-session-start.sh` -- deterministic staleness fact (commit count behind HEAD) appended
  to the injected context when current-state.md's stamp is behind.
- `SPEC.md` -- spec version marker, `Supersedes:`/`Commit:` conventions, INDEX.md cap rule, B2 residual
  gap note.
- `skills/wrap-up/SKILL.md`, `skills/init-compass/SKILL.md` -- wired the new mandatory steps
  (compass-check --fix, dedup check) and the AGENTS.md spec-version stamp.
- `tests/` (new) -- `run_tests.sh` runner + `test_compass_check.sh` (28 checks), `test_install_guard.sh`
  (12 checks), both plain bash, no framework; `test_learning_dedup.py` (25 pytest cases, one real `uv
  run` integration smoke test) -- all passing.
- This repo's own `.compass/` and `AGENTS.md` brought up to the new conventions via the tool itself
  (dogfooded compass-check.sh --fix rather than hand-editing).

## Assumptions changed
Considered and rejected three external tools (Caveman, Headroom, semantic-tool-selection) surfaced
during the same discussion -- all operate at the wrong layer (runtime compression vs. compass's curated
repo memory), or are already solved by the harness. See decisions/0002.

## Learnings
- `learnings/bash-pipefail-empty-substitution.md`
- `learnings/test-helper-naming-must-avoid-test-glob.md`
- `learnings/plan-approval-does-not-blanket-authorize-actions.md`

## Breaking changes
None -- all additive. `install.sh` now refuses to run from a non-canonical checkout without `--dev`,
which is a behavior change for anyone who was relying on the old unconditional-symlink behavior from a
dev clone; the fix is a one-time `--dev` flag or setting `$DOTFILES` correctly.

## Note on this entry's Commits: range
Everything above is currently **uncommitted** in the working tree (HEAD is still f01c3c9, same as the
prior log's anchor). The next wrap-up's anchor mechanism only works cleanly once this is committed --
this session commits it (see below), so the next wrap-up should anchor off that new commit.
