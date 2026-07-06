# calibrate learnings dedup against real content, close the residual paraphrase gap, streamline wrap-up

Date: 2026-07-06
Commits: 403a183..HEAD

## Why
Two follow-ups from the previous session's gap-closing work (decisions/0002): (1) the learnings-dedup
filter's thresholds were untested against this repo's own real content, only synthetic fixtures; (2)
wrap-up's review flow (dump full drafts, then ask write/commit/push separately, in sequence) was
dragging out sessions right when the user runs it precisely because they're low on context and want to
end.

## What changed
- `tests/test_learning_dedup.py` — added 10 calibration tests built from the actual current content of
  `.compass/learnings/*.md` and `context/current-state.md` (not synthetic fixtures), covering: stage-1
  reformatting-only duplicates, stage-2 near-verbatim reuse (both a clean cosmetic rename and a
  changed-tail case), true negatives with high shared domain jargon but low lexical overlap, and
  ambiguous-band cases resolved by stage 3 (verbatim excerpt shorter than its source, a genericized
  restatement, and a corrective counter-learning that must *not* be flagged as a duplicate).
- `tools/learning-dedup/check_similarity.py` — removed the `LEXICAL_LOW` "confidently novel, skip the
  model" short-circuit. Stage 3 now runs on every candidate stage 1/2 didn't already confidently
  resolve. See decisions/0003.
- Updated the calibration tests that had relied on the old short-circuit (3 tests) to assert the
  ground-truth-correct embedder response instead of documenting a slip-through gap, per the new
  test-writing rule below.
- `skills/wrap-up/SKILL.md` (step 7) — replaced "present every draft, then ask about write/commit/push
  in sequence" with: summarize each artifact in 2-3 sentences (no full-content dump by default), ask one
  consolidated batch of questions covering write/commit/push, execute, and stop.
- `~/.claude/skills/new-test/SKILL.md` and `~/.claude/skills/audit-tests/SKILL.md` (in the separate
  dotfiles repo, not this one — flagged here for visibility only) — added rule 6: assertions (including
  mocked/injected values like fake-embedder vectors) must encode the correct expected behavior decided
  from spec/intent before the code runs, never reverse-engineered from whatever the code currently
  outputs.

## Assumptions changed
decisions/0002 framed the embedding model as "a genuine last resort, not called on every run," gated by
a low-lexical-score short-circuit. Calibrating against real content found two real (not hypothetical)
pairs that short-circuit would have silently missed. decisions/0003 documents dropping it.

## Learnings
- `learnings/test-assertions-must-encode-intent-not-observed-output.md`

## Breaking changes
None — additive test coverage and a stricter (not looser) dedup filter. The only behavior change is
`check_similarity.py` now invoking the embedder (and paying its model-load cost) on more candidates than
before; still non-blocking, still exits 0 always.
