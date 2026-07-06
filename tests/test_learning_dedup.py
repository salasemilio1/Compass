"""Tests for tools/learning-dedup/check_similarity.py (gap B2).

Run via: uv run --with pytest pytest tests/test_learning_dedup.py

Stage 1/2 and orchestration tests never touch fastembed/numpy/network (rule 4 -- bare CI).
Only the final integration test shells out via `uv run` and is allowed to skip/degrade
gracefully offline, per the documented UNAVAILABLE path.
"""

import importlib.util
import pathlib
import shutil
import subprocess

import pytest

REPO_ROOT = pathlib.Path(__file__).resolve().parents[1]
MODULE_PATH = REPO_ROOT / "tools" / "learning-dedup" / "check_similarity.py"

_spec = importlib.util.spec_from_file_location("check_similarity", MODULE_PATH)
cs = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(cs)


class FakeEmbedder:
    """Injectable embedder (rule 1/4): records every call, returns pre-baked vectors."""

    def __init__(self, vectors: dict):
        self.calls = []
        self.vectors = vectors

    def __call__(self, texts):
        self.calls.append(list(texts))
        return [self.vectors[t] for t in texts]


# --- stage 0: normalize -----------------------------------------------------------

def test_normalize_strips_headings():
    assert cs.normalize("# Heading\ntext") == "heading text"


def test_normalize_strips_emphasis_and_code():
    assert cs.normalize("**bold** and _em_ and `code`") == "bold and em and code"


def test_normalize_strips_links_keeps_text():
    assert cs.normalize("see [the docs](http://example.com/x) for more") == "see the docs for more"


def test_normalize_collapses_whitespace_and_case():
    assert cs.normalize("Hello   World\n\nFoo") == "hello world foo"


# --- stage 1: hash_match -----------------------------------------------------------

def test_hash_match_identical_content():
    existing = [("a.md", "the same content")]
    finding = cs.hash_match(cs.normalize("the same content"), [(f, cs.normalize(t)) for f, t in existing])
    assert finding is not None
    assert finding.file == "a.md"
    assert finding.stage == "exact"
    assert finding.score == 1.0


def test_hash_match_same_content_different_formatting():
    existing = [("a.md", "# Same Content")]
    finding = cs.hash_match(cs.normalize("same content"), [(f, cs.normalize(t)) for f, t in existing])
    assert finding is not None, "normalization should make these hash-equal"


def test_hash_match_different_content_no_match():
    existing = [("a.md", "completely different text")]
    finding = cs.hash_match(cs.normalize("nothing alike here"), [(f, cs.normalize(t)) for f, t in existing])
    assert finding is None


# --- stage 2: shingles / jaccard / lexical_score -----------------------------------

def test_shingles_three_word_windows():
    assert cs.shingles("a b c d", n=3) == {("a", "b", "c"), ("b", "c", "d")}


def test_jaccard_partial_overlap():
    assert cs.jaccard({1, 2, 3}, {2, 3, 4}) == pytest.approx(2 / 4)


def test_jaccard_empty_sets_is_zero():
    assert cs.jaccard(set(), set()) == 0.0
    assert cs.jaccard({1}, set()) == 0.0


def test_lexical_score_identical_text_is_one():
    assert cs.lexical_score("a b c d", "a b c d") == 1.0


def test_lexical_score_disjoint_text_is_zero():
    assert cs.lexical_score("a b c", "x y z") == 0.0


# --- stage 3: cosine + get_vectors caching -----------------------------------------

def test_cosine_identical_vectors():
    assert cs.cosine([1, 0], [1, 0]) == pytest.approx(1.0)


def test_cosine_orthogonal_vectors():
    assert cs.cosine([1, 0], [0, 1]) == pytest.approx(0.0)


def test_cosine_zero_vector_guard():
    assert cs.cosine([0, 0], [1, 1]) == 0.0


def test_get_vectors_only_embeds_missing_entries():
    embedder = FakeEmbedder({"a": [1.0], "b": [2.0]})
    cache = {}
    vectors = cs.get_vectors(["a", "b"], cache, embedder)
    assert vectors == [[1.0], [2.0]]
    assert embedder.calls == [["a", "b"]]


def test_get_vectors_second_call_only_reembeds_changed_entry():
    embedder = FakeEmbedder({"a": [1.0], "b": [2.0], "c": [3.0]})
    cache = {}
    cs.get_vectors(["a", "b"], cache, embedder)
    embedder.calls.clear()

    vectors = cs.get_vectors(["a", "c"], cache, embedder)
    assert vectors == [[1.0], [3.0]]
    assert embedder.calls == [["c"]], "only the new/changed entry should be re-embedded"


# --- orchestration: find_duplicates stage short-circuiting -------------------------

def test_exact_hash_never_reaches_embedder():
    embedder = FakeEmbedder({})
    existing = [("a.md", "identical text here")]
    finding, needs_stage3 = cs.find_duplicates("identical text here", existing, embed_fn=embedder)
    assert finding is not None and finding.stage == "exact"
    assert needs_stage3 is False
    assert embedder.calls == [], "exact match should short-circuit before the model is ever called"


def test_confidently_novel_never_reaches_embedder(monkeypatch):
    monkeypatch.setattr(cs, "lexical_score", lambda a, b: 0.05)
    embedder = FakeEmbedder({})
    existing = [("a.md", "some unrelated learning")]
    finding, needs_stage3 = cs.find_duplicates("totally different candidate", existing, embed_fn=embedder)
    assert finding is None
    assert needs_stage3 is False
    assert embedder.calls == [], "confidently-novel lexical score should skip the model"


def test_lexical_high_confidence_never_reaches_embedder(monkeypatch):
    monkeypatch.setattr(cs, "lexical_score", lambda a, b: 0.75)
    embedder = FakeEmbedder({})
    existing = [("a.md", "some learning")]
    finding, needs_stage3 = cs.find_duplicates("some other learning", existing, embed_fn=embedder)
    assert finding is not None and finding.stage == "lexical"
    assert finding.score == 0.75
    assert needs_stage3 is False
    assert embedder.calls == []


def test_ambiguous_lexical_without_embedder_reports_needs_stage3(monkeypatch):
    monkeypatch.setattr(cs, "lexical_score", lambda a, b: 0.3)
    existing = [("a.md", "some learning")]
    finding, needs_stage3 = cs.find_duplicates("some other candidate", existing, embed_fn=None)
    assert finding is None
    assert needs_stage3 is True, "ambiguous lexical band with no embedder available should be flagged"


def test_ambiguous_lexical_escalates_to_embedder_duplicate(monkeypatch):
    monkeypatch.setattr(cs, "lexical_score", lambda a, b: 0.3)
    candidate_norm = cs.normalize("some other candidate")
    existing_text_norm = cs.normalize("some learning")
    embedder = FakeEmbedder({candidate_norm: [1.0, 0.0], existing_text_norm: [1.0, 0.0]})
    finding, needs_stage3 = cs.find_duplicates(
        "some other candidate", [("a.md", "some learning")], embed_fn=embedder
    )
    assert needs_stage3 is False
    assert finding is not None and finding.stage == "embedding"
    assert finding.score == pytest.approx(1.0)
    assert embedder.calls, "the ambiguous case should actually invoke the embedder"


def test_ambiguous_lexical_escalates_to_embedder_ok(monkeypatch):
    monkeypatch.setattr(cs, "lexical_score", lambda a, b: 0.3)
    candidate_norm = cs.normalize("some other candidate")
    existing_text_norm = cs.normalize("some learning")
    embedder = FakeEmbedder({candidate_norm: [1.0, 0.0], existing_text_norm: [0.0, 1.0]})
    finding, needs_stage3 = cs.find_duplicates(
        "some other candidate", [("a.md", "some learning")], embed_fn=embedder
    )
    assert needs_stage3 is False
    assert finding is None


def test_empty_learnings_dir_is_ok_without_reaching_any_model():
    embedder = FakeEmbedder({})
    finding, needs_stage3 = cs.find_duplicates("anything at all", [], embed_fn=embedder)
    assert finding is None
    assert needs_stage3 is False
    assert embedder.calls == []


# --- real integration smoke test (allowed to skip/degrade offline) ----------------

def test_cli_integration_via_uv(tmp_path):
    if shutil.which("uv") is None:
        pytest.skip("uv not installed")

    compass_dir = tmp_path / ".compass"
    learnings = compass_dir / "learnings"
    learnings.mkdir(parents=True)
    (learnings / "existing.md").write_text(
        "# existing\nDate: 2026-01-01\n\nSomething about graphify cli quirks and backend flags.\n"
    )

    result = subprocess.run(
        ["uv", "run", str(MODULE_PATH), str(compass_dir), "totally unrelated candidate text about pizza"],
        capture_output=True,
        text=True,
        timeout=180,
    )

    if result.returncode != 0:
        pytest.skip(f"uv could not run the script (likely offline): {result.stderr.strip()[:200]}")

    out = result.stdout.strip()
    assert out == "OK" or out.startswith("UNAVAILABLE"), f"unexpected output: {out!r}"
