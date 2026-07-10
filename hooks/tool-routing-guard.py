#!/usr/bin/env python3
"""
PreToolUse hook: deterministically routes heavy Bash calls to a cheaper alternative
instead of leaving the choice to agent judgment (decisions/0005).

One rule set, keyed on the shape of the command string alone -- no output is inspected,
nothing here is a heuristic guess about *this* call's actual size:

  ctx-first -- a command shape known to produce large output regardless of repo content:
  a repo-wide symbol search (recursive grep/ripgrep, or an unrestricted `find`), a full
  `git diff`, an unbounded `git log`, or a test/build runner. Denied toward `ctx_execute`.

(graphify is no longer part of this guard -- decisions/0005 forced repo-wide searches
toward `graphify query` when a graph existed, but a week of that enforcement produced
fewer than 5 real uses. graphify moved to an opt-in add-on; see the decision that
supersedes 0005 for the reasoning.)

Safety valve: the rule fires at most once per session. A second Bash call matching it in
the same session is allowed through unconditionally -- this is what keeps a genuinely
unavailable ctx_execute from turning into a dead end (see the audit finding this closes:
decisions/0005). Tracked via a marker file, same mechanism as
hooks/session-context-nudge.py's per-threshold marker.

Never denies on internal error -- printing nothing (allow, deferring to normal permission
flow) is always the fallback, since Claude Code's own hook runtime fails *closed* on a
crash/timeout; this script must not compound that by being the thing that crashes.

Also never denies a `run_in_background` call: every rule here exists to redirect a call
whose *output* should be captured/summarized elsewhere (ctx_execute), but a backgrounded
command (e.g. `npm run dev` to start a dev server) isn't run to capture a finished result
at all -- routing it through ctx_execute would mean waiting on a process that never exits.
This matters in practice: TEST_BUILD_RUNNER matches any `npm run <script>`, not just
tests, so an ordinary dev-server start would otherwise be denied on first use every
session.

Usage: reads a PreToolUse hook payload on stdin, prints a PreToolUse decision on stdout
(or nothing, to allow).
"""
import json
import os
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


def classify(command: str) -> "tuple[str, str] | None":
    """Returns (rule_id, reason) if `command` should be denied, else None (allow)."""
    if is_broad_search(command):
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
    try:
        # O_EXCL makes create-if-absent atomic, closing the check-then-create race that let
        # two concurrent Bash calls both see "not fired yet" and both deny.
        fd = os.open(marker, os.O_CREAT | os.O_EXCL | os.O_WRONLY)
        os.close(fd)
        return False
    except FileExistsError:
        return True
    except OSError:
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

        tool_input = payload.get("tool_input", {})
        command = tool_input.get("command")
        session_id = payload.get("session_id")
        cwd = payload.get("cwd")
        if not command or not session_id or not cwd:
            return 0

        if tool_input.get("run_in_background"):
            return 0

        finding = classify(command)
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
