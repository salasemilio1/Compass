#!/usr/bin/env python3
# /// script
# requires-python = ">=3.9"
# dependencies = [
#     "fastembed",
# ]
# ///
"""
Three-stage dedup filter for .compass/learnings/*.md (gap B2 -- gentle, non-blocking).

Stage 1 (exact hash) and stage 2 (lexical Jaccard) are fully deterministic, no model.
Stage 3 (embedding similarity) runs whenever stage 1/2 didn't already produce a confident
verdict (exact hash or high lexical overlap) -- including when the lexical score is low,
since a genuine paraphrase can share almost no vocabulary with the learning it duplicates.
Only the exact/lexical-high short-circuits skip the model; everything else falls through to
it. Cosine is plain Python (no numpy): the core logic here has zero external dependencies,
so it stays testable without fastembed installed. Only the real embedder (invoked via
`uv run`, wired in main()) needs it.

Usage: check_similarity.py <compass_dir> <candidate_text_or_@file>
Prints one of: DUPLICATE <file> exact:1.00 | POSSIBLE_DUPLICATE <file> lexical:<score> |
POSSIBLE_DUPLICATE <file> embedding:<score> | OK | UNAVAILABLE: <reason>
Always exits 0 -- this is a filter, not a gate.
"""

import hashlib
import json
import math
import pathlib
import re
import sys
from dataclasses import dataclass
from typing import Callable, Optional

LEXICAL_HIGH = 0.6   # >= this: confident lexical duplicate, stop
EMBEDDING_THRESHOLD = 0.80  # empirical near-duplicate cutoff for MiniLM on short text
MODEL_NAME = "sentence-transformers/all-MiniLM-L6-v2"


@dataclass
class Finding:
    file: str
    stage: str  # "exact" | "lexical" | "embedding"
    score: float


# --- stage 0: normalization -----------------------------------------------------

def normalize(text: str) -> str:
    text = text.lower()
    text = re.sub(r"^#+\s*", "", text, flags=re.MULTILINE)      # heading markers
    text = re.sub(r"\*\*|\*|__|_|`", "", text)                    # emphasis/code markers
    text = re.sub(r"\[([^\]]*)\]\([^)]*\)", r"\1", text)          # [text](link) -> text
    text = re.sub(r"\s+", " ", text).strip()
    return text


def content_hash(norm_text: str) -> str:
    return hashlib.sha256(norm_text.encode("utf-8")).hexdigest()


# --- stage 1: exact hash ---------------------------------------------------------

def hash_match(candidate_norm: str, existing_norm: list) -> Optional[Finding]:
    target = content_hash(candidate_norm)
    for file, norm_text in existing_norm:
        if content_hash(norm_text) == target:
            return Finding(file, "exact", 1.0)
    return None


# --- stage 2: lexical overlap (Jaccard on word shingles) -------------------------

def shingles(text: str, n: int = 3) -> set:
    words = text.split()
    if not words:
        return set()
    if len(words) < n:
        return {tuple(words)}
    return {tuple(words[i:i + n]) for i in range(len(words) - n + 1)}


def jaccard(a: set, b: set) -> float:
    if not a or not b:
        return 0.0
    return len(a & b) / len(a | b)


def lexical_score(a_norm: str, b_norm: str) -> float:
    return jaccard(shingles(a_norm), shingles(b_norm))


# --- stage 3: embedding similarity (only reached when stage 2 is ambiguous) ------

def cosine(a: list, b: list) -> float:
    dot = sum(x * y for x, y in zip(a, b))
    norm_a = math.sqrt(sum(x * x for x in a))
    norm_b = math.sqrt(sum(y * y for y in b))
    denom = norm_a * norm_b
    return dot / denom if denom else 0.0


def get_vectors(texts: list, cache: dict, embed_fn: Callable[[list], list]) -> list:
    """Embeds only texts whose content hash isn't already in `cache`; mutates `cache`."""
    hashes = [content_hash(t) for t in texts]
    missing_idx = [i for i, h in enumerate(hashes) if h not in cache]
    if missing_idx:
        new_vectors = embed_fn([texts[i] for i in missing_idx])
        for i, vec in zip(missing_idx, new_vectors):
            cache[hashes[i]] = vec
    return [cache[h] for h in hashes]


# --- orchestration -----------------------------------------------------------------

def find_duplicates(
    candidate_text: str,
    existing: list,
    embed_fn: Optional[Callable[[list], list]] = None,
    cache: Optional[dict] = None,
):
    """Returns (finding_or_None, stage3_needed_but_unavailable: bool).

    Stage 3 runs (and the second element is always False) whenever `embed_fn` is given and
    stage 1/2 didn't already produce a confident verdict -- a low lexical score does NOT
    skip it, since a true paraphrase can share almost no vocabulary with the learning it
    duplicates. If no `embed_fn` was passed and stage 1/2 wasn't confident, the caller gets
    (None, True) so it can decide whether to fetch one (e.g. load the real model) rather than
    this function silently skipping the hard case.
    """
    candidate_norm = normalize(candidate_text)
    existing_norm = [(f, normalize(t)) for f, t in existing]

    finding = hash_match(candidate_norm, existing_norm)
    if finding is not None:
        return finding, False

    if not existing_norm:
        return None, False

    best_file, best_score = None, 0.0
    for file, norm_text in existing_norm:
        score = lexical_score(candidate_norm, norm_text)
        if score > best_score:
            best_file, best_score = file, score

    if best_score >= LEXICAL_HIGH:
        return Finding(best_file, "lexical", best_score), False

    if embed_fn is None:
        return None, True

    if cache is None:
        cache = {}
    texts = [candidate_norm] + [t for _, t in existing_norm]
    vectors = get_vectors(texts, cache, embed_fn)
    candidate_vec = vectors[0]

    best_file, best_score = None, -1.0
    for (file, _), vec in zip(existing_norm, vectors[1:]):
        score = cosine(candidate_vec, vec)
        if score > best_score:
            best_file, best_score = file, score

    if best_score >= EMBEDDING_THRESHOLD:
        return Finding(best_file, "embedding", best_score), False
    return None, False


# --- CLI ------------------------------------------------------------------------

def read_learning_text(path: pathlib.Path) -> str:
    lines = path.read_text(encoding="utf-8").splitlines()
    body_lines = [l for l in lines if not re.match(r"^Date:\s", l)]
    return "\n".join(body_lines)


def load_cache(cache_path: pathlib.Path) -> dict:
    if cache_path.exists():
        try:
            return json.loads(cache_path.read_text())
        except (json.JSONDecodeError, OSError):
            return {}
    return {}


def save_cache(cache_path: pathlib.Path, cache: dict) -> None:
    try:
        cache_path.parent.mkdir(parents=True, exist_ok=True)
        cache_path.write_text(json.dumps(cache))
    except OSError:
        pass


def real_embed_fn(texts: list) -> list:
    from fastembed import TextEmbedding

    model = TextEmbedding(model_name=MODEL_NAME)
    return [vec.tolist() for vec in model.embed(texts)]


def main(argv: list) -> int:
    if len(argv) < 3:
        print("Usage: check_similarity.py <compass_dir> <candidate_text_or_@file>", file=sys.stderr)
        return 2

    compass_dir = pathlib.Path(argv[1])
    candidate_arg = argv[2]
    candidate_text = (
        pathlib.Path(candidate_arg[1:]).read_text(encoding="utf-8")
        if candidate_arg.startswith("@")
        else candidate_arg
    )

    learnings_dir = compass_dir / "learnings"
    existing = []
    if learnings_dir.is_dir():
        for f in sorted(learnings_dir.glob("*.md")):
            if f.name == "archive.md":
                continue
            existing.append((f.name, read_learning_text(f)))

    finding, needs_stage3 = find_duplicates(candidate_text, existing, embed_fn=None)

    if needs_stage3:
        cache_path = learnings_dir / ".embedding-cache.json"
        try:
            cache = load_cache(cache_path)
            finding, _ = find_duplicates(candidate_text, existing, embed_fn=real_embed_fn, cache=cache)
            save_cache(cache_path, cache)
        except Exception as exc:  # no uv-installed fastembed, no network, timeout, etc.
            print(f"UNAVAILABLE: {type(exc).__name__}: {exc}")
            return 0

    if finding is not None:
        label = "DUPLICATE" if finding.stage == "exact" else "POSSIBLE_DUPLICATE"
        print(f"{label} {finding.file} {finding.stage}:{finding.score:.2f}")
    else:
        print("OK")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
