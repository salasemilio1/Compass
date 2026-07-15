from click.testing import CliRunner

from compass_cli.commands.init import CATEGORIES, SPEC_VERSION, _write_agents_md
from compass_cli.main import cli


def run_init(tmp_path, mode="committed"):
    runner = CliRunner()
    return runner.invoke(cli, ["init", "--mode", mode, "--path", str(tmp_path)])


def test_init_creates_category_directories(tmp_path):
    result = run_init(tmp_path)
    assert result.exit_code == 0
    for category in CATEGORIES:
        assert (tmp_path / ".compass" / category).is_dir()


def test_init_seeds_index_md(tmp_path):
    run_init(tmp_path)
    index = (tmp_path / ".compass" / "INDEX.md").read_text()
    assert tmp_path.name in index
    assert "<PROJECT_NAME>" not in index
    assert "**Mode:** committed" in index


def test_init_refuses_to_clobber_existing_compass(tmp_path):
    (tmp_path / ".compass").mkdir()
    result = run_init(tmp_path)
    assert result.exit_code != 0
    assert "already exists" in result.output


def test_init_committed_mode_only_ignores_scratch(tmp_path):
    run_init(tmp_path, mode="committed")
    lines = (tmp_path / ".gitignore").read_text().splitlines()
    assert ".compass/scratch/" in lines
    assert ".compass/" not in lines


def test_init_branch_local_mode_ignores_whole_compass_dir(tmp_path):
    run_init(tmp_path, mode="branch-local")
    lines = (tmp_path / ".gitignore").read_text().splitlines()
    assert ".compass/" in lines


def test_init_appends_to_existing_gitignore_without_duplicating(tmp_path):
    (tmp_path / ".gitignore").write_text("node_modules/\n")
    run_init(tmp_path)
    lines = (tmp_path / ".gitignore").read_text().splitlines()
    assert lines.count(".compass/scratch/") == 1
    assert "node_modules/" in lines


def test_init_creates_agents_md_with_spec_version(tmp_path):
    run_init(tmp_path)
    agents = (tmp_path / "AGENTS.md").read_text()
    assert f"compass-spec-version: {SPEC_VERSION}" in agents


def test_init_appends_to_existing_agents_md(tmp_path):
    (tmp_path / "AGENTS.md").write_text("# Existing agent notes\n")
    run_init(tmp_path)
    agents = (tmp_path / "AGENTS.md").read_text()
    assert "Existing agent notes" in agents
    assert "compass-spec-version" in agents


def test_write_agents_md_is_idempotent(tmp_path):
    _write_agents_md(tmp_path)
    _write_agents_md(tmp_path)
    content = (tmp_path / "AGENTS.md").read_text()
    assert content.count("compass-spec-version") == 1


def test_init_creates_claude_md_pointer(tmp_path):
    run_init(tmp_path)
    claude_md = (tmp_path / "CLAUDE.md").read_text()
    assert ".compass/INDEX.md" in claude_md


def test_init_appends_claude_md_pointer_without_duplicating(tmp_path):
    (tmp_path / "CLAUDE.md").write_text("# Existing instructions\n")
    run_init(tmp_path)
    claude_md = (tmp_path / "CLAUDE.md").read_text()
    assert "Existing instructions" in claude_md
    assert claude_md.count(".compass/INDEX.md") == 1


def test_init_prompts_for_mode_when_omitted(tmp_path):
    runner = CliRunner()
    result = runner.invoke(cli, ["init", "--path", str(tmp_path)], input="branch-local\n")
    assert result.exit_code == 0
    lines = (tmp_path / ".gitignore").read_text().splitlines()
    assert ".compass/" in lines
