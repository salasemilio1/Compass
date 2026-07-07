#!/usr/bin/env python3
"""
Stop hook: nudge toward /wrap-up when context usage crosses a token threshold.

Fires once per finished assistant turn. Reads transcript_path straight off the hook's
stdin payload (Claude Code hands it over -- no path-guessing needed) and reuses
tools/session-context/session_status.py for the actual measurement, so there is exactly
one place that knows how to read a transcript's usage data.

Nudges once per threshold crossing per session (tracked via a marker file in the temp
directory), not on every subsequent Stop -- otherwise every turn past the threshold would
re-nudge.
"""
import json
import os
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "tools" / "session-context"))
from session_status import compute_status, load_assistant_turns  # noqa: E402

THRESHOLD_PERCENT = int(os.environ.get("COMPASS_CONTEXT_NUDGE_THRESHOLD", 75))
CONTEXT_WINDOW = int(os.environ.get("COMPASS_CONTEXT_WINDOW", 200_000))


def main():
    try:
        hook_input = json.load(sys.stdin)
    except (json.JSONDecodeError, ValueError):
        return

    session_id = hook_input.get("session_id")
    transcript_path = hook_input.get("transcript_path")
    if not session_id or not transcript_path or not Path(transcript_path).is_file():
        return

    turns = load_assistant_turns(transcript_path)
    status = compute_status(turns, CONTEXT_WINDOW)
    if "error" in status:
        return

    percent_used = status["raw"]["percent_used"]
    if percent_used < THRESHOLD_PERCENT:
        return

    marker_file = Path(tempfile.gettempdir()) / f"claude-context-nudge-{session_id}"
    if marker_file.exists():
        return
    marker_file.write_text(str(percent_used))

    used = status["raw"]["used_tokens"]
    print(json.dumps({
        "systemMessage": (
            f"Context usage is at {percent_used}% ({used:,}/{CONTEXT_WINDOW:,} tokens) -- "
            "worth a /wrap-up, then /clear and /warm-up to keep going fresh?"
        )
    }))


if __name__ == "__main__":
    main()
