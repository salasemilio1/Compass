"""Field retro: how compass behaved in real Claude Code work on one repo over a window.

Reads the repo's Claude Code transcripts (~/.claude/projects/<slug>*, worktrees included) and its
git history. Deterministic; no model calls. Writes a markdown report to stdout and, with
--flagged, a JSONL of edits that touch something a live decision names, for the judge pass in the
protocol (experiments/2026-10-26-v3-field-week.md).

Usage: python3 -I retro.py --repo ~/Dev/Misc/personal-dashboard --since 2026-10-26 --until 2026-10-31 [--flagged out.jsonl] [--former-path OLD_PATH]
"""
from __future__ import annotations

import argparse
import json
import pathlib
import re
import statistics
import subprocess

EDIT_TOOLS = {"Edit", "Write", "MultiEdit", "NotebookEdit"}
# Code-like names only (camelCase, PascalCase with an inner capital, snake_case), so prose words
# shared by many decisions don't flag every edit.
IDENTIFIER = re.compile(r"\b(?:[a-z]+[A-Z][A-Za-z0-9]*|[A-Z][a-z0-9]+[A-Z][A-Za-z0-9]*|[a-z0-9]+_[a-z0-9_]+)\b")
MAX_DECISIONS_PER_NAME = 3
COMMON = {"return", "string", "struct", "func", "import", "package", "testing", "errors", "context", "default",
          "should", "because", "before", "after", "between", "without", "change", "changes", "record", "records"}
HUMAN_CATCH = re.compile(r"we decided|that'?s intentional|that was intentional|on purpose|why did you change|revert (that|this|it)|undo (that|this)", re.I)
CONTINUE = re.compile(r"^\s*(continue|resume|pick up|carry on|where were we|keep going|next step)", re.I)


def slug(path: pathlib.Path) -> str:
    return re.sub(r"[^A-Za-z0-9]", "-", str(path))


def live_decisions(repo: pathlib.Path) -> dict[str, str]:
    """Decision id -> text, for decisions not superseded by a later record."""
    folder = repo / ".compass" / "decisions"
    texts = {path.name[:4]: path.read_text(errors="replace") for path in sorted(folder.glob("*.md"))}
    superseded = set()
    for text in texts.values():
        block = re.search(r"^supersedes:\n((?:\s+- .*\n)+)", text, re.M)
        if block:
            superseded |= {re.sub(r"[^0-9]", "", line)[:4] for line in block.group(1).splitlines()}
    return {key: text for key, text in texts.items() if key not in superseded}


def decisions_naming(decisions: dict[str, str], path: str, snippet: str) -> list[str]:
    names = {word for word in IDENTIFIER.findall(snippet) if len(word) >= 6 and word.lower() not in COMMON}
    base = pathlib.Path(path).name
    hits = set()
    for name in names:
        naming = [key for key, text in decisions.items() if re.search(rf"\b{re.escape(name)}\b", text)]
        if len(naming) <= MAX_DECISIONS_PER_NAME:
            hits.update(naming)
    # A path match needs the directory too: bare basenames like main.go or types.go are everywhere.
    relative = "/".join(pathlib.Path(path).parts[-2:])
    hits.update(key for key, text in decisions.items() if relative in text)
    return sorted(hits)


def is_compass(text: str) -> bool:
    return ".compass" in text or re.search(r"(^|[\s;&|(])compass (resume|record|check|index|init|doctor)", text) is not None


def session_rows(path: pathlib.Path, decisions: dict[str, str]) -> dict | None:
    events = []
    for line in path.read_text(errors="replace").splitlines():
        try:
            events.append(json.loads(line))
        except json.JSONDecodeError:
            continue
    stamps = [event["timestamp"] for event in events if event.get("timestamp")]
    if not stamps:
        return None
    row = {"session": path.stem[:8], "start": min(stamps), "brief": False, "first_prompt": "", "resume": False,
           "searched_before_edit": False, "edits": 0, "state_rewritten": False, "commits": 0, "records": 0,
           "tokens": 0, "compass_io": 0, "all_io": 0, "flagged": [], "catches": [], "_compass_ids": set()}
    searched = False
    for event in events:
        # Only the SessionStart hook's own output counts: the AGENTS.md block that the instructions
        # attachment carries also says "compass resume" and would count every session.
        attachment = event.get("attachment") if event.get("type") == "attachment" else None
        if isinstance(attachment, dict) and attachment.get("hookEvent") == "SessionStart" and (
                "# compass resume" in json.dumps(attachment.get("content", "")) + str(attachment.get("stdout", ""))):
            row["brief"] = True
        message = event.get("message") or {}
        usage = message.get("usage") if event.get("type") == "assistant" else None
        if isinstance(usage, dict):
            row["tokens"] += sum(usage.get(key, 0) for key in ("input_tokens", "cache_read_input_tokens", "cache_creation_input_tokens", "output_tokens"))
        content = message.get("content")
        if event.get("type") == "user" and isinstance(content, str) and not event.get("isSidechain"):
            if not row["first_prompt"]:
                row["first_prompt"] = content.strip()[:120]
            if HUMAN_CATCH.search(content) and not content.lstrip().startswith("<") and "agent-message" not in content:
                row["catches"].append(content.strip()[:160])
        if not isinstance(content, list):
            continue
        for block in content:
            if block.get("type") == "tool_result":
                size = len(json.dumps(block.get("content", "")))
                row["all_io"] += size
                if block.get("tool_use_id") in row["_compass_ids"]:
                    row["compass_io"] += size
            if block.get("type") != "tool_use":
                continue
            name, args = block["name"], block.get("input", {})
            text = json.dumps(args)
            row["all_io"] += len(text)
            if is_compass(text):
                row["_compass_ids"].add(block.get("id"))
                row["compass_io"] += len(text)
            command = args.get("command", "") if name == "Bash" else ""
            if "compass resume" in command:
                row["resume"] = True
            # Agents often rewrite current-state with a shell heredoc or script, not Edit/Write.
            if "current-state.md" in command and re.search(r"write_text|\.write\(|sed -i|>>? *\S*current-state", command):
                row["state_rewritten"] = True
            if "compass record" in command:
                row["records"] += 1
            if "git commit" in command:
                row["commits"] += 1
            # A search of the records: rg/grep over .compass/, or opening a decision. Reading INDEX.md
            # or current-state (warm-up, resume) is orientation, not a search.
            if (name in {"Bash", "Grep"} and ".compass" in text and re.search(r"\b(rg|grep)\b|\"pattern\"", text)) or (
                    name == "Read" and "/.compass/decisions/" in text):
                searched = True
            if name in EDIT_TOOLS:
                target = args.get("file_path", "")
                if target.endswith("current-state.md"):
                    row["state_rewritten"] = True
                    continue
                if "/.compass/" in target:
                    continue
                row["edits"] += 1
                if row["edits"] == 1:
                    row["searched_before_edit"] = searched
                snippet = args.get("old_string", "") + " " + args.get("content", "")[:2000]
                hits = decisions_naming(decisions, target, snippet)
                if hits:
                    row["flagged"].append({"session": row["session"], "file": target, "decisions": hits,
                                           "searched_before": searched, "old": args.get("old_string", "")[:800],
                                           "new": (args.get("new_string") or args.get("content", ""))[:800]})
    row.pop("_compass_ids", None)
    return row


def git(repo: pathlib.Path, *args: str) -> str:
    return subprocess.run(["git", "-C", str(repo), *args], capture_output=True, text=True).stdout


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", required=True)
    parser.add_argument("--since", required=True)
    parser.add_argument("--until", required=True)
    parser.add_argument("--flagged")
    # A repo that moved keeps its older transcripts under the old path's slug.
    parser.add_argument("--former-path", action="append", default=[])
    options = parser.parse_args()
    repo = pathlib.Path(options.repo).expanduser().resolve()
    decisions = live_decisions(repo)
    projects = pathlib.Path("~/.claude/projects").expanduser()
    paths = [repo] + [pathlib.Path(path).expanduser() for path in options.former_path]
    files = [path for root in paths for folder in projects.glob(slug(root) + "*") for path in folder.glob("*.jsonl")]
    rows = [row for row in (session_rows(path, decisions) for path in files)
            if row and options.since <= row["start"][:10] < options.until]
    # A session copied when a repo moves shows up under both slugs; keep one.
    rows = list({(row["start"], row["first_prompt"]): row for row in rows}.values())
    rows.sort(key=lambda row: row["start"])

    edited = [row for row in rows if row["edits"]]
    continuing = [row for row in rows if CONTINUE.search(row["first_prompt"])]
    committed = [row for row in rows if row["commits"]]
    flagged = [item for row in rows for item in row["flagged"]]
    shares = [row["compass_io"] / row["all_io"] for row in rows if row["all_io"]]
    pct = lambda part, whole: f"{len(part)}/{len(whole)}" + (f" ({100 * len(part) // len(whole)}%)" if whole else "")

    print(f"# Field retro: {repo.name}, {options.since} to {options.until}\n")
    print(f"Main sessions: {len(rows)} ({len(edited)} edited code). Live decisions: {len(decisions)}.\n")
    print("| Metric | Value | Bar |\n|---|---|---|")
    print(f"| Brief injected at start | {pct([r for r in rows if r['brief']], rows)} | tracked |")
    print(f"| Continue-style sessions that oriented (ran `compass resume` or got the brief) before the first edit | {pct([r for r in continuing if r['resume'] or r['brief']], continuing)} | ≥ 80% |")
    print(f"| Sessions with a commit that rewrote current-state | {pct([r for r in committed if r['state_rewritten']], committed)} | ≥ 90% |")
    print(f"| Editing sessions that searched the records before the first edit | {pct([r for r in edited if r['searched_before_edit']], edited)} | tracked |")
    print(f"| Flagged edits (touch something a live decision names) searched first | {pct([f for f in flagged if f['searched_before']], flagged)} | ≥ 80% |")
    print(f"| Compass share of tool I/O, median per session | {statistics.median(shares) * 100:.1f}% | ≤ 10% |" if shares else "| Compass share of tool I/O | n/a | ≤ 10% |")
    print(f"| `compass record` calls | {sum(r['records'] for r in rows)} | read by hand |")
    print(f"| Human catches (\"we decided\", \"why did you change\"…) | {sum(len(r['catches']) for r in rows)} | each read by hand |\n")

    added = git(repo, "log", f"--since={options.since}", f"--until={options.until}", "--diff-filter=A", "--name-only", "--format=", "--", ".compass/decisions")
    print(f"Decision records added in the window: {len(added.split())}\n")
    print("## Sessions\n| Start | Session | First prompt | Brief | Resume | Searched first | Edits | Commits | State rewritten | Flagged | Tokens processed |\n|---|---|---|---|---|---|---|---|---|---|---|")
    for r in rows:
        prompt = r["first_prompt"].replace("|", "/").replace("\n", " ")[:60]
        print(f"| {r['start'][:16]} | {r['session']} | {prompt} | {r['brief']} | {r['resume']} | {r['searched_before_edit']} | {r['edits']} | {r['commits']} | {r['state_rewritten']} | {len(r['flagged'])} | {r['tokens']:,} |")
    catches = [(r["session"], c) for r in rows for c in r["catches"]]
    if catches:
        print("\n## Human catches\n" + "\n".join(f"- {s}: {c}" for s, c in catches))
    if options.flagged:
        with open(options.flagged, "w") as out:
            for item in flagged:
                item["decision_text"] = {key: decisions[key][:3000] for key in item["decisions"][:3]}
                out.write(json.dumps(item) + "\n")


if __name__ == "__main__":
    main()
