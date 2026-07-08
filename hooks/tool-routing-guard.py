#!/usr/bin/env python3
"""
PreToolUse hook: deterministically routes heavy Bash calls to a cheaper alternative
instead of leaving the choice to agent judgment (decisions/0005).

Two rule sets, checked in priority order, each keyed on the shape of the command string
alone -- no output is inspected, nothing here is a heuristic guess about *this* call's
actual size:

  1. graphify-first -- a repo-wide symbol search (recursive grep/ripgrep, or an
     unrestricted `find`) when a graphify graph already exists for this repo. Denied
     toward `graphify query`.
  2. ctx-first -- a command shape known to produce large output regardless of repo
     content: a full `git diff`, an unbounded `git log`, a test/build runner, or (when no
     graphify graph exists) the same broad grep/find as above. Denied toward `ctx_execute`.

Safety valve: each rule fires at most once per session. A second Bash call matching the
same rule in the same session is allowed through unconditionally -- this is what keeps a
genuinely unavailable ctx_execute/graphify from turning into a dead end (see the audit
finding this closes: decisions/0005). Tracked via a marker file, same mechanism as
hooks/session-context-nudge.py's per-threshold marker.

Never denies on internal error -- printing nothing (allow, deferring to normal permission
flow) is always the fallback, since Claude Code's own hook runtime fails *closed* on a
crash/timeout; this script must not compound that by being the thing that crashes.

Usage: reads a PreToolUse hook payload on stdin, prints a PreToolUse decision on stdout
(or nothing, to allow).
"""
import json
import re
import sys
import tempfile
from pathlib import Path

RECURSIVE_GREP = re.compile(r"\bgrep\b[^|;&]*\s-\w*[rR]\w*(\s|$)")
RIPGREP = re.compile(r"(^|[|;&]\s*)rg\b")
BROAD_FIND = re.compile(r"\bfind\b(?![^|;&]*-maxdepth\b)")
FULL_GIT_DIFF = re.compile(r"\bgit\s+diff\b")
BOUNDED_LOG = re.compile(r"(-n\s*\d+|--oneline|-\d+\b|--max-count)")
UNBOUNDED_GIT_LOG = re.compile(r"\bgit\s+log\b")
TEST_BUILD_RUNNER = re.compile(
    r"\b(pytest|npm\s+(test|run)|yarn\s+test|jest|go\s+test|cargo\s+test|make|mvn|gradle|tox)\b"
)


def is_broad_search(command: str) -> bool:
    return bool(RECURSIVE_GREP.search(command) or RIPGREP.search(command) or BROAD_FIND.search(command))


def classify(command: str, graph_exists: bool) -> "tuple[str, str] | None":
    """Returns (rule_id, reason) if `command` should be denied, else None (allow)."""
    if is_broad_search(command):
        if graph_exists:
            return "graphify", (
                "This looks like a repo-wide symbol search and a graphify graph exists for "
                "this repo -- try `graphify query \"<question>\"` first. If it doesn't answer "
                "it, this exact command is allowed through on retry."
            )
        return "ctx", (
            "This looks like a repo-wide search with unbounded output -- route it through "
            "ctx_execute instead. If ctx_execute isn't available, this exact command is "
            "allowed through on retry."
        )

    if FULL_GIT_DIFF.search(command) and "--stat" not in command:
        return "ctx", (
            "A full `git diff` can be large -- route it through ctx_execute instead (or add "
            "--stat if a summary is enough). Allowed through on retry."
        )

    if UNBOUNDED_GIT_LOG.search(command) and not BOUNDED_LOG.search(command):
        return "ctx", (
            "An unbounded `git log` can be large -- add -n/--oneline, or route it through "
            "ctx_execute. Allowed through on retry."
        )

    if TEST_BUILD_RUNNER.search(command):
        return "ctx", (
            "Test/build output is usually large -- route it through ctx_execute instead. "
            "Allowed through on retry."
        )

    return None


def already_fired(session_id: str, rule_id: str) -> bool:
    marker = Path(tempfile.gettempdir()) / f"claude-tool-routing-{session_id}-{rule_id}"
    if marker.exists():
        return True
    try:
        marker.write_text("1")
    except OSError:
        pass
    return False


def deny(reason: str) -> None:
    print(json.dumps({
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "deny",
            "permissionDecisionReason": reason,
        }
    }))


def main() -> int:
    try:
        payload = json.load(sys.stdin)
        if payload.get("tool_name") != "Bash":
            return 0

        command = payload.get("tool_input", {}).get("command")
        session_id = payload.get("session_id")
        cwd = payload.get("cwd")
        if not command or not session_id or not cwd:
            return 0

        graph_exists = (Path(cwd) / "graphify-out" / "graph.json").is_file()
        finding = classify(command, graph_exists)
        if finding is None:
            return 0

        rule_id, reason = finding
        if already_fired(session_id, rule_id):
            return 0

        deny(reason)
        return 0
    except Exception:
        return 0


if __name__ == "__main__":
    sys.exit(main())
