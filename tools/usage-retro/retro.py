#!/usr/bin/env python3
"""Measure how a project's Claude Code sessions spent tokens and used .compass/.

Reads the transcripts Claude Code keeps under ~/.claude/projects/<slug>/ (main sessions as
<id>.jsonl, subagents as <id>/subagents/agent-*.jsonl with a .meta.json beside each) and reports:

- estimated cost split between main sessions and subagents, and by token kind;
- main-session cost by context size band (how much is spent re-reading long sessions);
- subagent runs by task category and model;
- for each .compass/ category, how many files were never read by a session other than their writer.

Costs use list prices per million tokens and are relative estimates, not a bill. File reads are
counted from tool inputs that name the file, so reads through grep over a directory are missed and
"never re-read" is an upper bound.

Usage: python3 -I retro.py --repo /path/to/repo [--transcripts ~/.claude/projects/<slug>]
"""
import argparse
import collections
import glob
import json
import os
import re
import statistics
import subprocess

# (input, cache write, cache read, output) dollars per million tokens. Relative only.
PRICES = {"opus": (5, 6.25, 0.5, 25), "sonnet": (3, 3.75, 0.3, 15), "haiku": (1, 1.25, 0.1, 5)}
USAGE_KEYS = ["input_tokens", "cache_creation_input_tokens", "cache_read_input_tokens", "output_tokens"]
CATEGORIES = [
    ("break", r"break|adversar|hunt|attack"),
    ("critique", r"critiq|objection|fresh.context"),
    ("implement", r"implement|wire|add |write |fix|edit|refactor|migrat|build"),
    ("verify", r"verify|check|confirm|test|audit"),
    ("lookup", r"find|locate|look ?up|list|where|explore|survey|map|read|research|investigat|measure"),
]
BANDS = [(100_000, "<100k"), (300_000, "100-300k"), (600_000, "300-600k"), (float("inf"), ">600k")]


def model_family(model):
    return next((family for family in PRICES if family in (model or "")), None)


def category(description):
    text = description.lower()
    return next((name for name, pattern in CATEGORIES if re.search(pattern, text)), "other")


def transcript_events(path):
    with open(path, errors="ignore") as handle:
        for line in handle:
            try:
                yield json.loads(line)
            except json.JSONDecodeError:
                continue


def scan(path, compass_names):
    """One pass over a transcript: cost by token kind, cost by context band, compass files named."""
    seen_messages = set()
    cost_by_kind = collections.Counter()
    cost_by_band = collections.Counter()
    turns_by_band = collections.Counter()
    families = collections.Counter()
    reads, writes = set(), set()
    for event in transcript_events(path):
        if event.get("type") != "assistant":
            continue
        message = event["message"]
        family = model_family(message.get("model"))
        if family and message.get("id") not in seen_messages:
            seen_messages.add(message.get("id"))
            usage = message.get("usage") or {}
            turn_cost = 0.0
            for index, key in enumerate(USAGE_KEYS):
                cost = (usage.get(key) or 0) * PRICES[family][index] / 1e6
                cost_by_kind[key] += cost
                turn_cost += cost
            context = sum(usage.get(key) or 0 for key in USAGE_KEYS[:3])
            band = next(label for limit, label in BANDS if context < limit)
            cost_by_band[band] += turn_cost
            turns_by_band[band] += 1
            families[family] += 1
        for block in message.get("content") or []:
            if block.get("type") != "tool_use":
                continue
            text = json.dumps(block.get("input"))
            for name in re.findall(r"([\w.-]+\.md)", text):
                if name in compass_names:
                    (writes if block["name"] in ("Write", "Edit") else reads).add(name)
    return {
        "cost_by_kind": cost_by_kind,
        "cost_by_band": cost_by_band,
        "turns_by_band": turns_by_band,
        "family": families.most_common(1)[0][0] if families else None,
        "reads": reads,
        "writes": writes,
    }


def compass_files(repo):
    listing = subprocess.run(
        ["git", "-C", repo, "ls-files", ".compass"], capture_output=True, text=True, check=True
    ).stdout.split()
    return {os.path.basename(path): path.split("/")[1] for path in listing if path.count("/") >= 2}


def default_transcripts(repo):
    slug = re.sub(r"[^A-Za-z0-9]", "-", os.path.abspath(repo))
    return os.path.expanduser(f"~/.claude/projects/{slug}")


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--repo", required=True)
    parser.add_argument("--transcripts")
    arguments = parser.parse_args()
    root = arguments.transcripts or default_transcripts(arguments.repo)
    names = compass_files(arguments.repo)

    readers, writers = collections.defaultdict(set), collections.defaultdict(set)
    main_kind, main_band, main_turns = collections.Counter(), collections.Counter(), collections.Counter()
    main_cost = 0.0
    for path in glob.glob(f"{root}/*.jsonl"):
        result = scan(path, names)
        main_kind.update(result["cost_by_kind"])
        main_band.update(result["cost_by_band"])
        main_turns.update(result["turns_by_band"])
        main_cost += sum(result["cost_by_kind"].values())
        session = os.path.basename(path)
        for name in result["reads"]:
            readers[name].add(session)
        for name in result["writes"]:
            writers[name].add(session)

    subagent_cells = collections.defaultdict(list)
    for path in glob.glob(f"{root}/*/subagents/*.jsonl"):
        meta_path = path.replace(".jsonl", ".meta.json")
        meta = json.load(open(meta_path)) if os.path.exists(meta_path) else {}
        result = scan(path, names)
        session = path.split("/")[-3] + ".jsonl"
        for name in result["reads"]:
            readers[name].add(session)
        for name in result["writes"]:
            writers[name].add(session)
        cell = (category(meta.get("description", "")), result["family"] or "unknown")
        subagent_cells[cell].append(sum(result["cost_by_kind"].values()))

    subagent_cost = sum(sum(costs) for costs in subagent_cells.values())
    total = main_cost + subagent_cost or 1
    print(f"estimated cost: main ${main_cost:,.0f} ({100 * main_cost / total:.0f}%), "
          f"subagents ${subagent_cost:,.0f} ({100 * subagent_cost / total:.0f}%)")
    print("main cost by token kind:", {key: f"{100 * value / (main_cost or 1):.0f}%" for key, value in main_kind.items()})
    print("\nmain cost by context size at the turn")
    for _, band in BANDS:
        share = 100 * main_band[band] / (main_cost or 1)
        per_turn = main_band[band] / max(main_turns[band], 1)
        print(f"  {band:9} turns={main_turns[band]:6}  share={share:3.0f}%  $/turn={per_turn:.3f}")
    print("\nsubagents: category model runs median$ total$")
    for (name, family), costs in sorted(subagent_cells.items()):
        print(f"  {name:9} {family:8} {len(costs):4} {statistics.median(costs):7.2f} {sum(costs):7.0f}")
    print("\n.compass files never named by a session other than their writer")
    by_type = collections.defaultdict(lambda: [0, 0])
    for name, kind in names.items():
        if name in ("INDEX.md", "current-state.md", "archive.md"):
            continue
        by_type[kind][0] += 1
        by_type[kind][1] += not (readers[name] - writers[name])
    for kind, (count, unread) in sorted(by_type.items()):
        print(f"  {kind:14} files={count:4}  never re-read={unread:4} ({100 * unread / count:.0f}%)")


if __name__ == "__main__":
    main()
