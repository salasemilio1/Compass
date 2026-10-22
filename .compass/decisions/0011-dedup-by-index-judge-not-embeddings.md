---
claim: "drop MiniLM embeddings (0.30 recall, 230 MB); dedup = cheap LLM judge over INDEX one-line hooks (P 0.96 / R 0.99, $0.017); merges append, never delete; no bulk consolidation (supersedes 0003)"
---
# 0011 — dedup learnings with a cheap LLM judge over the index; drop embeddings

Date: 2026-10-21
Supersedes: 0003
Status: **conditional** (2026-10-21): applies only if learnings survive as a category. v3 routes
learnings out (to a rule, a comment or the decision they affect), which may leave nothing to dedup;
see [positioning.md](../context/positioning.md). Not part of the core build.

## Context
The learnings dedup filter (`tools/learning-dedup/check_similarity.py`) runs exact hash, then
3-shingle Jaccard, then MiniLM embeddings. 0003 made the embedding stage always run, after two real
near-zero-overlap paraphrases slipped past the lexical stages. The embedding stage is about 230 MB of
dependencies (a 143 MB Python environment plus an 87 MB model) and needs onnxruntime and cgo in Go,
which breaks [0010](0010-go-binary-as-claude-code-plugin.md)'s static single binary.

## Evidence
From experiment `2026-10-21-dedup-architectures.md`, on branch `claude/dedup-experiment-436ce8`
(commits `dd319c8` protocol, `ded4590` results).
- **Setup:** ebay-flipper's 181 learnings, and 130 labelled candidates (synthetic paraphrases at three
  strengths, held-out negatives, natural pairs, and 0003's two hard cases). Labels were written before
  any results.
- **Today's filter:**
  - Hash and Jaccard: recall 0.00.
  - MiniLM at 0.80: precision 1.00 but recall 0.30. It caught 30/30 light rewordings, 0/30 heavy
    paraphrases and **0/2 of 0003's own cases**. The embedding ranked them first; the threshold never
    accepted them. No threshold worked.
- **BM25 matches MiniLM as a retriever:** recall@5 0.97 vs 0.98, with zero dependencies.
- **Haiku judging the INDEX one-line hooks plus the candidate:** precision 0.96–0.97, recall
  0.99–1.00, about $0.017 and 12 s per check.
- **Unsupervised Haiku consolidation passes:** found 2–4 of the 8 real merges, and all 18 merges they
  proposed beyond the labels were rated distinct.
- **Duplication is rare:** folding every real duplicate shrinks the corpus only 181 → 172 (−5%).

## Decision
1. **Drop the embedding stage and its dependencies.**
2. **Dedup is a judge over one-line summaries.**
   - The binary assembles the prompt deterministically: every learning's one-line summary plus the
     candidate, with BM25 top 5 as a fallback or prefilter past ~1,000 learnings.
   - The verdict comes from an LLM that is already available: the session agent, a Haiku subagent,
     or `claude -p --model haiku`.
   - No similarity score is ever thresholded into a verdict.
3. **Merges never delete.** A "same lesson" verdict appends the new detail to the existing entry, so a
   false positive (about 10% of same-incident follow-ups) costs length, never information. This makes
   it safe to run without human review.
4. **No bulk consolidation passes.** Leanness comes from routing learnings to rules, comments or the
   decisions they affect ([positioning](../context/positioning.md)), not from automatic merging.
5. The one-line `claim`/summary is required frontmatter, since judge quality depends on it.

## Alternatives considered
- **Keep embeddings as an optional add-on.** Rejected for now: 0.30 recall at the production
  threshold, and no threshold worked. Revisit only if a future retrieval eval shows semantic search
  beating BM25.
- **BM25 top 5 plus a full-text judge** (precision 0.91, recall 0.95, $0.010). This is a viable
  cheaper variant. The index judge won on recall because it never loses a candidate at retrieval.
- **Whole-corpus judge:** precision 0.94, recall 0.95, about $0.07 uncached. Scales worst.

## Consequences
- Zero native dependencies. Dedup ports to about 100 lines of standard-library Go.
- Dedup needs an LLM call, which is fine since one is always in the loop at write time. With no LLM
  available, it prints the BM25 top 5 instead of a verdict.
- The prompt grows about 45 tokens per learning, which is fine to about 1,000 learnings.
- **Threats not yet tested:** human-written paraphrases (the generator, labels and judge are all Claude
  models), a single corpus, and thinking left on (capping it is the main cost lever).
