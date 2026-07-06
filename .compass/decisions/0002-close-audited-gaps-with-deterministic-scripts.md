# 0002 -- close audited compass gaps with deterministic scripts, not prompts

Date: 2026-07-06

## Context
An audit of compass under continued real use (init/warm-up/wrap-up repeated indefinitely) surfaced five
gaps where prompt-only enforcement would let compass drift toward the bloat/staleness it exists to
prevent: unbounded INDEX.md growth, no dedup/supersession for learnings/decisions, global skill install
pointing at a mutable dev checkout, unverifiable current-state.md staleness, and silent AGENTS.md/spec
drift. Operating principle: prompts are suggestions, code is enforcement.

Before building anything in-house, evaluated three external tools surfaced during the same discussion --
Caveman (output-verbosity compression), semantic tool selection (FAISS/embeddings for tool-schema
filtering), and Headroom (reversible tool-output/RAG compression). All three operate at the wrong layer:
they compress runtime noise, whereas compass's actual lever is curated, durable, human-reviewed repo
memory. Semantic tool selection is also already solved by the harness itself (ToolSearch's deferred-tool
mechanism). None were adopted.

## Decision
- **`scripts/compass-check.sh`** -- a single deterministic checker+fixer (`--fix` mutates, default is
  check-only) closing four of the five gaps fully: INDEX.md archival past a 15-entry cap (moves overflow
  to `<category>/archive.md`), ADR supersession marking (`Supersedes: NNNN` convention), and
  `current-state.md`'s `Commit: <hash>` anchor. A fifth check (AGENTS.md spec-version drift) is
  detection-only and never blocks -- its fix is a manual content edit, not a mechanical one.
- **`install.sh` canonical-source guard** -- refuses to symlink global skills from any checkout other
  than `$DOTFILES/compass` unless `--dev` is passed explicitly, closing the blast-radius risk of a
  destructive git operation on a dev clone breaking compass globally.
- **`hooks/compass-session-start.sh`** -- deterministically compares `current-state.md`'s stamped commit
  against HEAD by commit count and injects the fact if stale. Detection only; deciding what to do about
  it correctly stays with the agent.
- **Learnings dedup (`tools/learning-dedup/check_similarity.py`) is the one gap that can't be fully
  closed** -- semantic duplication is a judgment call. Structured as three stages, hash -> lexical
  (Jaccard on word shingles) -> embedding (fastembed + MiniLM, brute-force cosine), so the model is a
  genuine last resort rather than called on every run. Rejected: chunking (SPEC already enforces short
  entries -- solves a long-document problem this corpus doesn't have), BM25 (its IDF statistics need a
  corpus far larger than a few dozen learnings to be meaningful -- plain Jaccard gets the same practical
  benefit for less code), sentence-transformers/torch and Ollama (both heavier or more infra-dependent
  than fastembed's in-process ONNX runtime, with no daemon to keep alive).
- **Enforcement scope is deliberately partial.** Wiring these into `wrap-up`'s steps is still a skill
  invoking a script -- real enforcement is the script's determinism, not whether the skill is followed.
  A genuine backstop (this repo's own local, untracked `.git/hooks/pre-commit` running compass-check.sh
  check-only) is *not* auto-installed into consumer repos via `init-compass` -- modifying another repo's
  hook chain is that repo owner's decision, not this plan's to make for them.

## Alternatives considered
- Caveman / Headroom / semantic-tool-selection -- see Context; wrong layer or already solved.
- sentence-transformers or Ollama for the dedup embedder -- heavier dependency or a standing daemon
  requirement; ADR 0001 already documented a local-model reliability caveat for a different sub-task
  (generative extraction), which doesn't directly transfer to embeddings, but the daemon dependency and
  added Python weight still weren't worth it against fastembed's self-contained ONNX runtime.
- Auto-installing the pre-commit backstop into every `init-compass`-scaffolded repo -- rejected; that's
  a per-repo decision the owner should make, not something compass should do on their behalf.

## Consequences
- Gain: INDEX.md growth, ADR supersession, and current-state.md staleness detection are now fully
  mechanical, not dependent on an agent remembering. Learnings dedup catches exact/near-exact duplicates
  deterministically and narrows (without guaranteeing) the harder paraphrase case.
- Lose/residual: a learning that has become outright false, or a true paraphrase with no shared
  vocabulary and low embedding similarity, can still slip through -- left open by design, not solved.
- Open: the local `.git/hooks/pre-commit` backstop for this repo was blocked by the auto-mode permission
  classifier during this session (writing into `.git/hooks/` was treated as unauthorized persistence
  despite being named in the approved plan) -- pending the user's explicit go-ahead, tracked in
  `context/current-state.md`.
