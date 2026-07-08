"""Tests for hooks/tool-routing-guard.py (decisions/0005).

Run via: pytest tests/test_tool_routing_guard.py
Pure classification tests plus the marker-file safety valve -- no real Claude Code hook
invocation, no network.
"""

import importlib.util
import json
import pathlib
import subprocess
import sys
import uuid

import pytest

REPO_ROOT = pathlib.Path(__file__).resolve().parents[1]
MODULE_PATH = REPO_ROOT / "hooks" / "tool-routing-guard.py"

_spec = importlib.util.spec_from_file_location("tool_routing_guard", MODULE_PATH)
g = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(g)


# --- classify: allowed (cheap / bounded) commands -----------------------------------

@pytest.mark.parametrize("command", [
    "grep -n 'define' foo.py",
    "find . -maxdepth 2 -name '*.py'",
    "git status",
    "git diff --stat",
    "git log --oneline -20",
    "ls -la",
    "wc -l tests/*.sh",
    "pgrep node",
])
def test_classify_allows_cheap_commands(command):
    assert g.classify(command, graph_exists=False) is None
    assert g.classify(command, graph_exists=True) is None


# --- classify: graphify-first (only when a graph exists) ----------------------------

@pytest.mark.parametrize("command", [
    "grep -rn 'functionName' src/",
    "rg 'TODO'",
    "find . -name '*.py'",
])
def test_classify_routes_broad_search_to_graphify_when_graph_exists(command):
    rule_id, reason = g.classify(command, graph_exists=True)
    assert rule_id == "graphify"
    assert "graphify" in reason


@pytest.mark.parametrize("command", [
    "grep -rn 'functionName' src/",
    "rg 'TODO'",
    "find . -name '*.py'",
])
def test_classify_routes_broad_search_to_ctx_without_graph(command):
    rule_id, reason = g.classify(command, graph_exists=False)
    assert rule_id == "ctx"
    assert "ctx_execute" in reason


# --- classify: ctx-first (heavy shapes regardless of graph) -------------------------

@pytest.mark.parametrize("command", [
    "git diff HEAD~1",
    "git diff --cached",
    "git log",
    "pytest tests/",
    "npm run build",
    "npm test",
    "make",
])
def test_classify_routes_heavy_commands_to_ctx(command):
    rule_id, _ = g.classify(command, graph_exists=False)
    assert rule_id == "ctx"


# --- main(): stdin/stdout contract ---------------------------------------------------

def run_hook(payload: dict) -> dict:
    result = subprocess.run(
        [sys.executable, str(MODULE_PATH)],
        input=json.dumps(payload),
        capture_output=True,
        text=True,
        timeout=5,
    )
    assert result.returncode == 0
    return json.loads(result.stdout) if result.stdout.strip() else None


def test_main_allows_non_bash_tool():
    assert run_hook({"tool_name": "Read", "tool_input": {}, "session_id": "s1", "cwd": "."}) is None


def test_main_allows_cheap_bash_command():
    payload = {"tool_name": "Bash", "tool_input": {"command": "git status"}, "session_id": "s1", "cwd": "."}
    assert run_hook(payload) is None


def test_main_denies_heavy_command_with_reason(tmp_path):
    session_id = str(uuid.uuid4())
    payload = {
        "tool_name": "Bash",
        "tool_input": {"command": "pytest tests/"},
        "session_id": session_id,
        "cwd": str(tmp_path),
    }
    result = run_hook(payload)
    assert result["hookSpecificOutput"]["permissionDecision"] == "deny"
    assert "ctx_execute" in result["hookSpecificOutput"]["permissionDecisionReason"]


def test_main_safety_valve_allows_second_denial_in_same_session(tmp_path):
    session_id = str(uuid.uuid4())
    payload = {
        "tool_name": "Bash",
        "tool_input": {"command": "pytest tests/"},
        "session_id": session_id,
        "cwd": str(tmp_path),
    }
    first = run_hook(payload)
    second = run_hook(payload)
    assert first is not None and first["hookSpecificOutput"]["permissionDecision"] == "deny"
    assert second is None


def test_main_malformed_stdin_fails_open():
    result = subprocess.run(
        [sys.executable, str(MODULE_PATH)],
        input="not json",
        capture_output=True,
        text=True,
        timeout=5,
    )
    assert result.returncode == 0
    assert result.stdout.strip() == ""


def test_main_missing_fields_fails_open():
    payload = {"tool_name": "Bash", "tool_input": {"command": "pytest tests/"}}  # no session_id/cwd
    assert run_hook(payload) is None
