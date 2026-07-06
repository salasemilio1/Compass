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


def test_low_lexical_score_still_reaches_embedder_without_embedder_reports_needs_stage3(monkeypatch):
    monkeypatch.setattr(cs, "lexical_score", lambda a, b: 0.05)
    existing = [("a.md", "some unrelated learning")]
    finding, needs_stage3 = cs.find_duplicates("totally different candidate", existing, embed_fn=None)
    assert finding is None
    assert needs_stage3 is True, (
        "a low lexical score must not be treated as confidently novel -- only exact/lexical-high "
        "verdicts may skip the model; everything else, however low-scoring, needs stage 3"
    )


def test_low_lexical_score_escalates_to_embedder_and_can_still_be_a_duplicate(monkeypatch):
    monkeypatch.setattr(cs, "lexical_score", lambda a, b: 0.05)
    candidate_norm = cs.normalize("totally different candidate")
    existing_text_norm = cs.normalize("some unrelated learning")
    embedder = FakeEmbedder({candidate_norm: [1.0, 0.0], existing_text_norm: [1.0, 0.0]})
    finding, needs_stage3 = cs.find_duplicates(
        "totally different candidate", [("a.md", "some unrelated learning")], embed_fn=embedder
    )
    assert needs_stage3 is False
    assert finding is not None and finding.stage == "embedding"
    assert embedder.calls, "a near-zero lexical score must still invoke the model, not skip it"


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


# --- calibration: hard cases mined from this repo's real .compass/learnings/* -----
#
# These candidates are checked against the actual current content of .compass/learnings/,
# not synthetic fixtures, per decisions/0002's framing of dedup as a genuine judgment call.
# Stage 3 now runs on every candidate that stage 1/2 didn't already confidently resolve --
# there is no more "confidently novel, skip the model" band -- so each test asserts either
# a confident stage-2 verdict (score >= LEXICAL_HIGH) or that the score falls below it and
# stage 3 is the one that must supply the real answer, with a fake embedder set to the
# *correct* ground-truth similarity for that pair, not whatever value would make the
# assertion pass. If a real learning's wording changes enough to cross LEXICAL_HIGH, that's
# a live tripwire meant to prompt re-tuning, not a test to silence.

COMPASS_LEARNINGS = REPO_ROOT / ".compass" / "learnings"


def _real_learning(name: str) -> str:
    return cs.read_learning_text(COMPASS_LEARNINGS / name)


def test_reformatted_duplicate_caught_at_stage1_exact_hash():
    """Same learning, only markdown formatting differs (heading level, bold, blank lines) --
    must be caught by the hash stage, never even reach lexical/embedding."""
    original = _real_learning("bash-pipefail-empty-substitution.md")
    reformatted = """## Guard command substitutions that may legitimately return empty under `set -euo pipefail`

With **pipefail** set, `var=$(grep ... | awk ...)` exits the whole script if `grep` finds nothing --
even though `awk` on empty input exits 0 -- because the assignment's exit status is the pipeline's,
and pipefail makes that the *first* nonzero exit in the pipe, not the last command's. Hit this writing
the `Commit:` staleness check in `hooks/compass-session-start.sh`. Guard with `|| true` on the whole
assignment whenever "not found" is an expected, non-error outcome.
"""
    embedder = FakeEmbedder({})
    finding, needs_stage3 = cs.find_duplicates(
        reformatted, [("bash-pipefail-empty-substitution.md", original)], embed_fn=embedder
    )
    assert finding is not None and finding.stage == "exact"
    assert needs_stage3 is False
    assert embedder.calls == [], "reformatting-only duplicate should never reach the model"


def test_near_verbatim_reuse_with_changed_tail_caught_at_stage2_lexical():
    """A near-copy of graphify-cli-quirks.md that keeps both bullets verbatim but rewrites the
    final clause to a materially different, more severe conclusion (leaks a paid cloud call vs.
    picks the wrong local backend). High lexical overlap (~0.61, just over LEXICAL_HIGH) means
    this is flagged as a lexical duplicate even though the tail carries new information --
    documenting that heavy verbatim reuse wins the vote even when the ending changed."""
    original = _real_learning("graphify-cli-quirks.md")
    near_copy = """graphify CLI quirks (for if the semantic layer is ever revisited)

Not used by compass's current wiring (deterministic-only, see decisions/0001), but worth recording since
we hit both directly:

- The Ollama backend needs `uv tool install "graphifyy[ollama]" --force` -- a plain `graphifyy` install
  fails on `--backend ollama` with a missing `openai` package error, even though Ollama itself needs no
  such package conceptually.
- Flag syntax is inconsistent across subcommands: `extract`/`update` take `--backend <name>`
  (space-separated); `cluster-only`/`label` require `--backend=<name>` (equals-sign only). Using the
  wrong form doesn't error clearly -- but this time the silent fallback landed on the *cloud* API
  instead of auto-detect, which is a materially worse failure mode since it can leak a paid API call
  instead of just picking the wrong local backend."""
    score = cs.lexical_score(cs.normalize(near_copy), cs.normalize(original))
    assert score >= cs.LEXICAL_HIGH, f"calibration assumption drifted: score={score:.3f}"

    embedder = FakeEmbedder({})
    finding, needs_stage3 = cs.find_duplicates(
        near_copy, [("graphify-cli-quirks.md", original)], embed_fn=embedder
    )
    assert finding is not None and finding.stage == "lexical"
    assert needs_stage3 is False
    assert embedder.calls == []


def test_shared_domain_jargon_but_distinct_topic_still_reaches_and_clears_stage3():
    """A candidate about install.sh's --dev guard reuses graphify-cli-quirks.md's exact
    vocabulary (flag, backend, graphify, install, silently falls back) but describes an
    unrelated fact. Lexical score is near-zero (3-word-shingle Jaccard needs consecutive
    phrase overlap, not just shared words), but under the always-run-stage-3 policy a low
    lexical score no longer skips the model -- it still has to be embedded and correctly
    judged unrelated. Ground truth here is genuinely distinct, so the fake embedder is given
    a low-similarity pair: the correct outcome is OK, reached via the model, not skipped
    before it had a chance to look."""
    original = _real_learning("graphify-cli-quirks.md")
    candidate = """install.sh's canonical-source guard requires an explicit --dev flag before it will symlink global
skills from a non-canonical checkout. Unlike graphify's --backend flag (space or equals-sign depending
on subcommand), --dev takes no argument -- omitting the guard silently falls back to refusing the
install rather than auto-detecting a default, so the flag syntax mismatch risk here is different from
graphify's backend flag inconsistency."""
    score = cs.lexical_score(cs.normalize(candidate), cs.normalize(original))
    assert score < cs.LEXICAL_HIGH, f"calibration assumption drifted: score={score:.3f}"

    embedder = FakeEmbedder({
        cs.normalize(candidate): [1.0, 0.0],
        cs.normalize(original): [0.1, 0.99],
    })
    finding, needs_stage3 = cs.find_duplicates(
        candidate, [("graphify-cli-quirks.md", original)], embed_fn=embedder
    )
    assert needs_stage3 is False
    assert finding is None
    assert embedder.calls, "low lexical score must still reach the model under always-run-stage-3"


def test_substring_containment_lands_in_ambiguous_band_not_stage1_or_confident_stage2():
    """Candidate is an exact, verbatim excerpt of verify-flagged-installs-directly.md's
    'How to apply' sentence -- 100% contained in the original, zero novel words. Because
    Jaccard is normalized by the *union* and the original document is much longer than the
    excerpt, containment alone doesn't reach LEXICAL_HIGH (~0.26, well under 0.6): a fully
    quoted excerpt of an existing learning is NOT guaranteed to be caught deterministically --
    it lands in the ambiguous band and depends on stage 3 to be rescued."""
    original = _real_learning("verify-flagged-installs-directly.md")
    excerpt = (
        "when an install is flagged, check the package registry's own api for ownership/url matches "
        "before deciding -- don't take the project's own docs as sufficient verification, and don't "
        "dismiss the flag either."
    )
    score = cs.lexical_score(cs.normalize(excerpt), cs.normalize(original))
    assert score < cs.LEXICAL_HIGH, f"calibration assumption drifted: score={score:.3f}"

    finding, needs_stage3 = cs.find_duplicates(
        excerpt, [("verify-flagged-installs-directly.md", original)], embed_fn=None
    )
    assert finding is None
    assert needs_stage3 is True, "a verbatim excerpt shorter than its source must not be silently OK'd"

    embedder = FakeEmbedder({
        cs.normalize(excerpt): [1.0, 0.0],
        cs.normalize(original): [0.99, 0.02],
    })
    finding, needs_stage3 = cs.find_duplicates(
        excerpt, [("verify-flagged-installs-directly.md", original)], embed_fn=embedder
    )
    assert needs_stage3 is False
    assert finding is not None and finding.stage == "embedding", (
        "stage 3 must be the one to catch a verbatim excerpt that stage 2 only left ambiguous"
    )


def test_generalized_restatement_of_same_fact_lands_in_ambiguous_band_and_is_caught():
    """Candidate restates bash-pipefail-empty-substitution.md's exact gotcha in generic terms
    (grep|awk -> cmd1|cmd2, the specific hook -> 'a different staleness check') -- same
    learning, deliberately genericized. Lexical score (~0.44) sits in the ambiguous band
    despite being conceptually identical; stage 3 has to do the real work here."""
    original = _real_learning("bash-pipefail-empty-substitution.md")
    generalized = """Guard command substitutions that may legitimately return empty under `set -euo pipefail`.

With pipefail enabled, `x=$(cmd1 | cmd2)` can exit the whole script if `cmd1` finds nothing -- even
though `cmd2` on empty input exits 0 -- because the assignment's exit status is the pipeline's, and
pipefail makes that the first nonzero exit in the pipe, not the last command's. Hit this in a different
staleness check. Guard with `|| true` on the whole assignment whenever an empty result is expected."""
    score = cs.lexical_score(cs.normalize(generalized), cs.normalize(original))
    assert score < cs.LEXICAL_HIGH, f"calibration assumption drifted: score={score:.3f}"

    embedder = FakeEmbedder({
        cs.normalize(generalized): [1.0, 0.0, 0.0],
        cs.normalize(original): [0.95, 0.05, 0.0],
    })
    finding, needs_stage3 = cs.find_duplicates(
        generalized, [("bash-pipefail-empty-substitution.md", original)], embed_fn=embedder
    )
    assert needs_stage3 is False
    assert finding is not None and finding.stage == "embedding"


def test_verbatim_setup_with_reversed_conclusion_is_a_genuine_judgment_call():
    """Candidate opens with bash-pipefail-empty-substitution.md's exact problem description,
    then argues AGAINST that learning's own fix (blanket `|| true` also swallows real `awk`
    errors and permission failures). Same setup, opposite advice -- lands ambiguous (~0.40)
    on lexical grounds alone. This is a genuine judgment call: an embedder that scores it
    below the duplicate threshold is *correct* to let it through as a distinct (corrective)
    learning, not a duplicate to be suppressed."""
    original = _real_learning("bash-pipefail-empty-substitution.md")
    correction = """guard command substitutions that may legitimately return empty under `set -euo pipefail` -- but not
with a blanket `|| true`

With pipefail set, `var=$(grep ... | awk ...)` exits the whole script if `grep` finds nothing --
even though `awk` on empty input exits 0 -- because the assignment's exit status is the pipeline's,
and pipefail makes that the first nonzero exit in the pipe, not the last command's. The fix is not a
blanket `|| true` on the whole assignment, though -- that also silences a genuine `awk` syntax error or
a permissions failure on the file `grep` reads, which is a real bug you want the script to stop on.
Check `$?` against the specific expected-empty case instead of suppressing the whole pipeline's exit
status generically."""
    score = cs.lexical_score(cs.normalize(correction), cs.normalize(original))
    assert score < cs.LEXICAL_HIGH, f"calibration assumption drifted: score={score:.3f}"

    embedder = FakeEmbedder({
        cs.normalize(correction): [1.0, 0.0],
        cs.normalize(original): [0.3, 0.95],
    })
    finding, needs_stage3 = cs.find_duplicates(
        correction, [("bash-pipefail-empty-substitution.md", original)], embed_fn=embedder
    )
    assert needs_stage3 is False
    assert finding is None, "a corrective counter-learning must not be suppressed as a duplicate"


def test_zero_lexical_overlap_paraphrase_is_now_caught_by_always_running_stage3():
    """True paraphrase of verify-flagged-installs-directly.md with essentially no shared
    vocabulary (dependency/self-description/cross-reference/metadata-endpoint vs.
    install/flagged/PyPI/typosquat/JSON-API). Lexical score is 0.0. Under the old
    confidently-novel short-circuit this slipped through silently, never reaching the model
    -- the residual gap decisions/0002 called out. Under always-run-stage-3, a 0.0 lexical
    score no longer excuses skipping the model: ground truth here is a real duplicate, so a
    correctly-behaving embedder must be given the chance to catch it, and this test asserts
    that it does."""
    original = _real_learning("verify-flagged-installs-directly.md")
    paraphrase = """When a dependency trips a suspicious-package warning, don't lean on the repo's own self
description -- cross-reference the hosting registry's official metadata endpoint to confirm the listed
maintainer and source link genuinely line up before extending trust."""
    score = cs.lexical_score(cs.normalize(paraphrase), cs.normalize(original))
    assert score < cs.LEXICAL_HIGH, f"calibration assumption drifted: score={score:.3f}"

    embedder = FakeEmbedder({
        cs.normalize(paraphrase): [1.0, 0.0],
        cs.normalize(original): [0.97, 0.24],
    })
    finding, needs_stage3 = cs.find_duplicates(
        paraphrase, [("verify-flagged-installs-directly.md", original)], embed_fn=embedder
    )
    assert needs_stage3 is False
    assert finding is not None and finding.stage == "embedding", (
        "the gap is closed: a same-meaning paraphrase with zero shared 3-grams must still reach "
        "and be caught by the model, not dismissed before it gets a look"
    )
    assert embedder.calls, "a 0.0 lexical score must not be treated as confidently novel anymore"


def test_same_real_incident_across_two_real_documents_is_now_caught():
    """Not synthetic at all: context/current-state.md's own 'In flight' bullet about the
    unblocked .git/hooks/pre-commit backstop describes the *exact same real incident* as
    plan-approval-does-not-blanket-authorize-actions.md's generalized learning -- both already
    exist in this repo, about the same event, written in unrelated vocabulary (terse
    status-update prose vs. generalized-lesson prose). Lexical score is ~0.05. Before
    always-run-stage-3, this real pair would never have reached the model if one were
    re-submitted as a new learning candidate today. Now it must -- and a correctly-behaving
    embedder (ground truth: same incident) should flag it."""
    learning = _real_learning("plan-approval-does-not-blanket-authorize-actions.md")
    current_state_bullet = """This repo's own local .git/hooks/pre-commit backstop is not yet installed -- the auto-mode
permission classifier blocked writing it during the hardening session, even though it was named in
the approved plan. Needs an explicit go-ahead."""
    score = cs.lexical_score(cs.normalize(current_state_bullet), cs.normalize(learning))
    assert score < cs.LEXICAL_HIGH, f"calibration assumption drifted: score={score:.3f}"

    embedder = FakeEmbedder({
        cs.normalize(current_state_bullet): [1.0, 0.0],
        cs.normalize(learning): [0.95, 0.31],
    })
    finding, needs_stage3 = cs.find_duplicates(
        current_state_bullet, [("plan-approval-does-not-blanket-authorize-actions.md", learning)],
        embed_fn=embedder,
    )
    assert needs_stage3 is False
    assert finding is not None and finding.stage == "embedding"
    assert embedder.calls, "same real incident, near-zero lexical overlap -- must still reach the model"


def test_cosmetic_rename_of_real_learning_caught_cleanly_at_stage2():
    """Baseline contrast to the changed-tail case above: test-helper-naming-must-avoid-test-glob.md
    with only the example filename swapped (test_helpers.sh -> test_utils.sh) and no content
    change otherwise. Score (~0.69) clears LEXICAL_HIGH comfortably -- confirming stage 2 is not
    just barely scraping by on the trickier cases elsewhere in this file; a genuinely clean
    near-duplicate is caught with headroom to spare."""
    original = _real_learning("test-helper-naming-must-avoid-test-glob.md")
    renamed = """Name shared test helpers outside the `test_*.sh` glob.

`tests/run_tests.sh` discovers tests via the `test_*.sh` pattern. A shared assertion-helper file named
`test_utils.sh` matched that glob and was "run" as its own test -- sourcing it standalone has no
assertions, so it silently reported PASS with zero checks, no error. Renamed to `utils.sh`. Any
future shared bash test fixture/helper file must avoid the `test_*.sh` prefix, or the runner will
silently treat it as a passing test."""
    score = cs.lexical_score(cs.normalize(renamed), cs.normalize(original))
    assert score >= cs.LEXICAL_HIGH, f"calibration assumption drifted: score={score:.3f}"

    embedder = FakeEmbedder({})
    finding, needs_stage3 = cs.find_duplicates(
        renamed, [("test-helper-naming-must-avoid-test-glob.md", original)], embed_fn=embedder
    )
    assert finding is not None and finding.stage == "lexical"
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
