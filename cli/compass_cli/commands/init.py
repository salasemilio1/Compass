"""`compass init` -- deterministic scaffolding of .compass/ into a repo.

Does only the mechanical part: create the category directories, seed INDEX.md from the
bundled template, wire .gitignore, and stamp AGENTS.md/CLAUDE.md. Content that requires
reading and understanding the repo (INDEX.md's project description, context/overview.md)
is deliberately left to the agent running afterward -- this command never fabricates
project context, it only lays out the structure.
"""
from datetime import date
from importlib import resources
from pathlib import Path

import click

CATEGORIES = ["context", "architecture", "decisions", "experiments", "logs", "learnings", "scratch"]

# Keep in sync with SPEC.md's "Spec version:" line -- stamped into AGENTS.md so
# scripts/compass-check.sh can warn when a scaffolded repo drifts behind the spec.
SPEC_VERSION = 1

MODE_LINE = "<languages/frameworks>   ·   **Mode:** committed | branch-local   ·   **Updated:** <YYYY-MM-DD>"

AGENTS_MD_BLOCK = """<!-- compass-spec-version: {version} -->
# Agent instructions

This repo uses the compass protocol for agent memory. Before substantial work:
1. Read `.compass/INDEX.md` -- the retrieval map.
2. Read `.compass/context/current-state.md` -- where the work stands and what's next.

At task end, run the wrap-up workflow (git-diff-first, human-reviewed) to capture knowledge back
into `.compass/`. Never write `.compass/` silently.
"""

CLAUDE_MD_POINTER = "This repo uses the compass protocol. Read `.compass/INDEX.md` first.\n"


@click.command()
@click.option(
    "--mode",
    type=click.Choice(["committed", "branch-local"]),
    default=None,
    help="Visibility mode. Prompted interactively if omitted.",
)
@click.option(
    "--path",
    "target",
    type=click.Path(file_okay=False, path_type=Path),
    default=Path("."),
    help="Repo root to scaffold into (default: current directory).",
)
def init(mode, target):
    """Scaffold .compass/ into a repo."""
    target = target.resolve()
    compass_dir = target / ".compass"

    if compass_dir.exists():
        click.echo(f"error: {compass_dir} already exists -- refusing to overwrite.", err=True)
        raise SystemExit(1)

    if mode is None:
        mode = click.prompt(
            "Visibility mode ('committed' for repos you own, 'branch-local' for inherited/guest repos)",
            type=click.Choice(["committed", "branch-local"]),
            default="committed",
        )

    _scaffold_directories(compass_dir)
    _write_index(compass_dir, target.name, mode)
    _wire_gitignore(target, mode)
    _write_agents_md(target)
    _wire_claude_md(target)

    click.echo(f"compass initialized at {compass_dir} (mode: {mode})")
    click.echo(
        "Next: ask your agent to seed INDEX.md's project description and "
        "context/overview.md from the README -- compass init only lays out structure."
    )


def _scaffold_directories(compass_dir: Path) -> None:
    for category in CATEGORIES:
        (compass_dir / category).mkdir(parents=True, exist_ok=True)


def _write_index(compass_dir: Path, project_name: str, mode: str) -> None:
    template_text = resources.files("compass_cli.template").joinpath("INDEX.md").read_text()
    today = date.today().isoformat()
    text = template_text.replace("<PROJECT_NAME>", project_name)
    text = text.replace(
        MODE_LINE,
        f"<languages/frameworks>   ·   **Mode:** {mode}   ·   **Updated:** {today}",
    )
    (compass_dir / "INDEX.md").write_text(text)


def _wire_gitignore(target: Path, mode: str) -> None:
    gitignore = target / ".gitignore"
    line = ".compass/" if mode == "branch-local" else ".compass/scratch/"
    existing = gitignore.read_text() if gitignore.exists() else ""
    if line in existing.splitlines():
        return
    with gitignore.open("a") as f:
        if existing and not existing.endswith("\n"):
            f.write("\n")
        f.write(line + "\n")


def _write_agents_md(target: Path) -> None:
    agents_md = target / "AGENTS.md"
    block = AGENTS_MD_BLOCK.format(version=SPEC_VERSION)
    if agents_md.exists():
        existing = agents_md.read_text()
        if "compass-spec-version" in existing:
            return
        with agents_md.open("a") as f:
            f.write("\n" + block)
    else:
        agents_md.write_text(block)


def _wire_claude_md(target: Path) -> None:
    claude_md = target / "CLAUDE.md"
    if claude_md.exists():
        existing = claude_md.read_text()
        if ".compass/INDEX.md" in existing:
            return
        with claude_md.open("a") as f:
            f.write("\n" + CLAUDE_MD_POINTER)
    else:
        claude_md.write_text("# CLAUDE.md\n\n" + CLAUDE_MD_POINTER)
