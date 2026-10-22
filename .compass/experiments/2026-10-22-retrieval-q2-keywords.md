---
claim: "Blind graded query set for decision retrieval: do ranking hygiene and a record-side keywords line close the gap to BM25's ceiling?"
date: 2026-10-22
refs:
  - decisions/0018-one-choice-per-record.md
---
# Blind graded query set for decision retrieval: do ranking hygiene and a record-side keywords line close the gap to BM25's ceiling?

Status of this file: **protocol and pass bars written before any query of the new set exists**
(committed first). Results are appended below in a later commit.

## Why
Offline retrieval over ebay-flipper's 127 live records (the retrieval-at-scale experiment's
corpus) stalls near MRR 0.62–0.68 for BM25, a field-weighted BM25F and a static-embedding blend.
A failure analysis of the old 60-query set found the gap to 1.0 is ~45% vocabulary the target
record never uses, ~40% label noise (5 vague queries; 7 where another record is an equally fair
answer) and ~15% ranking defects. That analysis read every old query, so the old set can no
longer score anything fairly. This experiment builds a new, blind, graded query set and measures
two fixes on it: ranking hygiene, and a record-side line of search keywords.

## Corpus
The 127 live records of the retrieval-at-scale fixture (`c127.json`), unchanged except where an
arm adds keywords.

## New query set (Q2), built blind
1. **Tasks:** 30 live decisions sampled at random (seed 20261023) from the 102, excluding the 15
   old targets, plus 5 live non-decision records (experiments, context; same seed), so a
   `decisions/` prior can't win by construction. Agent A (Codex, `gpt-6.1-sol`, one batch call
   with a fixed prompt, isolated as in the agent evals) gets each record's text and writes a plausible user
   request that would tempt a developer to reverse it, in the user's words, never quoting the
   record. A sees nothing else.
2. **Queries:** agent B (a fresh Claude Sonnet subagent, fixed prompt) gets only the 35 task texts and writes 4 searches (1–5 words) per
   task that a developer would type before changing the code: 140 queries.
3. **Relevance, pooled and graded:** the top 10 of every arm for every query, merged per task;
   records outside the pool count as 0. Judges C (fresh Claude subagents, one fixed prompt, a
   share of the tasks each) get the task text and each pooled record (method hidden, order shuffled) and grades
   it, shown without any `keywords:` line, 2 (explains why the requested change was rejected or is constrained), 1 (relevant context a
   developer should read first) or 0. The sampled decision is graded too; it is not assumed to be
   2.
None of A, B or C sees keywords, results, the old set or this analysis.

## Keywords, written as production would write them
Agent D (fresh Claude Sonnet subagents, a different model family from A, one fixed prompt, a
single pass per record, no re-prompting) plays the recording agent. For each of the 102 live decisions, it gets the
record's text and the exact rule that would ship in the AGENTS.md block:

> Give it `--keywords "..."`: 5–10 words or short phrases someone would search for when about to
> change this behaviour without knowing the decision exists: the symptom a user would report,
> their words for it, and the rejected alternative's name. Don't repeat the claim's words.

D never sees Q2 or any task. Its output becomes a `keywords:` frontmatter line in every decision
(not only the sampled 30), so no record is favoured. That is the production layout: a
frontmatter line, so every lookup that reads the file (rg, keyword, FTS5's body column) sees it;
only hygiene+kw also weights it like the claim. Reported: the share of Q2 query words that appear
in a task's keywords but not in its record, the measure of A and D sharing a source.
**Production gap, stated:** in production the recording agent writes keywords at decision time,
with the session's context, and compass does not backfill old records; this test is the
all-records-have-keywords case, the ceiling of the field's effect.

## Arms
| Arm | Ranker | Keywords |
|---|---|---|
| rg | keyword matching over the whole file, ranked by distinct query words (the prototype's keyword ranker; what an agent's `rg` sees) | no |
| rg+kw | rg | yes |
| fts5 | SQLite FTS5 BM25 (today's prototype) | no |
| bm25f | BM25F, claim + filename ×3 | no |
| hygiene | bm25f + Porter stemmer, hyphenated words kept whole, a prior for `decisions/` (all fixed from the old-set analysis, no tuning on Q2) | no |
| fts5+kw | fts5 | yes |
| hygiene+kw | hygiene, keywords weighted like the claim | yes |
| blend | cvx(BM25F, potion-retrieval-32M/256, α 0.5), for reference | no |

## Order (nothing is scored until all of it is committed)
1. This protocol. 2. Q2 tasks and queries, D's keywords, and the hygiene ranker's code with its
parameters frozen (from the old-set analysis). 3. The pools and C's grades. 4. Scores, once.

## Metrics
Per arm over Q2's 120 queries: nDCG@10 on the graded labels (primary), MRR and recall@5 of the
first grade-2 record, ms per query at 127 records. Paired comparisons by query with a
bootstrap 95% interval (10,000 resamples over tasks, seed 1).

## Pass bars (set before Q2 exists)
1. **Hygiene ships in any compass search command** (replacing FTS5 with BM25F in Go) if its
   nDCG@10 ≥ max(fts5, bm25f) + 0.02 and the bootstrap interval of the difference excludes 0, and
   its Go port runs ≤ 100 ms per query at 1,000 records (c311 padded by duplication).
2. **Keywords ship to production** (the AGENTS.md rule above, `compass record --keywords`, the
   `keywords:` field in SPEC, and `rg`/search seeing it) if, with each interval excluding 0,
   hygiene+kw ≥ hygiene + 0.05, fts5+kw ≥ fts5 + 0.05 **and** rg+kw ≥ rg + 0.05: the field must
   help whatever lookup an agent uses, including plain `rg`.
3. **The embedding blend is reconsidered** only if it beats the best keyword-free lexical arm by
   ≥ 0.05 nDCG@10 with an interval excluding 0; otherwise it is dropped for its 4–16 MB.

## What it can't show
One repo, one keyword writer, one judge; Q2 is LLM-written, so it may miss how real agents
search (the retrieval-at-scale agent runs measure that). 30 tasks bound the effect, they don't
settle it.

---

# Results (2026-10-22, scored once)

Built as pre-registered: 35 tasks (30 decisions, 5 non-decisions) written by Codex from the records
alone; 140 queries by a fresh Sonnet agent from the tasks alone; keywords for all 102 decisions by
three fresh Sonnet agents from the record and the shipping rule alone; 1,773 pooled records graded
by 12 fresh Claude judges (method hidden, keywords stripped). Inputs, pools, grades, arms and scorer
are fingerprinted in `tools/retrieval-q2/` (commits 4d5b65c, a61202e) before scoring. Every task
had at least one grade-2 record. **Deviation:** judges 6 and 7 reported grading many records from
their claim lines without reading them; both were re-run with "read every record in full" before
any score existed, and the re-runs' grades are the ones used.

| Arm | nDCG@10 | MRR (first grade 2) | recall@5 (grade 2) |
|---|---|---|---|
| rg | 0.365 | 0.379 | 0.521 |
| rg+kw | 0.379 | 0.402 | 0.557 |
| fts5 | 0.397 | 0.428 | 0.593 |
| fts5+kw | 0.402 | 0.432 | 0.593 |
| bm25f | 0.406 | 0.424 | 0.621 |
| hygiene | 0.398 | 0.438 | 0.629 |
| hygiene+kw | 0.417 | 0.447 | 0.650 |
| blend (BM25F + potion-retrieval-32M/256) | 0.451 | 0.462 | 0.643 |

Paired nDCG@10 differences, bootstrap 95% over tasks:
- hygiene − fts5 +0.001 [−0.014, +0.016]; hygiene − bm25f −0.008 [−0.028, +0.011]
- hygiene+kw − hygiene +0.019 [+0.007, +0.031]; fts5+kw − fts5 +0.006 [−0.002, +0.013];
  rg+kw − rg +0.014 [+0.004, +0.026]
- blend − bm25f +0.045 [+0.025, +0.066] (MRR +0.039 [−0.005, +0.083], r@5 +0.021 [−0.029, +0.079])

Query words that appear in the target's keywords but not in its record: 17 of 357 (4.8%), so the
task writer (Codex) and the keyword writer (Claude) barely shared vocabulary.

## Verdicts
1. **Hygiene: FAIL** (needed ≥ max(fts5, bm25f) + 0.02; got +0.001 / −0.008). The old-set
   analysis's gains (BM25F 0.652 → 0.683) were fitted to the queries it read and did not transfer
   to a blind set: the reason this set was built.
2. **Keywords: FAIL** (needed +0.05 with every lookup). The effect is real but small: +0.019 with
   hygiene and +0.014 with plain rg, both intervals above 0; +0.006 with FTS5, interval spanning 0.
   Under the pre-registered rule the field does not ship.
3. **Embedding blend: FAIL, narrowly** (+0.045 nDCG@10 against the bar's 0.05; MRR and recall@5
   intervals span 0). Dropped for its 4–16 MB.

## What it says
No ranking change moved retrieval by the margin that would justify it: every lexical arm sits at
nDCG@10 0.37–0.42, and the best non-lexical one adds 0.045. Retrieval quality of a single
search is not where the gain is; how the agent searches is (the retrieval-at-scale agent runs
measure that). One repo, LLM-written tasks and judges, 35 tasks.
