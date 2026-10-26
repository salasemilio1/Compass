"""Migrate a v2 .compass/ to v3 records, the way a user would by hand: give every decision and
experiment a one-line `claim:` (from its existing description or its H1 title) and a `date:`,
move plans out of context/ into logs/plans/ (v3 context is standing context only), and give the
kept context files claims too.

Usage: python3 -I migrate.py <compass_dir> <standing-context-file>...
"""
import json
import pathlib
import re
import sys

CLAIM_MAX = 200
TITLE_PREFIX = re.compile(r"^(\d{4}\s*[—–-]+\s*|Decision\s+\d+:\s*|Experiment:\s*|Plan\s*[—–-]+\s*)", re.IGNORECASE)
BODY_DATE = re.compile(r"^(?:\*\*)?Date:(?:\*\*)?\s+(\d{4}-\d{2}-\d{2})", re.MULTILINE)
FILE_DATE = re.compile(r"^(\d{4}-\d{2}-\d{2})-")


def split(text: str) -> tuple[dict[str, str], str]:
    if not text.startswith("---\n"):
        return {}, text
    end = text.index("\n---\n", 4)
    fields = {}
    for line in text[4:end].splitlines():
        if line and not line.startswith((" ", "\t")) and ":" in line:
            key, value = line.split(":", 1)
            fields[key.strip()] = value.strip().strip('"')
    return fields, text[end + 5:]


def shorten(text: str) -> str:
    text = " ".join(text.split())
    if len(text) <= CLAIM_MAX:
        return text
    cut = text[: CLAIM_MAX - 1].rsplit(" ", 1)[0]
    return cut.rstrip(",;:—- ") + "…"


def migrate(path: pathlib.Path) -> None:
    fields, body = split(path.read_text())
    if "claim" in fields:
        return
    title = next((line[2:] for line in body.splitlines() if line.startswith("# ")), path.stem)
    claim = fields.get("description") or TITLE_PREFIX.sub("", title)
    date = BODY_DATE.search(body) or FILE_DATE.match(path.name)
    front = "---\nclaim: " + json.dumps(shorten(claim), ensure_ascii=False) + "\n"
    if date:
        front += "date: " + date.group(1) + "\n"
    path.write_text(front + "---\n" + body.lstrip("\n"))


def main(compass_dir: str, standing: list[str]) -> None:
    root = pathlib.Path(compass_dir)
    plans = root / "logs" / "plans"
    plans.mkdir(parents=True, exist_ok=True)
    for path in sorted((root / "context").glob("*.md")):
        if path.name == "current-state.md":
            continue
        if path.name in standing:
            migrate(path)
        else:
            path.rename(plans / path.name)
    for category in ("decisions", "experiments", "goals"):
        for path in sorted((root / category).glob("*.md")):
            migrate(path)


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2:])
