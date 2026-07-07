#!/usr/bin/env python3
"""
Pure measurement of per-tool context-token cost, attributed from real session transcripts.

For each tool_result in a transcript, attributes the token growth on the *next* assistant
turn to the tool(s) that produced it, then aggregates by tool name. Answers "does routing a
call through ctx_execute/graphify query actually cost fewer context tokens than the
equivalent Bash/Read/Grep call, in this repo's real sessions" -- see decisions/0004.

Deliberately has no policy: raw + aggregated numbers only, no threshold, no verdict.

Usage: tool_efficiency.py [--transcript PATH ...] [--all]
Exit 0 with a JSON error object if no usable transcript/turns are found.
"""

import argparse
import glob
import json
import os
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from session_status import context_total  # noqa: E402


def resolve_project_dir() -> "Path | None":
    project_dir = os.environ.get("CLAUDE_PROJECT_DIR") or os.getcwd()
    slug = re.sub(r"[^A-Za-z0-9]", "-", project_dir)
    path = Path.home() / ".claude" / "projects" / slug
    return path if path.is_dir() else None


def _result_size(content) -> int:
    """Byte size of a tool_result's content, whatever shape it takes (str or block list)."""
    if isinstance(content, str):
        return len(content.encode("utf-8"))
    return len(json.dumps(content).encode("utf-8"))


def load_tool_windows(transcript_path: "str | Path") -> list[dict]:
    """Walk a transcript and attribute each turn-over-turn token delta to the tool_result(s)
    that preceded it.

    A window closes at every assistant turn that carries `usage` (i.e. every main-chain
    assistant turn). Multiple tool_results can land in the same window when the model made
    parallel tool calls -- those are flagged low_confidence and split by result size, since
    the delta can't be cleanly divided among them.
    """
    tool_use_names: dict[str, str] = {}
    pending: list[dict] = []
    prev_total = None
    windows = []

    with open(transcript_path) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            entry = json.loads(line)
            if entry.get("isSidechain"):
                continue

            msg = entry.get("message", {})
            content = msg.get("content")
            if not isinstance(content, list):
                continue

            if entry.get("type") == "assistant":
                for block in content:
                    if block.get("type") == "tool_use":
                        tool_use_names[block.get("id")] = block.get("name", "unknown")

                usage = msg.get("usage")
                if usage:
                    total = context_total({
                        "input_tokens": usage.get("input_tokens", 0),
                        "cache_creation_input_tokens": usage.get("cache_creation_input_tokens", 0),
                        "cache_read_input_tokens": usage.get("cache_read_input_tokens", 0),
                    })
                    if prev_total is not None and pending:
                        windows.append({
                            "delta": total - prev_total,
                            "tools": pending,
                            "low_confidence": len(pending) > 1,
                        })
                    pending = []
                    prev_total = total

            elif entry.get("type") == "user":
                for block in content:
                    if block.get("type") == "tool_result":
                        name = tool_use_names.get(block.get("tool_use_id"), "unknown")
                        pending.append({"name": name, "size": _result_size(block.get("content"))})

    return windows


def aggregate(windows: list[dict]) -> dict:
    per_tool: dict[str, dict] = {}

    def bucket(name):
        return per_tool.setdefault(name, {
            "total_delta_tokens": 0, "attributed_calls": 0, "low_confidence_calls": 0,
        })

    for w in windows:
        tools = w["tools"]
        size_sum = sum(t["size"] for t in tools) or 1
        for t in tools:
            share = w["delta"] * (t["size"] / size_sum)
            b = bucket(t["name"])
            b["total_delta_tokens"] += share
            b["attributed_calls"] += 1
            if w["low_confidence"]:
                b["low_confidence_calls"] += 1

    for name, b in per_tool.items():
        b["total_delta_tokens"] = round(b["total_delta_tokens"])
        b["avg_delta_tokens_per_call"] = (
            round(b["total_delta_tokens"] / b["attributed_calls"]) if b["attributed_calls"] else 0
        )

    return dict(sorted(per_tool.items(), key=lambda kv: -kv[1]["total_delta_tokens"]))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--transcript", action="append", help="Transcript JSONL path (repeatable)")
    parser.add_argument(
        "--all", action="store_true",
        help="Scan every transcript in this project's ~/.claude/projects/<slug>/ directory",
    )
    args = parser.parse_args()

    if args.transcript:
        paths = [Path(p) for p in args.transcript]
    elif args.all:
        project_dir = resolve_project_dir()
        if not project_dir:
            print(json.dumps({"error": "could not resolve this project's transcript directory"}))
            return 0
        paths = sorted(Path(p) for p in glob.glob(str(project_dir / "*.jsonl")))
    else:
        print(json.dumps({"error": "specify --transcript PATH or --all"}))
        return 0

    all_windows = []
    for path in paths:
        if path.is_file():
            all_windows.extend(load_tool_windows(path))

    if not all_windows:
        print(json.dumps({"error": "no attributable tool windows found"}))
        return 0

    print(json.dumps({
        "sessions_scanned": len(paths),
        "windows_attributed": len(all_windows),
        "by_tool": aggregate(all_windows),
    }, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
