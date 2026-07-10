# dedup stage 3 has never been verified against a real embedder

decisions/0003's "calibration against real content" only exercised stages 1-2 (hash, lexical
Jaccard) with real `.compass/learnings/*.md` text. Stage 3 (fastembed/MiniLM cosine similarity)
was calibrated with hand-picked mock vectors standing in for "the ground-truth-correct embedder
response" (e.g. `[0.1, 0.99]`) — never the real model's actual output. No log or test in the repo
shows a `POSSIBLE_DUPLICATE`/`DUPLICATE ... embedding:` verdict produced by the real embedder.

The `fastembed` package is cached in `uv`'s wheel cache on this machine (confirms it was resolved
at least once), but the MiniLM model weights it would download on first real run were not found
under this user's cache directory — consistent with the model never having actually loaded.

**Why it matters:** the design (three-stage cascade, model as last resort) is sound and the
docs/tests read as if it's been validated end-to-end, but "validated with a mock" and "confirmed
working" are different claims. Before trusting stage 3 to catch a real paraphrase in practice,
someone needs to run `check_similarity.py` with the real `fastembed` dependency against a genuine
near-duplicate pair and confirm it actually fires — not just that the mocked test asserts the
intended behavior.

**How to apply:** when auditing any compass component described as "calibrated against real
content," check whether every stage was calibrated that way, or whether some stage's tests
substitute a mock for the expensive/slow part. A green test suite doesn't mean every code path in
it has executed for real.
