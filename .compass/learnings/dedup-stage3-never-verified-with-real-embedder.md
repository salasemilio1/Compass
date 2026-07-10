# dedup stage 3 verified against a real embedder (2026-07-10)

decisions/0003's "calibration against real content" only exercised stages 1-2 (hash, lexical
Jaccard) with real `.compass/learnings/*.md` text. Stage 3 (fastembed/MiniLM cosine similarity) was
calibrated with hand-picked mock vectors — never the real model's actual output.

**Verified 2026-07-10**: ran `check_similarity.py` for real (via `uv run`, pulling the real
`fastembed` dependency) against two hand-written paraphrases of
`bash-pipefail-empty-substitution.md`, low enough lexical overlap to skip straight to stage 3:
- A loose paraphrase (different structure, same idea) scored 0.59 cosine — correctly stayed `OK`,
  below the 0.80 threshold.
- A tight paraphrase (same structure, synonyms swapped) scored 0.85 cosine — correctly printed
  `POSSIBLE_DUPLICATE bash-pipefail-empty-substitution.md embedding:0.85`.

Also resolved the earlier "model weights not found" observation: `fastembed`'s default cache is the
OS temp dir (`$TMPDIR/fastembed_cache`, ~87MB for MiniLM), not `~/.cache` — the model had never
failed to download, the wrong directory was checked. Stage 3 is confirmed working end-to-end, not
just correct on paper.

**How to apply:** no further action needed on stage 3 itself. If auditing other compass components
for "calibrated against real content," check whether *every* stage was, the way this one now has
been — and remember `fastembed`'s cache lives in the OS temp dir, not `~/.cache`, when checking
whether a model was actually downloaded.
