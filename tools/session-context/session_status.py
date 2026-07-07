#!/usr/bin/env python3
"""
Pure measurement of context-window usage for the current Claude Code session.

Reads the session's transcript JSONL and reports the exact token usage the API already
billed for the latest turn (input + cache_creation + cache_read tokens) -- no tokenization,
no calibration, no confidence score. That number is ground truth, not an estimate.

Deliberately has no policy: it returns raw + derived metrics only. What to do with them
(e.g. nudge toward /wrap-up at 75% used) belongs to a caller such as
hooks/session-context-nudge.py, not this module.

Usage: session_status.py [--transcript PATH] [--context-window N]
Exit 0 with a JSON error object if the transcript can't be found or has no usable turns --
this is a diagnostic tool, not a gate.
"""

import argparse
import json
import os
import re
import sys
from pathlib import Path

DEFAULT_CONTEXT_WINDOW = 200_000

# How many of the most recent turns to use for the growth-rate estimate. Early turns in a
# session (warm-up reads, skill loads) are disproportionately large one-off costs; averaging
# over the whole session understates near-term risk right after such a spike.
TRAILING_WINDOW = 5


def resolve_transcript_path() -> "Path | None":
    """Reconstruct the current session's transcript path from environment variables.

    Only needed for a standalone/manual invocation -- a hook receives transcript_path
    directly on stdin and should prefer that over calling this.
    """
    project_dir = os.environ.get("CLAUDE_PROJECT_DIR") or os.getcwd()
    session_id = os.environ.get("CLAUDE_CODE_SESSION_ID")
    if not project_dir or not session_id:
        return None

    # Claude Code's project-dir slug replaces every non-alphanumeric character with "-"
    # (observed: "/Users/emilio.kungfu/Dev/compass" -> "-Users-emilio-kungfu-Dev-compass").
    slug = re.sub(r"[^A-Za-z0-9]", "-", project_dir)
    path = Path.home() / ".claude" / "projects" / slug / f"{session_id}.jsonl"
    return path if path.is_file() else None


def load_assistant_turns(transcript_path: "str | Path") -> list[dict]:
    """Extract usage-bearing assistant turns, in order, excluding subagent (sidechain) turns.

    Sidechain turns are separate subagent conversations with their own context budget --
    summing them into the main agent's usage would overstate it.
    """
    turns = []
    with open(transcript_path) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            entry = json.loads(line)
            if entry.get("type") != "assistant" or entry.get("isSidechain"):
                continue
            usage = entry.get("message", {}).get("usage")
            if not usage:
                continue
            turns.append({
                "timestamp": entry.get("timestamp"),
                "input_tokens": usage.get("input_tokens", 0),
                "cache_creation_input_tokens": usage.get("cache_creation_input_tokens", 0),
                "cache_read_input_tokens": usage.get("cache_read_input_tokens", 0),
                "output_tokens": usage.get("output_tokens", 0),
            })
    return turns


def context_total(turn: dict) -> int:
    """Total context tokens the model saw for this turn's request."""
    return (
        turn["input_tokens"]
        + turn["cache_creation_input_tokens"]
        + turn["cache_read_input_tokens"]
    )


def compute_status(turns: list[dict], context_window: int = DEFAULT_CONTEXT_WINDOW) -> dict:
    if not turns:
        return {"error": "no assistant turns with usage data found"}

    totals = [context_total(t) for t in turns]
    used = totals[-1]
    remaining = context_window - used
    percent_used = round(100 * used / context_window, 1)

    trailing = totals[-(TRAILING_WINDOW + 1):]
    deltas = [trailing[i] - trailing[i - 1] for i in range(1, len(trailing))]
    avg_growth_per_turn = round(sum(deltas) / len(deltas), 1) if deltas else 0
    estimated_turns_remaining = (
        int(remaining / avg_growth_per_turn) if avg_growth_per_turn > 0 else None
    )

    return {
        "raw": {
            "used_tokens": used,
            "remaining_tokens": remaining,
            "percent_used": percent_used,
            "context_window": context_window,
        },
        "derived": {
            "turns_measured": len(turns),
            "avg_growth_per_turn_last_n": avg_growth_per_turn,
            "estimated_turns_remaining": estimated_turns_remaining,
        },
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--transcript", help="Path to the session's transcript JSONL")
    parser.add_argument(
        "--context-window", type=int, default=DEFAULT_CONTEXT_WINDOW,
        help=f"Model context window in tokens (default: {DEFAULT_CONTEXT_WINDOW})",
    )
    args = parser.parse_args()

    transcript_path = Path(args.transcript) if args.transcript else resolve_transcript_path()
    if not transcript_path or not transcript_path.is_file():
        print(json.dumps({"error": "could not resolve a session transcript path"}))
        return 0

    turns = load_assistant_turns(transcript_path)
    print(json.dumps(compute_status(turns, args.context_window), indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
