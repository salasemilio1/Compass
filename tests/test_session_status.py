"""Tests for tools/session-context/session_status.py.

Run via: pytest tests/test_session_status.py
Pure computation on synthetic transcript data -- no real transcript, no network.
"""

import importlib.util
import json
import pathlib

import pytest

REPO_ROOT = pathlib.Path(__file__).resolve().parents[1]
MODULE_PATH = REPO_ROOT / "tools" / "session-context" / "session_status.py"

_spec = importlib.util.spec_from_file_location("session_status", MODULE_PATH)
ss = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(ss)


def _assistant_line(input_tokens, cache_creation, cache_read, output_tokens=10, sidechain=False):
    entry = {
        "type": "assistant",
        "timestamp": "2026-07-07T00:00:00Z",
        "message": {
            "usage": {
                "input_tokens": input_tokens,
                "cache_creation_input_tokens": cache_creation,
                "cache_read_input_tokens": cache_read,
                "output_tokens": output_tokens,
            }
        },
    }
    if sidechain:
        entry["isSidechain"] = True
    return entry


def write_transcript(tmp_path, entries):
    path = tmp_path / "session.jsonl"
    with open(path, "w") as f:
        for e in entries:
            f.write(json.dumps(e) + "\n")
    return path


def test_load_assistant_turns_extracts_usage(tmp_path):
    path = write_transcript(tmp_path, [
        _assistant_line(10, 0, 1000),
        {"type": "user", "message": {}},
        _assistant_line(10, 0, 2000),
    ])
    turns = ss.load_assistant_turns(path)
    assert len(turns) == 2
    assert ss.context_total(turns[-1]) == 2010


def test_load_assistant_turns_excludes_sidechain(tmp_path):
    path = write_transcript(tmp_path, [
        _assistant_line(10, 0, 1000),
        _assistant_line(10, 0, 50000, sidechain=True),
        _assistant_line(10, 0, 2000),
    ])
    turns = ss.load_assistant_turns(path)
    assert len(turns) == 2
    assert ss.context_total(turns[-1]) == 2010


def test_load_assistant_turns_skips_entries_without_usage(tmp_path):
    path = write_transcript(tmp_path, [
        {"type": "assistant", "message": {}},
        _assistant_line(10, 0, 1000),
    ])
    turns = ss.load_assistant_turns(path)
    assert len(turns) == 1


def test_compute_status_reports_used_remaining_percent():
    turns = [{"input_tokens": 0, "cache_creation_input_tokens": 0,
              "cache_read_input_tokens": 50000, "output_tokens": 0}]
    status = ss.compute_status(turns, context_window=200_000)
    assert status["raw"]["used_tokens"] == 50000
    assert status["raw"]["remaining_tokens"] == 150000
    assert status["raw"]["percent_used"] == 25.0


def test_compute_status_empty_turns_reports_error():
    status = ss.compute_status([], context_window=200_000)
    assert "error" in status


def test_compute_status_growth_rate_uses_trailing_window_only():
    # 10 turns growing by 1000 each, then a final turn that jumps by 100000 --
    # the trailing-window average should reflect the recent jump, not be diluted
    # by the whole session's history.
    turns = []
    total = 0
    for _ in range(10):
        total += 1000
        turns.append({"input_tokens": 0, "cache_creation_input_tokens": 0,
                       "cache_read_input_tokens": total, "output_tokens": 0})
    total += 100_000
    turns.append({"input_tokens": 0, "cache_creation_input_tokens": 0,
                  "cache_read_input_tokens": total, "output_tokens": 0})

    status = ss.compute_status(turns, context_window=200_000)
    # trailing window = 5 turns -> 5 deltas: four 1000s and one 100000 jump
    expected_avg = (1000 * 4 + 100_000) / 5
    assert status["derived"]["avg_growth_per_turn_last_n"] == pytest.approx(expected_avg, abs=0.1)


def test_compute_status_no_estimate_when_growth_is_zero():
    turns = [{"input_tokens": 0, "cache_creation_input_tokens": 0,
              "cache_read_input_tokens": 1000, "output_tokens": 0}] * 3
    status = ss.compute_status(turns, context_window=200_000)
    assert status["derived"]["avg_growth_per_turn_last_n"] == 0
    assert status["derived"]["estimated_turns_remaining"] is None


def test_resolve_transcript_path_missing_env_returns_none(monkeypatch):
    monkeypatch.delenv("CLAUDE_CODE_SESSION_ID", raising=False)
    assert ss.resolve_transcript_path() is None
