#!/usr/bin/env python3
"""Append a hook command to a Claude Code settings.json, creating the file if needed.

Used by install.sh in place of jq (python3 is already a hard dependency of the hooks
themselves, so this avoids requiring a second JSON tool).

Usage: register_hook.py <settings_file> <event> <hook_command> [matcher]
`matcher` restricts which tool(s) the hook applies to (required by events like
PreToolUse; SessionStart/Stop have no matcher concept and omit it).
Exits 1 on any failure (missing args, unreadable/invalid existing JSON) so the caller
can surface a clear error instead of silently doing nothing.
"""
import json
import sys


def main() -> int:
    if len(sys.argv) not in (4, 5):
        print("usage: register_hook.py <settings_file> <event> <hook_command> [matcher]", file=sys.stderr)
        return 1

    settings_file, event, hook_command = sys.argv[1], sys.argv[2], sys.argv[3]
    matcher = sys.argv[4] if len(sys.argv) == 5 else None

    try:
        with open(settings_file) as f:
            data = json.load(f)
    except FileNotFoundError:
        data = {}
    except json.JSONDecodeError as e:
        print(f"invalid JSON in {settings_file}: {e}", file=sys.stderr)
        return 1

    entry = {"hooks": [{"type": "command", "command": hook_command}]}
    if matcher:
        entry["matcher"] = matcher

    data.setdefault("hooks", {})
    data["hooks"].setdefault(event, [])
    data["hooks"][event].append(entry)

    with open(settings_file, "w") as f:
        json.dump(data, f, indent=2)
        f.write("\n")

    return 0


if __name__ == "__main__":
    sys.exit(main())
