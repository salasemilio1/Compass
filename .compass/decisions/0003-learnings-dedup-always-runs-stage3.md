---
claim: "dropped the confidently-novel lexical short-circuit after real-content calibration found true paraphrases it silently missed"
---
# 0003 — learnings dedup always runs stage 3, dropping the confidently-novel lexical short-circuit

Date: 2026-07-06

## Context

decisions/0002 introduced a three-stage dedup filter for `.compass/learnings/*.md`: exact hash -> lexical
Jaccard -> embedding, with the embedding model deliberately gated as "a genuine last resort, not called
on every run." Concretely, a lexical score at or below `LEXICAL_LOW` (0.1) was treated as confidently
novel and returned `OK` without ever invoking the embedder.

Calibrating that filter against this repo's own real `.compass/learnings/*.md` content (not synthetic
fixtures) surfaced two real, not hypothetical, false negatives under that short-circuit:

- A true paraphrase of `verify-flagged-installs-directly.md` with almost no shared vocabulary
  (0.0 lexical score) — a real paraphrase risk, not an edge case.
- `context/current-state.md`'s own "In flight" bullet about the blocked `.git/hooks/pre-commit`
  backstop, which describes the *exact same real incident* as
  `plan-approval-does-not-blanket-authorize-actions.md`'s generalized learning — two real documents
  already in this repo, ~0.05 lexical score, that would never reach the model if one were resubmitted
  as a new learning candidate today.

3-word-shingle Jaccard structurally cannot see a same-meaning pair that shares no consecutive phrase —
that's exactly the case an embedding model exists to catch. Skipping the model on a low lexical score
throws away the one stage that could have caught these.

## Decision

`find_duplicates` no longer short-circuits on a low lexical score. `LEXICAL_LOW` is removed. Only two
things now skip the embedder: an exact hash match, and a lexical score at or above `LEXICAL_HIGH` (0.6).
Every other candidate — including a 0.0 lexical score — falls through to stage 3 when an embedder is
available.

## Alternatives considered

- **Keep the short-circuit, lower `LEXICAL_LOW` instead** — doesn't help; both real false negatives
  above scored at or near 0.0, so no nonzero cutoff would have caught them without also being low
  enough to be meaningless as a filter.
- **Leave it as documented residual risk (status quo)** — decisions/0002 already flagged this
  trade-off explicitly as accepted risk. Revisited because calibration testing turned "could happen"
  into "already has happened, twice, in this repo's own content."

## Consequences

- Gain: closes the specific residual gap decisions/0002 called out ("a true paraphrase with no shared
  vocabulary ... can still slip through") for the paraphrase case; substring/containment and
  genericized-restatement cases (already ambiguous-band, not confidently-novel) were unaffected by this
  change.
- Lose: every learning-dedup check that isn't an exact or high-lexical match now pays the fastembed
  model-load cost via `uv run`, including ones that are obviously unrelated. Accepted because the check
  only runs at `wrap-up` time (non-blocking, human-reviewed), not on every keystroke — decisions/0002's
  "gentle filter, never a gate" framing still holds.
- `tests/test_learning_dedup.py`'s calibration tests assert the *ground-truth-correct* embedder
  response for each real-content pair (not whatever value would make the assertion pass) — see
  `learnings/test-assertions-must-encode-intent-not-observed-output.md`.
